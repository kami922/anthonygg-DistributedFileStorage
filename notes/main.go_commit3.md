# Q: Explain main.go (commit 3)

---

# `main.go` — 4 Perspectives

---

## 1. Network Engineering

`main.go` is the **entry point** — the moment the program starts, this runs. Its only job at this stage is to spin up one node on port 3000 and keep it alive.

In a distributed system, this is the **bootstrap** step. Before a node can participate in the network — before it can store files, talk to peers, or replicate data — it needs to be reachable. Binding a port is step zero. Everything else comes after.

Right now you have one node. The final system will spin up multiple nodes (ports 3000, 7000, 5000) that discover and connect to each other. `main.go` will grow to wire all of that up.

---

## 2. Clean Code / Design

`main.go` is purely **configuration and wiring** — it creates things and connects them. Zero business logic. That's exactly right.

The options pattern pays off here — reading this file tells you everything about the server configuration at a glance:

```go
tcpOpts := p2p.TCPTransportOpts{
    ListenAddr:    ":3000",
    HandshakeFunc: p2p.NOPHandshakeFunc,
    Decoder:       p2p.DefaultDecoder{},
}
```

Three lines, full configuration visible. No need to look up function signatures.

**`log.Fatalf` vs `fmt.Printf`** — if the server can't bind its port, there's nothing to do — the program should die loudly. `log.Fatalf` prints the error and calls `os.Exit(1)`. `fmt.Printf` would just print and keep going into a broken state.

---

## 3. Senior Engineer Lens

**`select {}` is the correct way to block `main` forever** — idiomatic Go for "this program runs until killed." The alternative `for {}` would spin the CPU doing nothing. `select {}` sleeps with zero CPU usage.

**`OnPeer` is not set in opts** — unset function fields default to `nil` in Go. If `handleConn` ever called `t.OnPeer(peer)` without a nil check, it would panic. It doesn't call it yet — but it's a trap waiting to happen.

**Only one node** — the final `main.go` creates three servers and connects them. Right now this just proves the transport layer works in isolation. Good incremental approach.

**The import path** `github.com/kami922/anthonygg-DistributedFileStorage/p2p` — this is your Go module name from `go.mod`. Nothing to do with GitHub at runtime — it's just how Go identifies packages internally. You don't need to push to GitHub for this to work.

---

## 4. Code Level

```go
package main
// Every Go program has exactly one "main" package.
// The "main" package is the entry point — go build looks for it.

import (
    "log"   // for log.Fatalf — prints error + exits with code 1
    "github.com/kami922/anthonygg-DistributedFileStorage/p2p"
    // your own p2p package — everything in p2p/ folder accessible as p2p.Something
)

func main() {
    // main() is the function Go calls when the program starts.
    // When main() returns, the program exits — killing ALL goroutines.

    tcpOpts := p2p.TCPTransportOpts{
        ListenAddr:    ":3000",               // all interfaces, port 3000
        HandshakeFunc: p2p.NOPHandshakeFunc,  // the function itself (no parentheses)
        Decoder:       p2p.DefaultDecoder{},  // empty struct literal
        // OnPeer not set → nil
    }

    tr := p2p.NewTCPTransport(tcpOpts)   // creates TCPTransport, does NOT start it yet

    if err := tr.ListenAndAccept(); err != nil {
        log.Fatalf("Failed to start TCP transport: %v", err)
        // %v = default format for the error value
        // Fatalf = Printf + os.Exit(1)
    }
    // At this point: port 3000 is bound, accept loop running in goroutine

    select {}
    // Block forever — main() must not return or all goroutines die.
    // select with no cases = sleep forever, zero CPU cost.
}
```

**Execution timeline:**
```
go run main.go
    │
    ├─ tcpOpts created           (config only, nothing running yet)
    ├─ NewTCPTransport(tcpOpts)  (creates struct, nothing running yet)
    ├─ ListenAndAccept()
    │       ├─ net.Listen(":3000")    port 3000 now owned by this process
    │       └─ go startAcceptLoop()   goroutine launched, running in background
    │
    └─ select{}                  main goroutine sleeps forever
                                 accept loop runs forever in background
                                 handleConn goroutines spawn as peers connect
```
