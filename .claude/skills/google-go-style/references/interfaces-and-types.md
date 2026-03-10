# Interfaces and Types

## Interfaces

Interfaces belong in the **consumer** package, not the producer. Implementing packages should
return concrete (usually pointer or struct) types.

```go
// Good: consumer defines what it needs
package consumer
type Thinger interface { Thing() bool }
func Foo(t Thinger) string { ... }

// Good: producer returns concrete type
package producer
type Thinger struct{ ... }
func NewThinger() Thinger { return Thinger{...} }

// Bad: producer pre-defines interface
package producer
type Thinger interface { Thing() bool }
func NewThinger() Thinger { return defaultThinger{...} }
```

Rules:
- Don't define interfaces before they're used (YAGNI)
- Don't use interface-typed parameters if users don't need to pass different types
- Don't export interfaces that package users don't need
- Don't export test double implementations — design testable APIs using real implementations
- Consumers can create minimal interfaces containing only the methods they need

## Generics

Use only when interfaces or concrete types don't work. Don't use generics to build DSLs or
assertion frameworks.

Rules:
- Don't use generics just because you're implementing a type-agnostic algorithm — if only one
  type is instantiated in practice, use that type first
- Adding polymorphism later is straightforward; removing unnecessary abstraction is not
- If several types share a useful interface, consider modeling with that interface first
- Otherwise, prefer generics over `any` with type switching

## Pass Values

Don't use `*string` or `*io.Reader` — pass values directly. Fixed-size values should not
be passed by pointer.

Exception: large structs and protocol buffer messages (use pointers). Proto messages satisfy
`proto.Message` interface and often grow over time.

## Receiver Type

Use **pointer** when:
- Method mutates the receiver
- Struct has `sync.Mutex` or similar non-copyable fields
- Large struct
- Elements are pointers to things that may be mutated
- When in doubt

Use **value** when:
- Slice method that doesn't reslice
- Built-in type (int, string) that doesn't need mutation
- Map, function, or channel
- Small immutable struct (like `time.Time`)

Keep all methods for a type consistently pointer or value.

Don't make performance assumptions about pointer vs value — profile with realistic benchmarks.

## Must Functions

Setup helpers that stop the program on failure use the `Must` prefix:

```go
func MustParse(version string) *Version {
    v, err := Parse(version)
    if err != nil {
        panic(fmt.Sprintf("MustParse(%q) = _, %v", version, err))
    }
    return v
}

var DefaultVersion = MustParse("1.2.3")
```

Rules:
- Call them early on program startup, not on user input
- In tests, `Must` helpers should call `t.Helper()` and `t.Fatal`
- Don't use `Must` functions where ordinary error handling is possible

## Don't Panic

Use `error` and multiple returns for normal error handling. Reserve `panic` for:
- API misuse (like slice bounds violations)
- Internal implementation details that never escape the package
- `Must` functions at init time

For `main` and initialization code, use `log.Exit` (terminates without stack trace) for
configuration errors rather than panic.

## Package Organization (Best Practice)

Balance cohesion with usability. If client code needs multiple types to interact meaningfully,
combine them. Use the standard library as reference: `bytes`, `http`, `net`.
