# Q: Explain tcp_transport.go (commit 3 — big refactor)

---

# `tcp_transport.go` — 4 Perspectives

---

## 1. Network Engineering

This is the file where the entire network lifecycle lives — from opening a port to receiving a message. Every other file in commit 3 feeds into this one:

```
net.Listen()          → own a port
Accept()              → get a connection
HandshakeFunc(peer)   → verify the peer       ← handshake.go
Decoder.Decode()      → turn bytes → Message  ← encoding.go + message.go
```

The connection lifecycle for each peer:

```
1. TCP connects (OS handles SYN/SYN-ACK/ACK)
2. Handshake         → trusted? yes → continue. no → close.
3. Read loop         → sit on connection forever, decode each message as it arrives
4. Error / EOF       → peer disconnected → goroutine exits → connection cleaned up
```

This is a **persistent connection** model — the connection to each peer stays open for the lifetime of the session, not just one request like HTTP.

---

## 2. Clean Code / Design

**The biggest change from commit 2: the Options Pattern.**

```go
// Commit 2 — one hardcoded string, no flexibility
func NewTCPTransport(listenAddr string) *TCPTransport

// Commit 3 — a config struct, fully flexible
func NewTCPTransport(opts TCPTransportOpts) *TCPTransport
```

`TCPTransportOpts` holds all the things that can vary:

```go
type TCPTransportOpts struct {
    ListenAddr    string           // which port
    HandshakeFunc HandshakeFunc    // how to verify peers
    Decoder       Decoder          // how to parse messages
    OnPeer        func(Peer) error // what to do when a peer connects
}
```

Add a new config option tomorrow — add one field here. Every existing caller still compiles unchanged.

**Struct embedding:**
```go
type TCPTransport struct {
    TCPTransportOpts   // embedded — no field name
    listener net.Listener
}
```
All fields of `TCPTransportOpts` are promoted directly onto `TCPTransport`. Instead of `t.opts.ListenAddr` you write `t.ListenAddr`. Cleaner access, same structure.

**Dependency injection** — `HandshakeFunc` and `Decoder` are passed in from outside. `TCPTransport` never instantiates them itself. This means you can test with a mock decoder, use a real one in production, swap between them in config — all without touching this file.

---

## 3. Senior Engineer Lens

**`sync.RWMutex` and `peers` map removed — good.** Commit 2 added them without using them. Removing unused code is always the right call.

**`type Temp struct{}` on line 65 — dead code, should not be here.** Serves no purpose. Would be caught in code review.

**`OnPeer func(Peer) error` is defined but never called.** It's in `TCPTransportOpts` but `handleConn` never calls it after a peer connects. Forward placeholder — it'll matter when the FileServer needs to be notified a peer joined. For now unused, which is a mild smell.

**The `continue` bug from commit 2 is still here:**
```go
conn, err := t.listener.Accept()
if err != nil {
    fmt.Printf("TCP Accept Error:%s\n", err)
    // ← still missing: continue
}
go t.handleConn(conn)  // still runs with nil conn on error
```

**The commented-out `rpcch` is the most important thing in this file** — it's the next architectural step:
```go
// rpcch chan RPC  ← this channel connects transport to FileServer
```
Transport pushes messages in → FileServer pulls them out. Right now messages get printed instead. The comment is a signpost for where the code is going.

**`outbound: true` bug still present** — connections from `Accept()` should be `false`.

---

## 4. Code Level

```go
// OPTIONS STRUCT — plain config, no methods
type TCPTransportOpts struct {
    ListenAddr    string
    HandshakeFunc HandshakeFunc       // function type from handshake.go
    Decoder       Decoder             // interface from encoding.go
    OnPeer        func(Peer) error    // inline function type — no separate typedef
}

// STRUCT EMBEDDING
type TCPTransport struct {
    TCPTransportOpts          // all its fields promoted to TCPTransport
    listener net.Listener
    // rpcch chan RPC          // commented out — not yet implemented
}

// CONSTRUCTOR
func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
    return &TCPTransport{
        TCPTransportOpts: opts,  // assign the embedded struct by its type name
    }
}

// handleConn — the most important function
func (t *TCPTransport) handleConn(conn net.Conn) {
    peer := NewTCPPeer(conn, true)           // wrap connection in TCPPeer

    if err := t.HandshakeFunc(peer); err != nil {
        conn.Close()                         // reject the peer
        fmt.Printf("Handshake error: %s\n", err)
        return                               // exit goroutine
    }

    fmt.Printf("new incoming connection %+v\n", peer)

    msg := &Message{}                        // one Message, reused each iteration
    for {
        if err := t.Decoder.Decode(conn, msg); err != nil {
            fmt.Printf("TCP error: %s\n", err)
            return                           // connection closed → exit goroutine cleanly
        }
        // msg.From = conn.RemoteAddr()      // not wired up yet — comes in commit 4
        fmt.Printf("new message received %+v\n", msg)
        // future: t.rpcch <- *msg           // push to FileServer instead of printing
    }
}
```

**Read loop visualised over time:**
```
handleConn goroutine for Node A:

Decode()... blocked ...  Node A sends "hello"  →  msg.Payload = [hello]  → print
Decode()... blocked ...  Node A sends "world"  →  msg.Payload = [world]  → print
Decode()... blocked ...  Node A disconnects    →  err = io.EOF           → return
```

**How all commit 3 files connect:**
```
main.go
  └─ TCPTransportOpts{ HandshakeFunc: NOPHandshakeFunc, Decoder: DefaultDecoder{} }
  └─ NewTCPTransport(opts)
  └─ tr.ListenAndAccept()
          └─ go startAcceptLoop()
                  └─ Accept() → conn
                  └─ go handleConn(conn)
                          └─ NewTCPPeer(conn)          [tcp_transport.go]
                          └─ HandshakeFunc(peer)        [handshake.go]
                          └─ Decoder.Decode(conn, msg)  [encoding.go]
                          └─ Message{ Payload: bytes }  [message.go]
                          └─ fmt.Printf(msg)            ← temporary, replaced by rpcch later
```
