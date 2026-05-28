# Commit Breakdown — 4 Perspectives

Each commit explained from:
1. **Network engineering** — what problem is being solved at the systems level
2. **Clean code / design** — patterns, naming, structure
3. **Senior engineer lens** — good calls, bad calls, what's missing
4. **Code level** — what each line actually does in Go

---

# Commit 1 — `d3abd25` · "first commit"

One file changed: an **empty `README.md`**.

**1. Network level** — Nothing yet.

**2. Design** — An empty README signals intent: "this project will be documented." It's a placeholder, not content.

**3. Senior engineer** — The habit of initializing a repo before writing code is good. Committing an empty file is noise though — a README should have *something* in it before it's committed.

**4. Code** — No Go yet. Just `git init` → `git add README.md` → `git commit`.

---

# Commit 2 — `7d34062` · "accept and listen done"

Files added: Makefile, go.mod, main.go, p2p/transport.go, p2p/tcp_transport.go, p2p/handshake.go, p2p/tcp_transport_test.go.

---

## 1. Network engineering level

TCP is *connection-oriented*. Before any data flows, two parties must complete a three-way handshake (SYN → SYN-ACK → ACK). Your code starts AFTER that — `net.Listen` + `Accept()` is you saying "I'm open for business, give me the connection once the OS has done the TCP handshake."

```
Your code:   net.Listen(":3000")  →  listener.Accept()  →  net.Conn
OS handles:  SYN → SYN-ACK → ACK  (invisible to you)
```

Each `Accept()` returns a `net.Conn` — a **full-duplex pipe**: data can flow in both directions simultaneously, like a phone call.

The **goroutine-per-connection** model: for every new peer, you spawn `go t.handleConn(conn)`. That goroutine owns the connection and runs until the peer disconnects. This is how your server handles many peers at once without blocking.

`outbound bool` on `TCPPeer` marks who initiated: if YOU dial out = `outbound: true`. If THEY connect to you via `Accept()` = `outbound: false`.

---

## 2. Clean code / design

- **Interfaces before implementation** — `transport.go` defines `Peer` and `Transport` before any implementation exists. You're designing the *contract* first: "I don't care how you communicate, as long as you satisfy this interface." The TCP implementation is just one possibility.

- **Package separation** — all networking lives in `p2p/`. `main.go` just wires things up. Good boundary.

- **Constructor pattern** — `NewTCPPeer(conn, outbound)`, `NewTCPTransport(addr)`. Callers never set struct fields directly. This lets you add validation or defaults inside the constructor later without changing callers.

- **Makefile** — `make run`, `make test`, `make build`. Standardizes the dev workflow. Anyone cloning this repo knows exactly how to run it.

---

## 3. Senior engineer lens

**Bug — wrong `outbound` value:**
```go
// in handleConn — called from startAcceptLoop (meaning: THEY connected to US)
peer := NewTCPPeer(conn, true)  // ← should be false — this is an INBOUND connection
```
Connections received via `Accept()` are inbound. `outbound: true` means YOU dialed out. This is backwards.

**Premature fields — added then immediately removed:**
```go
type TCPTransport struct {
    mu    sync.RWMutex         // ← never used
    peers map[net.Addr]Peer    // ← never used
}
```
Both get deleted in the next commit. Senior rule: don't add fields until you have a caller for them.

**Broken test:**
```go
func TestTCPTransport(t *testing.T) {
    // ...
    select {}  // ← blocks forever. go test will hang and never pass.
}
```
Tests must return. This test is unusable as written.

**Wrong handshake abstraction:**
```go
type Handshaker interface {
    Handshake() error
}
```
An interface with one method. In Go, single-behavior abstractions belong as function types, not interfaces. Fixed in commit 3.

---

## 4. Code level

```go
net.Listen("tcp", ":3000")
// Binds a TCP socket to port 3000 on all network interfaces.
// Returns a net.Listener — the "listening ear".

t.listener.Accept()
// Blocks the goroutine until a client connects.
// Returns net.Conn — the live connection to that one client.

go t.startAcceptLoop()
// "go" keyword = launch a goroutine.
// Goroutines are lightweight (a few KB each), managed by the Go runtime.
// This one runs forever in the background, accepting new connections.

go t.handleConn(conn)
// Each connection gets its own goroutine — they run concurrently.

select {}
// An empty select blocks forever.
// Without this, main() would return and the process would exit,
// killing all goroutines. This keeps the server alive.
```

---

# Commit 3 — `39090e5` · "hard handshakes and error handling"

The design refactor. Added: encoding.go, message.go. Rewrote: handshake.go, tcp_transport.go, main.go.

---

## 1. Network engineering level

**Application-layer handshake** — separate from the TCP handshake the OS does. This is YOUR code verifying the peer: "Who are you? Are you authorized?" `NOPHandshakeFunc` does nothing (always passes) — it's a placeholder. Later you'd swap in a function that checks a token or certificate.

**Persistent connection + read loop** — once handshake passes, `handleConn` enters an infinite loop reading messages. This is the *streaming* model: the connection stays open and messages arrive continuously, unlike HTTP which opens/closes per request.

**RPC pattern** — `Message{Payload []byte}` is a *Remote Procedure Call* unit. One side sends a `Message`; the other receives it and acts on it. Everything on TCP is raw bytes — the `Decoder` is what gives those bytes meaning.

**Pluggable decoders:**
- `DefaultDecoder` — reads raw bytes, no structure. Good for simple binary data.
- `GOBDecoder` — uses Go's `encoding/gob` binary format. Typed, structured — you can encode a Go struct directly.

---

## 2. Clean code / design

**Options pattern:**
```go
// Before (commit 2):
NewTCPTransport(":3000")  // what if you need 5 config values? Ugly.

// After (commit 3):
TCPTransportOpts{
    ListenAddr:    ":3000",
    HandshakeFunc: NOPHandshakeFunc,
    Decoder:       DefaultDecoder{},
}
```
Clean, readable, and extensible — add a new option without breaking existing callers.

**Struct embedding:**
```go
type TCPTransport struct {
    TCPTransportOpts   // embedded — no field name
    listener net.Listener
}
// Now: t.ListenAddr, t.HandshakeFunc, t.Decoder all work directly
// even though they live inside TCPTransportOpts
```

**Dependency injection** — `HandshakeFunc` and `Decoder` are *passed in*, not hardcoded. `TCPTransport` doesn't know or care which implementation it gets. This is what makes code testable: you can inject a mock decoder in tests.

**Function type over single-method interface:**
```go
// Old (commit 2):
type Handshaker interface { Handshake() error }

// New (commit 3):
type HandshakeFunc func(any) error
```
In Go, if an abstraction has exactly one behavior, make it a function type. Standard library examples: `http.HandlerFunc`, `filepath.WalkFunc`. You can pass any matching function directly — no struct needed.

---

## 3. Senior engineer lens

**Good call — function type:**
The `HandshakeFunc` redesign is idiomatic Go. Recognizing this pattern early is valuable.

**Concern — `any` loses type safety:**
```go
type HandshakeFunc func(any) error
// Called as:
t.HandshakeFunc(peer)  // peer is *TCPPeer, passed as any (interface{})
```
The compiler accepts anything here. Should be `func(Peer) error` — then the compiler enforces you can only pass a `Peer`.

**Dead code committed:**
```go
type Temp struct{}  // serves no purpose, shouldn't be here
```

**Commented-out code is fine during learning — but understand what it means:**
```go
// rpcch    chan RPC
```
This channel is the *next architectural step*. Right now `handleConn` prints messages directly. The proper design pushes messages into this channel so the `FileServer` (not yet built) can consume them. This is the separation between "transport layer" and "application layer."

---

## 4. Code level

```go
// Struct embedding — TCPTransport gets all fields of TCPTransportOpts for free
type TCPTransport struct {
    TCPTransportOpts        // promoted fields: ListenAddr, HandshakeFunc, Decoder, OnPeer
    listener net.Listener
}

// msg is a pointer — Decode() modifies it in place
msg := &Message{}
for {
    if err := t.Decoder.Decode(conn, msg); err != nil {
        // err here = connection closed (io.EOF) or network error
        // return exits the goroutine — cleans up naturally
        return
    }
    fmt.Printf("%+v\n", msg)  // %+v = print with field names: &{From:... Payload:[...]}
}

// DefaultDecoder internals
buf := make([]byte, 1080)   // allocate 1080 bytes on the heap
n, err := r.Read(buf)       // Read() blocks until data arrives; n = bytes actually read
msg.Payload = buf[:n]       // slice — only the real data, not the whole 1080-byte buffer
```

---

# Commit 4 — `ed52714` · "Custom decoder for TCP transport"

Two lines changed across two files: uncommenting `From net.Addr` in Message, and assigning it in the read loop.

---

## 1. Network engineering level

Every message now records its sender: `From net.Addr` = the remote peer's IP:port (e.g., `192.168.1.5:54321`). Without this, when the FileServer receives a message, it has no idea who sent it and can't route a response back. This is **message routing metadata** — standard in any message-passing system.

---

## 2. Clean code / design

Small, focused commit — one logical change. This is what good commit hygiene looks like: one reason to change, one commit. `net.Addr` is an interface, so `Message` doesn't hard-code TCP — a UDP peer's address would also satisfy `net.Addr`.

---

## 3. Senior engineer lens

**Misleading commit message:** "Custom decoder for TCP transport" — nothing about the decoder changed. The actual change is "track message sender address." Commit messages matter: your future self will `git log` to understand what changed and why.

**Minor inefficiency:**
```go
for {
    // ...
    msg.From = conn.RemoteAddr()  // runs every iteration
}
```
`RemoteAddr()` returns the same value every time — it doesn't change while the connection is open. Set it once before the loop.

**Why this matters architecturally:** Once `rpcch` is implemented, the `FileServer` receives `Message` objects off the channel. It needs `msg.From` to know which peer asked for a file, so it can stream the response back to them specifically.

---

## 4. Code level

```go
type Message struct {
    From    net.Addr  // interface — actual runtime type is *net.TCPAddr
    Payload []byte
}

// net.Addr interface has two methods:
//   Network() string  →  "tcp"
//   String() string   →  "192.168.1.5:54321"

msg.From = conn.RemoteAddr()
// conn.RemoteAddr() = the OTHER end of the connection
// conn.LocalAddr()  = your own end (":3000")
```

---

# Where This All Points

These 4 commits built the **transport layer** — the lowest layer of the system:

```
[Application layer]   FileServer        ← not built yet
                           ↕  (via rpcch channel — next step)
[Transport layer]    TCPTransport       ← what you built
                           ↕  (raw TCP connections)
[Network layer]      OS / net package   ← Go standard library handles this
```

The next step is wiring up `rpcch` so `TCPTransport` stops printing and starts delivering messages to the `FileServer`. That's when the two layers connect.
