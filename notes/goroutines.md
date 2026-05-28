# Q: What are goroutines?

---

# Goroutines — 4 Perspectives

---

## 1. Network Engineering

A server has one job: handle many connections at the same time. The problem is — each connection needs someone "sitting on it" waiting for data. If you only have one worker, everyone else waits in line.

```
Without goroutines:
Node A connects → you handle Node A → Node B waits → Node C waits → ...

With goroutines:
Node A connects → goroutine 1 handles Node A  ┐
Node B connects → goroutine 2 handles Node B  ├─ all running simultaneously
Node C connects → goroutine 3 handles Node C  ┘
```

In your distributed file system, each peer needs a persistent, dedicated connection. Goroutines make that possible without writing complex multi-threaded code.

---

## 2. Clean Code / Design

Goroutines are Go's answer to one of software's oldest problems: **how do you do multiple things at once without making your code a mess?**

Other languages use threads — heavy, expensive, hard to manage. Go gives you goroutines — lightweight, cheap, and built into the language with one keyword: `go`.

The design intent is: **launch it and forget it.** You write the function as if it runs alone. The `go` keyword is the only difference between sequential and concurrent.

```go
handleConn(conn)    // sequential — blocks everything until done
go handleConn(conn) // concurrent — runs independently, you move on
```

Same function. One word changes everything.

---

## 3. Senior Engineer Lens

**Goroutines are cheap but not free.**
- Starting cost: ~2–8 KB of stack memory each
- OS threads: ~1–2 MB each
- You can comfortably run 100,000 goroutines. You cannot run 100,000 OS threads.

**The Go scheduler runs goroutines on OS threads** — the runtime manages this for you. Multiple goroutines share a small pool of OS threads. When one goroutine blocks (e.g. waiting on `Accept()`), the scheduler runs another goroutine on that same thread. No CPU is wasted.

```
OS Thread 1: [goroutine A running] → A blocks on Accept() → [goroutine B runs] → ...
OS Thread 2: [goroutine C running] → C blocks on Read()   → [goroutine D runs] → ...
```

**The risk: goroutine leaks.** If a goroutine blocks forever and is never cleaned up, it stays in memory. In your current code, `handleConn` exits immediately — no leak. But in commit 3 where a `for` loop is added, if the connection never closes and no `return` is hit, that goroutine lives forever. This is why the error check does `return` not `continue`.

---

## 4. Code Level

```go
// WITHOUT "go" — sequential, blocks here until startAcceptLoop finishes
// (it never finishes — infinite loop — so the program would freeze here)
t.startAcceptLoop()
return nil  // never reached

// WITH "go" — concurrent, launches startAcceptLoop in a new goroutine
// returns immediately, program continues
go t.startAcceptLoop()
return nil  // reached instantly
```

In your tcp_transport.go there are exactly **two** goroutine launches:

```go
// Launch 1 — in ListenAndAccept()
go t.startAcceptLoop()
// Spawns one goroutine that runs forever accepting connections
// Main program gets control back immediately

// Launch 2 — in startAcceptLoop()
go t.handleConn(conn)
// Spawns one goroutine PER connection
// Accept loop gets control back immediately and waits for the next peer
```

**Visualized over time:**

```
main goroutine:       NewTCPTransport → ListenAndAccept → select{} (blocks forever)
                                              │
                                              └─ spawns:
acceptLoop goroutine:                    Accept()...Accept()...Accept()...
                                              │         │         │
                                              └─spawn   └─spawn   └─spawn
handleConn goroutines:                   peer1...  peer2...  peer3...
```

**A goroutine ends when its function returns.** That's it. No cleanup needed, no explicit kill. In your current code:

```go
func (t *TCPTransport) handleConn(conn net.Conn) {
    peer := NewTCPPeer(conn, true)
    fmt.Printf("new incomming connection %+v\n", peer)
    // function returns here → goroutine is gone
}
```

The goroutine lives for milliseconds right now. In commit 3 it grows a `for` loop and lives for the entire lifetime of the connection.
