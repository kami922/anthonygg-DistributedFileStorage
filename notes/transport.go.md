# Q: Explain transport.go

---

# `transport.go` — 4 Perspectives

---

## 1. Network Engineering

This file defines the **blueprint** for the entire network layer — before a single implementation is written. Two concepts:

- **Peer** — any remote node in the network. Could be on the same machine, same city, or a different continent. Your code doesn't care — it just needs something that represents "the other side."
- **Transport** — the mechanism that connects peers. Could be TCP, UDP, WebSockets, or anything else. Your server logic shouldn't be glued to TCP specifically.

This is the foundation of a **protocol-agnostic** system. The rest of the codebase talks to `Transport` and `Peer`, not to `TCPTransport` or `TCPPeer` directly.

---

## 2. Clean Code / Design

This is **interface-first design** — you define the contract before the implementation. The pattern:

```
transport.go        → defines WHAT something must do   (the contract)
tcp_transport.go    → defines HOW TCP does it          (one implementation)
```

This means tomorrow you could write `UDPTransport` or `WebSocketTransport` and the rest of the system works without any changes. The server only ever talks to the `Transport` interface.

Both interfaces are **empty right now** — `interface{}`. This is intentional at this stage: you're staking out the concepts and naming them before you know exactly what methods they need. The methods get added in later commits.

---

## 3. Senior Engineer Lens

**Empty interfaces give zero compile-time safety:**
```go
type Peer interface {}
```
Right now anything satisfies `Peer` — a string, an int, a banana. The interface exists in name only. It won't enforce anything until methods are added.

**This is acceptable as scaffolding**, but in a real codebase you'd want at least the core methods defined before committing. By commit 3, `Peer` should have at minimum `Send()` and `CloseStream()`, and `Transport` should have `Dial()`, `ListenAndAccept()`, `Consume()`, and `Close()`.

**Naming is correct** — `Peer` and `Transport` are nouns, not verbs. Go interface names typically end in `-er` (`Reader`, `Writer`, `Decoder`) when they describe a single action. Multi-method interfaces like these use noun names. Good instinct.

---

## 4. Code Level

```go
type Peer interface {}
// An interface with zero methods.
// Every type in Go satisfies an empty interface — it's like saying "anything".
// Equivalent to "any" or "interface{}" which you'll see used interchangeably.

type Transport interface {}
// Same — empty contract.
// The comments describe the INTENT, not what the code enforces.
// The code enforces nothing yet.
```

The comments are the most valuable part of this file right now — they document WHY these concepts exist:

```go
// Transport is anything that can be used to send and receive messages
// between peers... it can be implemented using various protocols such as
// TCP, UDP, or even WebSockets.
```

That sentence IS the design decision. The code will catch up.
