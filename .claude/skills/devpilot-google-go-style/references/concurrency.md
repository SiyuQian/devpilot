# Concurrency

## Goroutine Lifetimes

Make goroutine exit conditions obvious. Never start a goroutine without knowing how it will
stop.

Goroutines can leak by blocking on channel sends/receives — the garbage collector won't
terminate them even if no other goroutine holds a reference to the channel.

```go
// Good: goroutines bounded by WaitGroup
func (w *Worker) Run(ctx context.Context) error {
    var wg sync.WaitGroup
    for item := range w.q {
        wg.Add(1)
        go func() {
            defer wg.Done()
            process(ctx, item)
        }()
    }
    wg.Wait() // goroutines don't outlive this function
}

// Bad: fire-and-forget goroutines
func (w *Worker) Run() {
    for item := range w.q {
        go process(item) // may leak, hard to test, undefined shutdown
    }
}
```

Use `context.Context`, `sync.WaitGroup`, signal channels (`chan struct{}`), or condition
variables to make goroutine lifetimes explicit.

## Synchronous Functions

Prefer synchronous functions over async. Let callers add concurrency. Removing concurrency
from an async API is much harder than adding it to a sync one.

Synchronous functions:
- Keep goroutines localized within a call
- Are easier to test
- Help reason about lifetimes
- Avoid leaks and data races

## Contexts

`context.Context` is always the first parameter:
```go
func F(ctx context.Context /* other arguments */) {}
```

Exceptions:
- HTTP handlers: context from `req.Context()`
- Streaming RPC methods: context from the stream

Don't store context in structs — pass as a parameter to each method.

Use `context.Background()` only in entrypoints: `main`, `init`, `TestXXX`, `BenchmarkXXX`,
`FuzzXXX`. In tests, prefer `tb.Context()` (Go 1.24+) over `context.Background()`.

Contexts are immutable — fine to pass the same context to multiple calls sharing the same
deadline, cancellation signal, and credentials.

### Custom Contexts

**Never create custom context types.** Don't use interfaces other than `context.Context` in
function signatures. There are no exceptions.

If every team had a custom context, every function call between packages would need conversion
logic. Put application data in parameters, receivers, globals, or `Context` values.

## crypto/rand

Use `crypto/rand` for generating keys, never `math/rand`. If unseeded, `math/rand` is
completely predictable. Even seeded with `time.Nanoseconds()`, there are few bits of entropy.

```go
func Key() string {
    buf := make([]byte, 16)
    if _, err := rand.Read(buf); err != nil {
        log.Fatalf("out of randomness: %v", err)
    }
    return fmt.Sprintf("%x", buf)
}
```
