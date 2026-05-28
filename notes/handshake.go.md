# Q: Explain handshake.go

---

# `handshake.go` — 4 Perspectives

---

## 1. Network Engineering

A handshake is a **verification exchange** before real communication begins. Think of it as:

```
Node A connects
Node B: "prove who you are"
Node A: "here is my token / certificate / secret"
Node B: "accepted" or "rejected — connection closed"
```

This is an **application-layer handshake** — separate from the TCP handshake the OS already did. TCP proves the connection is live. YOUR handshake proves the peer is trusted.

Right now this file just defines the concept. The actual verification logic comes later.

---

## 2. Clean Code / Design

```go
type Handshaker interface {
    Handshake() error
}
```

A single-method interface. The idea: any type that has a `Handshake() error` method can act as a handshaker. You could inject different handshake strategies — no-op for testing, token-based for production.

**However** — this is the wrong design for Go, and it gets replaced completely in commit 3. A single-method interface in Go should almost always be a **function type** instead. This is one of Go's key idioms, and the tutorial corrects it in the next commit.

---

## 3. Senior Engineer Lens

**Wrong abstraction for Go.** A one-method interface like this:
```go
type Handshaker interface {
    Handshake() error
}
```

Forces you to create a struct just to implement it:
```go
type MyHandshaker struct{}
func (h MyHandshaker) Handshake() error { return nil }
```

Go's idiomatic approach — a function type — is simpler:
```go
type HandshakeFunc func(Peer) error  // commit 3's version
```

Now you just pass any matching function directly. No struct needed. The standard library uses this pattern everywhere: `http.HandlerFunc`, `filepath.WalkFunc`. This is why commit 3 deletes this file's content entirely.

**Also note:** `Handshake() error` takes no arguments — it doesn't know WHO it's handshaking with. You'd need to store the peer on the struct. The function type approach fixes this by accepting `Peer` as a parameter.

---

## 4. Code Level

```go
type Handshaker interface {
    Handshake() error
}
// Defines an interface — any type with a Handshake() method satisfies it.
// error is Go's built-in error type — nil means success, non-nil means failure.
// No struct, no implementation here — just the shape.
```

This file is 9 lines and mostly whitespace. Its entire value is the concept it names. The implementation is elsewhere — and in this commit, it doesn't exist yet.
