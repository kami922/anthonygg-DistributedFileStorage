# Q: Explain message.go

---

# `message.go` — 4 Perspectives

---

## 1. Network Engineering

On a network, **everything is bytes**. When Node A sends a file chunk to Node B, what actually travels down the wire is a raw stream of bytes — `01001101 01100101...`. There's no concept of "this is a filename" or "this is file data" at the TCP level. TCP just delivers bytes in order.

`Message` is your answer to: **"how do we give structure to those bytes?"**

```
Raw TCP stream:    [byte][byte][byte][byte][byte][byte]...
                              ↓ Decoder
Message:           { Payload: [byte][byte][byte][byte] }
```

Once bytes become a `Message`, your application code can work with them meaningfully — pass them to the FileServer, route them to the right handler, store them to disk.

`From net.Addr` (commented out) will eventually record who sent this message — essential for routing a reply back to the right peer. It's not wired up yet in this commit.

---

## 2. Clean Code / Design

`Message` is a **value object** — a plain container with no behaviour, no methods, just data. This is intentional. It separates concerns cleanly:

```
Message          → holds the data
Decoder          → creates a Message from raw bytes
FileServer       → decides what to do with a Message
```

Each piece has one job. `Message` doesn't know how it was created or what will be done with it. It's just the handoff between layers.

`[]byte` for `Payload` is the right choice — it's the universal container for raw data in Go. Whether the payload is a filename, a file chunk, or a command, the transport layer doesn't need to care. The layer above interprets it.

---

## 3. Senior Engineer Lens

**Six lines, but the design decision is significant.** By wrapping bytes in a named struct, you've given yourself room to grow:

```go
// Without Message — passing raw bytes everywhere
func handle(data []byte) {}

// With Message — can add fields without changing every caller
func handle(msg *Message) {}
// Later: msg.From, msg.Type, msg.Size, msg.ID — all addable without breaking anything
```

**`From` being commented out is a smell** — it was planned, left out, then added in commit 4. In production you'd either ship it or not — don't commit half-finished structs. During learning it's fine; just know it's a code smell.

**`[]byte` vs `io.Reader`** — storing the full payload as `[]byte` loads everything into memory at once. For large files this is expensive. Later in the project, streaming via `io.Reader` is more appropriate. For this stage of learning, `[]byte` is the right call — simpler.

---

## 4. Code Level

```go
type Message struct {
    // From    net.Addr  ← commented out — net.Addr is an interface from the "net" package
                         //   represents a network address (e.g. "192.168.1.5:3000")
                         //   not imported yet because it's commented out

    Payload []byte       // []byte = a slice of bytes — Go's raw binary container
                         // zero value = nil (not an empty slice, but nil)
                         // filled in by Decoder.Decode()
}
```

**How it flows through your code:**

```go
// In tcp_transport.go handleConn:
msg := &Message{}              // allocate empty Message on heap (pointer)
t.Decoder.Decode(conn, msg)    // Decoder reads bytes from conn, fills msg.Payload
fmt.Printf("%+v\n", msg)       // prints: &{Payload:[104 101 108 108 111]}
                               //          (those numbers are ASCII for "hello")
```

**`&Message{}`** — the `&` creates a pointer. `Decode()` needs a pointer so it can modify the original, not a copy. If you passed `Message{}` (no `&`), `Decode()` would fill a copy and the caller would see an empty message.

---

## What does Message hold and when?

It holds **anything one node sends to another node** over the TCP connection. In this distributed file system there are two kinds of things nodes will say to each other:

```
"Hey, store this file"     → payload = the file bytes
"Hey, send me that file"   → payload = the filename
```

Both travel as a `Message`. The payload is just bytes — the receiver decides what those bytes mean.

**Right now** (commit 3) — Message just gets printed:
```
Node A sends bytes → Message{Payload: bytes} → fmt.Printf
```

**Later** (when FileServer exists):
```
Node A: "store cat.jpg"   → Message → FileServer writes to disk, replicates to peers
Node A: "give me cat.jpg" → Message → FileServer finds file, streams back to Node A
```
