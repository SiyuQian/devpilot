# Language Patterns

## Literal Formatting

### Field Names

Struct literals must specify **field names** for types defined outside the current package.
For local types with many fields, also use field names.

```go
// Good: external type
r := csv.Reader{Comma: ',', Comment: '#', FieldsPerRecord: 4}

// Bad: positional
r := csv.Reader{',', '#', 4, false, false, false, false}
```

### Matching Braces

Closing brace at same indentation as opening. Don't put closing brace on same line as last
value in multi-line literals:

```go
// Good:
good := []*Type{
    {Key: "multi"},
    {Key: "line"},
}

// Bad:
bad := []*Type{
    {Key: "multi"},
    {Key: "line"}}
```

### Cuddled Braces

Dropping whitespace between braces is only permitted when both:
- Indentation matches
- Inner values are also literals (not variables or expressions)

```go
// Good: cuddled
good := []*Type{{
    Field: "value",
}, {
    Field: "value",
}}

// Bad: can't cuddle when mixing variables and literals
bad := []*Type{
    first,
    {
        Field: "second",
    }}
```

### Repeated Type Names

Omit repeated type names in slice/map literals:

```go
// Good:
[]*Type{{A: 42}, {A: 43}}

// Bad:
[]*Type{&Type{A: 42}, &Type{A: 43}}
```

Run `gofmt -s` to simplify automatically.

### Zero-Value Fields

Omit zero-value fields unless the zero value is meaningful to the reader.

Well-designed APIs use zero-value construction. Omitting zero fields draws attention to
what's actually configured:

```go
// Good: only non-zero fields
ldb := leveldb.Open("/my/table", &db.Options{
    BlockSize:       1 << 16,
    ErrorIfDBExists: true,
})

// Bad: zero fields add noise
ldb := leveldb.Open("/my/table", &db.Options{
    BlockSize:            1 << 16,
    ErrorIfDBExists:      true,
    BlockRestartInterval: 0,
    Comparer:             nil,
    Compression:          nil,
})
```

In table-driven tests, omit zero-value fields when unrelated to the test case.

## Nil Slices

Prefer nil: `var s []string` over `s := []string{}`.

Don't distinguish nil from empty in APIs — use `len(s) == 0` to check emptiness.

```go
// Good:
func describeInts(prefix string, s []int) {
    if len(s) == 0 {
        return
    }
    fmt.Println(prefix, s)
}

// Bad: relies on nil vs empty distinction
func describeInts(prefix string, s []int) {
    if s == nil {
        return
    }
    fmt.Println(prefix, s)
}
```

## Indentation Confusion

Avoid line breaks that align the rest of the line with an indented code block:

```go
// Bad: conditions 3 and 4 look like body code
if longCondition1 && longCondition2 &&
    longCondition3 && longCondition4 {
    log.Info("all conditions met")
}
```

This applies to function signatures, conditionals, and literals.

## Function Formatting

Keep signatures on one line. Don't break parameter lists — it causes indentation confusion.

```go
// Bad:
func (r *SomeType) SomeLongFunctionName(foo1, foo2, foo3 string,
    foo4, foo5, foo6 int) {
    foo7 := bar(foo1) // looks like it's in the function body
}
```

Factor out local variables to shorten call sites:
```go
local := helper(some, parameters, here)
good := foo.Call(list, of, parameters, local)
```

Don't break long strings — break after the format string:
```go
// Good:
log.Warningf("Database key (%q, %d, %q) incompatible in transaction started by (%q, %d, %q)",
    currentCustomer, currentOffset, currentKey,
    txCustomer, txOffset, txKey)

// Bad: broken string
log.Warningf("Database key (%q, %d, %q) incompatible in"+
    " transaction started by (%q, %d, %q)",
    currentCustomer, currentOffset, currentKey, txCustomer,
    txOffset, txKey)
```

## Conditionals and Loops

Don't break `if` conditions across lines. Extract conditions:

```go
// Good:
inTransaction := db.CurrentStatusIs(db.InTransaction)
keysMatch := db.ValuesEqual(db.TransactionKey(), row.Key())
if inTransaction && keysMatch { ... }

// Bad: multi-line if
if db.CurrentStatusIs(db.InTransaction) &&
    db.ValuesEqual(db.TransactionKey(), row.Key()) {
    // ...
}
```

Closures and struct literals in `if` are OK if braces match:
```go
if err := db.RunInTransaction(func(tx *db.TX) error {
    return tx.Execute(userUpdate, x, y, z)
}); err != nil {
    return fmt.Errorf("user update failed: %s", err)
}
```

Don't break `for` statements. Let long lines be long, or refactor:
```go
// OK:
for i, max := 0, collection.Size(); i < max && !collection.HasPendingWriters(); i++ { ... }

// Also OK: refactored
for i, max := 0, collection.Size(); i < max; i++ {
    if collection.HasPendingWriters() {
        break
    }
}
```

`switch`/`case` on single lines. Variable on left: `if result == "foo"` not `if "foo" == result`.

## Switch and Break

No redundant `break` in switch — Go cases don't fall through by default. Use `fallthrough`
for C-style behavior.

```go
// Good:
switch x {
case "A", "B":
    buf.WriteString(x)
case "C":
    // handled outside of the switch statement
default:
    return fmt.Errorf("unknown value: %q", x)
}
```

When `switch` is inside a `for` loop, `break` exits the switch, not the loop. Use a label:
```go
loop:
    for {
        switch x {
        case "A":
            break loop // exits the loop
        }
    }
```

## Copying

Don't copy structs with `sync.Mutex` or `bytes.Buffer`. Use pointer receivers for types
containing these.

```go
// Bad:
b1 := bytes.Buffer{}
b2 := b1 // copies the internal slice — aliasing bug

// Good: use pointer types
func New() *Record { ... }
func (r *Record) Process(...) { ... }
func Consumer(r *Record) { ... }
```

## Type Aliases

Prefer type definitions (`type T1 T2`) over aliases (`type T1 = T2`). Aliases are rare —
primarily for package migration.

## Use %q

Prefer `%q` over `%s` for string formatting in errors and user output:
```go
// Good:
fmt.Printf("value %q looks like English text", someText)

// Bad:
fmt.Printf("value \"%s\" looks like English text", someText)
```

`%q` makes empty strings and control characters visible.

## Use any

Use `any` instead of `interface{}` in new code. They're equivalent.

## Variable Declarations (Best Practice)

- `:=` for non-zero values
- `var` for zero values "ready for later use" (e.g., unmarshal targets)
- Composite literals when values are known at init time
- Always specify channel direction (`<-chan` or `chan<-`)

## Function Arguments (Best Practice)

For complex signatures, use option structures or variadic options:

```go
// Option structure — when most callers specify options:
func New(ctx context.Context, opts Options) *Server

// Variadic options — when most callers need no configuration:
func New(ctx context.Context, opts ...Option) *Server
```
