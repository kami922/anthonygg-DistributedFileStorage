# Q: Explain tcp_transport_test.go

---

# `tcp_transport_test.go` — 4 Perspectives

---

## 1. Network Engineering

Tests at the network layer verify that your server can actually bind a port and accept the OS giving it a socket. This test checks:
1. Did the transport store the address correctly?
2. Did `ListenAndAccept()` succeed without error — meaning the OS granted port 4000?

Port 4000 is used instead of 3000 so the test doesn't clash with a running server on 3000.

---

## 2. Clean Code / Design

Go's testing convention: test files end in `_test.go`, live in the same package, and test functions start with `Test`. The `testify` library (`assert`) gives cleaner failure messages than raw `if` checks:

```go
assert.Equal(t, tr.listenAddress, listenAddr)
// On failure prints: "Expected :4000, got something else"
// vs raw Go:
if tr.listenAddress != listenAddr { t.Fatalf("...") }
```

The test is structured correctly — create, assert properties, assert behaviour. Good instinct.

---

## 3. Senior Engineer Lens

**This test is broken — it hangs forever:**
```go
select {}  // blocks forever — go test will never finish
```

`go test` runs each `Test*` function and waits for it to return. `select {}` never returns. Running `make test` on this commit hangs indefinitely.

A correct approach would either:
```go
// Option A: just don't block — test that ListenAndAccept doesn't error, done
assert.Nil(t, tr.ListenAndAccept())
// return here — test passes

// Option B: if you need to test incoming connections, use a timeout context
```

**`listenAddress` is unexported** (lowercase) — `tr.listenAddress` only works because the test is in the same package (`package p2p`). If this were `package p2p_test` (external test package, also valid in Go), this line would fail to compile. This is fine for now but worth knowing.

**No cleanup** — after the test, port 4000 stays open until the process is killed. A proper test would defer `tr.listener.Close()` to release the port when done.

---

## 4. Code Level

```go
func TestTCPTransport(t *testing.T) {
    // t *testing.T is injected by the test runner — it's how you report failures

    listenAddr := ":4000"
    tr := NewTCPTransport(listenAddr)

    assert.Equal(t, tr.listenAddress, listenAddr)
    // checks: tr.listenAddress == ":4000"
    // if false → test fails with a message, but continues running

    assert.Nil(t, tr.ListenAndAccept())
    // checks: ListenAndAccept() returns nil (no error)
    // actually binds port 4000 on your machine when this runs

    select {}
    // empty select = block forever
    // no cases, no default — the goroutine sleeps indefinitely
    // this is the bug — test never returns
}
```
