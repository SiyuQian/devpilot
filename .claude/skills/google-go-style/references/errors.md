# Errors

## Returning Errors

- `error` always last return value
- Return `error` interface, never concrete error types in exported APIs (avoids nil-interface bugs)
- Functions taking `context.Context` should usually return `error`

```go
// Good:
func Good() error { /* ... */ }

// Bad: concrete error type
func Bad() *os.PathError { /* ... */ }
```

## Error Strings

Lowercase, no punctuation (unless starting with exported names, proper nouns, or acronyms):

```go
// Good:
err := fmt.Errorf("something bad happened")

// Bad:
err := fmt.Errorf("Something bad happened.")
```

Display messages (logs, test output) get capitalized:
```go
log.Infof("Operation aborted: %v", err)
t.Errorf("Op(%q) failed unexpectedly; err=%v", args, err)
```

## Handle Errors

Don't discard with `_` unless documented safe. When ignoring:
```go
n, _ := b.Write(p) // never returns a non-nil error
```

Do one of:
- Handle and address the error immediately
- Return the error to the caller
- In exceptional situations, call `log.Fatal` or `panic`

## In-Band Errors

Don't return special values (-1, "", nil) to signal errors. Use multiple returns:

```go
// Bad:
func Lookup(key string) int // returns -1 on not found

// Good:
func Lookup(key string) (value string, ok bool)
```

This prevents callers from writing `Parse(Lookup(key))` — the compiler catches it because
`Lookup` has 2 outputs.

## Indent Error Flow

Handle errors first, happy path stays at left margin:

```go
// Good:
x, err := f()
if err != nil {
    return err
}
// lots of code using x

// Bad:
if x, err := f(); err != nil {
    // error handling
} else {
    // normal code indented unnecessarily
}
```

If using a variable for more than a few lines, avoid the `if`-with-initializer style.

## Structured Errors (Best Practice)

Provide structured errors using sentinel values or custom types rather than requiring string
matching. Use `errors.Is()` with wrapped errors via `fmt.Errorf()` with `%w`.

## Adding Context (Best Practice)

Don't repeat info already in the underlying error. `os.Open()` includes the file path — don't
add it again.

Prefer `%w` at the end:
```go
fmt.Errorf("failed to update user: %w", err)
```

## Don't Double-Log (Best Practice)

Don't log an error and also return it. Let callers decide whether to log. This enables
rate-limiting and proper error handling context.
