# Testing

## Useful Test Failures

Test failures must be diagnosable without reading the test source. Include:
- What caused the failure
- What inputs resulted in error
- Actual result vs expected

### Identify the Function

Failure messages should include the function name:
```go
// Good:
t.Errorf("YourFunc(%v) = %v, want %v", input, got, want)

// Bad:
t.Errorf("got %v, want %v", got, want)
```

### Identify the Input

Include function inputs if they're short. If inputs are large or opaque, name your test cases
and print the description.

### Got Before Want

Always print actual value before expected: `YourFunc(%v) = %v, want %v`.

For diffs, explicitly indicate direction: `(-want +got)`.

### Full Structure Comparisons

For structs, slices, arrays, maps — don't compare field-by-field. Construct expected data and
compare directly:

```go
want := &Doc{
    Type:     "blogPost",
    Comments: 2,
    Body:     "This is the post body.",
    Authors:  []string{"isaac", "albert", "emmy"},
}
if !cmp.Equal(got, want) {
    t.Errorf("AddPost() = %+v, want %+v", got, want)
}
```

For approximate equality or fields that can't be compared, use `cmpopts`:
```go
if diff := cmp.Diff(want, got, cmpopts.IgnoreInterfaces(...)); diff != "" {
    t.Errorf("Foo() mismatch (-want +got):\n%s", diff)
}
```

Multiple return values can be compared individually — no need to wrap in a struct.

### Compare Stable Results

Don't compare results that depend on output stability of packages you don't own. For example,
`json.Marshal` can change the specific bytes it emits. Parse and compare semantically instead.

### Keep Going

Tests should continue after failure to report all failed checks in one run. Use `t.Error`
for mismatches (continues), `t.Fatal` for setup failures (stops).

```go
// Good: continues to check all properties
gotMean, gotVariance, err := MyDistribution(input)
if err != nil {
    t.Fatalf("MyDistribution(%v) returned unexpected error: %v", input, err)
}
if diff := cmp.Diff(wantMean, gotMean); diff != "" {
    t.Errorf("mean mismatch (-want +got):\n%s", input, diff)
}
if diff := cmp.Diff(wantVariance, gotVariance); diff != "" {
    t.Errorf("variance mismatch (-want +got):\n%s", input, diff)
}
```

Use `t.Fatal` when subsequent checks would be meaningless (e.g., encoding failed, so
decoding makes no sense).

For table-driven tests, use subtests and `t.Fatal` within each subtest.

### Equality Comparison and Diffs

- `==` works for scalars. Pointers compare identity, not values.
- Use `cmp.Equal` for complex data: slices, maps, structs with nested fields
- Use `cmp.Diff` for human-readable diffs in failure messages
- `cmp` is maintained by the Go team and produces stable results

```go
if diff := cmp.Diff(want, got); diff != "" {
    t.Errorf("Foo() mismatch (-want +got):\n%s", diff)
}
```

## No Assertion Libraries

Use `cmp.Equal` and standard `t.Errorf`/`t.Fatalf` with descriptive messages. Don't create
or use assertion libraries — they fragment developer experience and produce less useful
failure messages.

```go
// Bad:
assert.IsNotNil(t, "obj", obj)
assert.StringEq(t, "obj.Type", obj.Type, "blogPost")

// Good:
if !cmp.Equal(got, want) {
    t.Errorf("Blog post = %v, want = %v", got, want)
}
```

## Table-Driven Tests

Use field names in struct literals. Omit irrelevant zero-value fields:

```go
tests := []struct {
    input      string
    wantPieces []string
    wantErr    error
}{
    {
        input:      "1.2.3.4",
        wantPieces: []string{"1", "2", "3", "4"},
    },
    {
        input:   "hostname",
        wantErr: ErrBadHostname,
    },
}
```

Use explicit field names when:
- Large test cases
- Fields share types (ambiguity risk)
- Testing zero values that are meaningful

## Test Helpers (Best Practice)

- Call `t.Helper()` to attribute failures to the right line
- Fail fast with `t.Fatal` and descriptive messages
- Don't return errors — call `t.Fatal` directly

## Test Structure (Best Practice)

Keep validation and failure reporting in the test function itself. Don't hide them in
assertion helpers.

## Acceptance Tests (Best Practice)

Create test packages validating interface implementations without knowing internals. Package
in a `test`-suffixed package, following `testing/fstest` pattern.

## Real Transports (Best Practice)

Prefer production clients connected to test doubles over hand-implementing clients.

## Error Handling in Tests (Best Practice)

- `t.Fatal()` for setup failures
- `t.Error()` with `continue` for table entry failures
- In subtests: `t.Fatal()` is fine since it only stops the subtest
