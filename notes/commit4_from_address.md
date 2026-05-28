# Q: Explain commit 4 — the 2 changes (From address)

Commit: ed52714 — "Custom decoder for TCP transport"

The entire commit is this diff:

```go
// message.go — before          // message.go — after
type Message struct {            type Message struct {
    // From    net.Addr              From    net.Addr      ← uncommented
    Payload []byte                  Payload []byte
}                                }

// tcp_transport.go — before    // tcp_transport.go — after
// msg.From = conn.RemoteAddr() msg.From = conn.RemoteAddr() ← uncommented
```

---

## 1. Network Engineering

Every message travelling across the network now carries a **return address**.

Think of it like a letter. Before this commit, messages arrived with no envelope — you had the content but no idea who sent it. After this commit, every message has a stamp: `From: 192.168.1.5:54321`.

This is **message routing metadata** — fundamental to any distributed system. Without it:

```
FileServer receives Message{ Payload: "give me cat.jpg" }
FileServer: "...who asked? I have no idea who to send the file back to."
```

With it:
```
FileServer receives Message{ From: 192.168.1.5:54321, Payload: "give me cat.jpg" }
FileServer: "Node at 192.168.1.5:54321 wants cat.jpg — stream it back to them."
```

`conn.RemoteAddr()` returns the **other end's** address — the peer's IP and port, not yours.

---

## 2. Clean Code / Design

Small, focused commit — one logical change, cleanly isolated. This is what good commit discipline looks like: one reason to change, one commit.

`net.Addr` is an **interface** — not hardcoded to TCP:

```go
From net.Addr
// net.Addr interface has two methods:
//   Network() string  →  "tcp"
//   String()  string  →  "192.168.1.5:54321"
```

A UDP peer's address would also satisfy `net.Addr`. The `Message` struct stays protocol-agnostic — it works regardless of what kind of transport sends it.

---

## 3. Senior Engineer Lens

**Misleading commit message.** "Custom decoder for TCP transport" — nothing about the decoder changed. The actual change is "track message sender address." Future you reading `git log` will be confused. Commit messages should match the change.

**Minor inefficiency — `RemoteAddr()` called every loop iteration:**
```go
for {
    t.Decoder.Decode(conn, msg)
    msg.From = conn.RemoteAddr()  // ← runs on every single message
}
```
`conn.RemoteAddr()` returns the same value for the entire lifetime of a connection — the peer's address doesn't change mid-session. The correct place is **before** the loop:

```go
msg := &Message{}
msg.From = conn.RemoteAddr()  // set once
for {
    t.Decoder.Decode(conn, msg)
    // From is already set, no need to reassign
}
```

**This is scaffolding for `rpcch`.** Once the channel is wired up, `FileServer` receives `Message` off the channel and uses `msg.From` to know which peer to respond to. Without `From`, the channel approach doesn't work — you'd have the message content but no routing information.

---

## 4. Code Level

```go
// message.go
import "net"
// "net" imported now because net.Addr is actually used
// Previously commented out so the import wasn't needed

type Message struct {
    From    net.Addr  // interface — runtime type will be *net.TCPAddr
                      // *net.TCPAddr has .IP and .Port fields if you need them
    Payload []byte
}


// tcp_transport.go — inside handleConn read loop
msg.From = conn.RemoteAddr()
// conn.RemoteAddr() → the OTHER side's address
// conn.LocalAddr()  → YOUR address (":3000")
// RemoteAddr() returns net.Addr interface — compatible with msg.From field

// What it looks like when printed:
fmt.Printf("new message received %+v\n", msg)
// Before commit 4: &{Payload:[104 101 108 108 111]}
// After  commit 4: &{From:192.168.1.5:54321 Payload:[104 101 108 108 111]}
```

---

## The Full Journey — All 4 Commits

```
Commit 1 (d3abd25)  Empty README         → repo exists
Commit 2 (7d34062)  Accept & Listen      → server can receive connections
Commit 3 (39090e5)  Handshake + Decode   → server can verify peers and read messages
Commit 4 (ed52714)  From address         → messages know who sent them

Next:               rpcch channel        → messages reach the FileServer
After that:         FileServer           → messages drive real file operations
```
