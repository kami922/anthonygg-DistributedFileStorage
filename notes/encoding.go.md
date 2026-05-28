# Q: Explain encoding.go

---

# `encoding.go` — 4 Perspectives

---

## 1. Network Engineering

TCP delivers a raw **byte stream** — there are no boundaries, no labels, no structure. If Node A sends "hello" and then "world", Node B might receive "helloworld" as one chunk, or "hel" then "loworld" as two — TCP makes no guarantees about message boundaries.

A **Decoder** solves this: it reads from the raw stream and produces one structured `Message` at a time. It answers: **"where does one message end and the next begin?"**

```
Raw TCP stream:   [h][e][l][l][o][w][o][r][l][d]...
                              ↓ Decoder
Message 1:        { Payload: [h][e][l][l][o] }
Message 2:        { Payload: [w][o][r][l][d] }
```

Two decoders, two strategies:
- `DefaultDecoder` — reads whatever bytes are available right now (up to 1080)
- `GOBDecoder` — reads a fully structured Go binary object, handles boundaries automatically

---

## 2. Clean Code / Design

**Program to an interface, not an implementation:**

```
Decoder interface      → defines WHAT a decoder must do
DefaultDecoder struct  → HOW to do it with raw bytes
GOBDecoder struct      → HOW to do it with gob format
```

`TCPTransport` only ever calls `t.Decoder.Decode(...)` — it doesn't know or care which decoder it has. Swap `DefaultDecoder` for `GOBDecoder` in `main.go` and the transport works identically.

`io.Reader` as the first parameter is a key design choice. `io.Reader` is Go's universal "something you can read bytes from" interface. `net.Conn` satisfies it, but so does a file, a buffer, a compressed stream. The decoder is not tied to network connections specifically.

---

## 3. Senior Engineer Lens

**`DefaultDecoder` has a subtle problem — it doesn't handle message boundaries:**
```go
n, err := r.Read(buf)
msg.Payload = buf[:n]
```
`r.Read()` returns however many bytes happen to be available at that instant. If Node A sends a 5000-byte file chunk, `Read()` might return 1080 bytes the first call, 1080 the second, 2840 the third. Each call creates a separate `Message` with a fragment. The receiver has to reassemble — but nothing in this code does that.

`GOBDecoder` handles this correctly — `gob` is a self-delimiting format, it knows exactly how many bytes to read for one complete object.

**`DefaultDecoder` is fine for now** — you're testing with telnet which sends small text lines well under 1080 bytes. The problem surfaces with large files. Known limitation to revisit later.

**1080 bytes** — not a round number like 1024. `1024` (1KB) or `4096` (4KB) would be more conventional buffer sizes.

**`gob.NewDecoder(r).Decode(msg)`** — creates a new gob decoder on every call. Slightly wasteful — you could store the decoder and reuse it. Minor, not worth fixing at this stage.

---

## 4. Code Level

```go
// THE INTERFACE
type Decoder interface {
    Decode(io.Reader, *Message) error
    //     ^^^^^^^^^  ^^^^^^^^
    //     where to   where to put
    //     read from  the result
}
// *Message is a pointer — Decode fills it in-place
// io.Reader is an interface — net.Conn satisfies it automatically


// GOB DECODER
type GOBDecoder struct{}   // empty struct — no fields, just a method carrier

func (d GOBDecoder) Decode(r io.Reader, msg *Message) error {
    return gob.NewDecoder(r).Decode(msg)
    // gob = Go's built-in binary serialization format
    // Reads exactly one encoded object from r, writes it into msg
    // Handles message boundaries automatically
}


// DEFAULT DECODER
type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, msg *Message) error {
    buf := make([]byte, 1080)  // allocate a 1080-byte buffer on the heap
                                // allocated fresh every call

    n, err := r.Read(buf)      // Read() blocks until bytes arrive
                                // n = how many bytes actually came in
                                // n could be 1, could be 1080 — not guaranteed
    if err != nil {
        return err             // io.EOF when connection closes — normal exit
    }

    msg.Payload = buf[:n]      // slice down to actual bytes received
                                // NOT buf — otherwise you'd have garbage at the end
    return nil
}
```

**`buf[:n]` vs `buf` — why it matters:**
```
buf = [h][e][l][l][o][0][0][0][0]...  ← 1080 slots, only 5 used

buf[:5]  → [h][e][l][l][o]            ← correct
buf      → [h][e][l][l][o][0][0]...   ← 1080 bytes, 1075 of them garbage
```

**How both structs satisfy the Decoder interface:**
```go
// Go checks: does GOBDecoder have Decode(io.Reader, *Message) error? Yes ✓
// Does DefaultDecoder have Decode(io.Reader, *Message) error? Yes ✓
// So both can be assigned to a Decoder field:

Decoder: p2p.GOBDecoder{}      // works
Decoder: p2p.DefaultDecoder{}  // works
```
