---
name: devpilot-google-go-style
description: >
  ALWAYS use when working in a Go project — writing new Go code, modifying .go files, reviewing Go PRs,
  or discussing Go design decisions. Triggers on any .go file interaction, Go package design, Go naming
  questions, Go error handling patterns, Go test writing, or Go code review. Even if the user doesn't
  mention "style" or "conventions", if you're touching Go code, this skill applies.
license: Complete terms in LICENSE.txt
---

# Google Go Style Guide

This skill enforces the Google Go Style Guide when writing or reviewing Go code. It covers naming,
formatting, error handling, documentation, testing, and language patterns.

The guide is organized by priority: naming and errors matter most because they affect every reader.
Formatting is handled by `gofmt`. The rest improves clarity and maintainability.

When **writing** Go code, follow these rules directly. When **reviewing** Go code, flag violations
with the specific rule and a concrete fix.

## Quick Reference — The Rules That Matter Most

### Naming

**MixedCaps everywhere.** Use `MixedCaps` or `mixedCaps`, never `snake_case`. This applies to
constants, variables, functions, methods — everything except filenames and flag names.

**No stuttering.** Don't repeat the package name in exported symbols:
- `widget.New` not `widget.NewWidget`
- `db.Load` not `db.LoadFromDatabase`

**Short receivers.** One or two letters, abbreviating the type. Consistent across all methods:
- `func (s *Server)` not `func (self *Server)` or `func (server *Server)`

**No Get prefix.** Use `Counts()` not `GetCounts()`. Use `Compute` or `Fetch` for expensive operations.

**Variable length proportional to scope.** Single-letter vars are fine in small scopes (loops, short
functions). Larger scopes need descriptive names. Don't abbreviate by dropping letters — `Sandbox`
not `Sbx`.

**Don't encode types in names:**
- `users` not `userSlice`
- `count` not `numUsers` or `usersInt`

**Initialisms keep consistent case:** `URL` or `url`, `ID` or `id`, `HTTP` or `http`. Never `Url`,
`Id`, or `Http`.

**Avoid uninformative package names:** `util`, `common`, `helper`, `base`, `model` are banned.
Package names should convey what they do.

### Errors

**Always return `error` as the last return value.** Use the `error` interface type, never concrete
error types in exported function signatures (avoids nil-interface bugs).

**Error strings are lowercase, no punctuation:**
```go
// Good:
fmt.Errorf("something bad happened")

// Bad:
fmt.Errorf("Something bad happened.")
```

**Handle every error.** Don't discard errors with `_` unless the function is documented to never fail
(like `bytes.Buffer.Write`). When ignoring, add a comment explaining why.

**Indent error flow — happy path at the left margin:**
```go
// Good:
if err != nil {
    return err
}
// normal code continues unindented

// Bad:
if err != nil {
    // error handling
} else {
    // normal code indented unnecessarily
}
```

**No in-band errors.** Don't return `-1` or `""` to signal failure. Use multiple returns:
```go
// Good:
func Lookup(key string) (value string, ok bool)

// Bad:
func Lookup(key string) int // returns -1 on not found
```

**Wrap errors with context using `%w`:**
```go
return fmt.Errorf("failed to load user: %w", err)
```

### Documentation

**All exported names get doc comments.** Start with the name. Use full sentences:
```go
// A Request represents a request to run a command.
type Request struct { ...

// Encode writes the JSON encoding of req to w.
func Encode(w io.Writer, req *Request) { ...
```

**Comment line length ~80 chars.** Not a hard limit, but wrap for readability on narrow screens.
Don't break URLs.

**Package comments** go directly above `package` with no blank line. One per package. For `main`
packages, describe the binary's purpose.

### Formatting & Structure

**`gofmt` is mandatory.** Run `gofmt -s` to also simplify composite literals.

**Keep function signatures on one line.** Don't break parameter lists across lines — it causes
indentation confusion. Factor out local variables to shorten call sites instead.

**Don't break `if` conditions across lines.** Extract boolean operands into named variables:
```go
// Good:
inTransaction := db.CurrentStatusIs(db.InTransaction)
keysMatch := db.ValuesEqual(db.TransactionKey(), row.Key())
if inTransaction && keysMatch {
    // ...
}

// Bad:
if db.CurrentStatusIs(db.InTransaction) &&
    db.ValuesEqual(db.TransactionKey(), row.Key()) {
    // ...
}
```

**`switch`/`case` on single lines.** Don't break case lists across lines unless excessively long.

**No redundant `break` in switch.** Go cases don't fall through by default.

**Variable on left in comparisons:** `if result == "foo"` not `if "foo" == result`.

### Imports

**Group imports in order:** (1) standard library, (2) third-party/project packages. Separate groups
with blank lines.

**Don't rename imports** unless there's a collision or the name is truly uninformative (like `v1`).
When renaming proto packages, use a `pb` suffix.

**No dot imports** (`import . "pkg"`). They obscure where things come from.

**Blank imports** (`import _ "pkg"`) only in `main` or tests, never in libraries.

### Interfaces

**Interfaces belong in the consumer package, not the producer. This applies when writing code too — not
just during reviews.** When you create a service that depends on a database, do NOT define the DB
interface in the same package. The interface should live where it's consumed:

```go
// ❌ BAD: interface defined alongside its implementation (producer package)
package users
type DB interface { FindUser(id string) (*User, error) }
type Service struct { db DB }

// ✅ GOOD: consumer defines its own interface; producer returns a concrete type
package users
type Service struct { db *postgres.Client }

// If you need testability, define a minimal interface in the TEST file:
package users_test
type stubDB struct { ... }
```

**When writing a new package with dependencies:** accept a concrete type, not an interface you just
invented. If you need to abstract the dependency for testing, define the interface in the test file
or in the consuming package — never next to the implementation.

**Return concrete types from constructors.** Let consumers define interfaces for what they need.

**Don't define interfaces before you need them.** YAGNI applies strongly here.

### Testing

**No assertion libraries.** Use `cmp.Equal` and standard `t.Errorf`/`t.Fatalf` with descriptive
messages. Test failures should be diagnosable without reading the test source.

**Table-driven tests:** Use field names in struct literals. Omit zero-value fields when they're not
relevant to the test case.

**Test helpers call `t.Helper()` — but only helper functions, never `Test` functions themselves.**
`t.Helper()` goes in functions like `assertUser(t, got, want)`, not in `TestGetUser(t)`.

**Use `t.Fatal` for setup failures**, `t.Error` for test case validation (allows other cases to run).

### Concurrency

**Prefer synchronous functions.** Let callers add concurrency if needed. Removing concurrency from
an async API is much harder than adding it to a sync one.

**Make goroutine lifetimes obvious.** Use `context.Context` for cancellation. Use `sync.WaitGroup`
to ensure goroutines don't outlive their parent function.

**`context.Context` is always the first parameter.** Don't store it in structs.

### Other Rules

**Prefer `nil` slices** over empty literals (`var s []string` not `s := []string{}`). Don't design
APIs that distinguish between nil and empty slices — use `len(s) == 0` to check emptiness.

**Use `%q`** for string formatting in error messages and user-facing output. It handles empty strings
and control characters gracefully.

**Use `any`** instead of `interface{}` in new code.

**Don't `panic` for normal error handling.** Reserve `panic` for truly impossible conditions (bugs).
Use `Must` prefix for init-time helpers that panic on failure.

**Don't copy structs with `sync.Mutex` or `bytes.Buffer` fields.** Use pointer receivers and
pointer parameters for types containing these.

**Use `crypto/rand`** for generating keys, never `math/rand`.

**Generics:** Use only when concrete types or interfaces don't suffice. Don't use generics to build
DSLs or assertion frameworks.

**Type aliases are rare.** Prefer type definitions (`type T1 T2`) over aliases (`type T1 = T2`).

## Struct Literals

**Use field names** for types from other packages — always. For local types with many fields — also
use field names.

**Omit zero-value fields** unless the zero value is meaningful to the reader.

**Matching braces:** closing brace at same indentation as opening. Don't put closing brace on same
line as last value in multi-line literals.

**Omit repeated type names** in slice/map literals:
```go
// Good:
[]*Type{{A: 42}, {A: 43}}

// Bad:
[]*Type{&Type{A: 42}, &Type{A: 43}}
```

## Common Mistakes

These are the violations agents commit most often, even when they know the rules:

| Mistake | Why it happens | Fix |
|---------|---------------|-----|
| Define interface in same package as implementation | Feels "clean" to co-locate | Accept concrete types; define interfaces in consumer or test file |
| `GetUser` instead of `User` or `ByID` | Habit from other languages | Drop the `Get` prefix — Go getters don't use it |
| `UserFilter` in package `users` | Forget the package name is part of the API | Call it `Filter` — callers see `users.Filter` |
| Store `context.Context` in a struct | Convenient for "reuse" | Pass `ctx` as the first param to every method that needs it |
| `t.Helper()` in `TestXxx` functions | Confuse test functions with helper functions | Only call `t.Helper()` in non-`Test` helper functions |
| `Id`, `Url`, `Http` | Muscle memory | All-caps for initialisms: `ID`, `URL`, `HTTP` |
| `map[string]*User{}` instead of nil | Defensive instinct | Use `var m map[string]*User` unless you need a non-nil empty map |

## Review Checklist

When reviewing Go code, check for these violations in priority order:

1. **Naming:** stuttering, snake_case, Get prefix, uninformative names, wrong initialism casing
2. **Error handling:** discarded errors, in-band errors, capitalized error strings, missing wrapping
3. **Documentation:** missing doc comments on exported symbols, comments not starting with the name
4. **Interfaces:** defined in producer not consumer, premature interface definitions
5. **Testing:** assertion libraries, missing `t.Helper()`, unclear failure messages
6. **Formatting:** multi-line `if` conditions, broken function signatures, redundant `break`
7. **Concurrency:** context not first param, unclear goroutine lifetimes, context stored in struct

For detailed rules with more examples, read the relevant reference file:
- `references/naming.md` — MixedCaps, package names, receivers, getters, repetition, variable names
- `references/commentary.md` — Doc comments, comment sentences, examples, package comments
- `references/imports.md` — Import grouping, renaming, dot imports, blank imports
- `references/errors.md` — Returning errors, error strings, wrapping, in-band errors, indent error flow
- `references/language-patterns.md` — Literals, nil slices, formatting, conditionals, switch, copying, type aliases
- `references/interfaces-and-types.md` — Interfaces, generics, receivers, Must functions, panic
- `references/concurrency.md` — Goroutine lifetimes, contexts, synchronous functions, crypto/rand
- `references/testing.md` — Test failures, table-driven tests, cmp.Diff, test helpers, no assertion libraries
