# Commentary

## Comment Line Length

Aim for ~80 characters. Wrap at punctuation. Don't break URLs. Don't force-wrap lines that
would look worse broken.

```go
// Good:
// This is a comment paragraph.
// The length of individual lines doesn't matter in Godoc;
// but the choice of wrapping makes it easy to read on narrow screens.
//
// Don't worry too much about the long URL:
// https://supercalifragilisticexpialidocious.example.com:8080/Animalia/Chordata/
```

Avoid comments that wrap repeatedly on small screens:

```go
// Bad:
// This is a comment paragraph. The length of individual lines doesn't matter in
// Godoc; but the choice of wrapping causes jagged lines on narrow screens or in code
// review, which can be annoying, especially when in a comment block that will wrap
// repeatedly.
```

## Doc Comments

All exported names must have doc comments. Full sentences starting with the object's name:

```go
// A Request represents a request to run a command.
type Request struct { ...

// Encode writes the JSON encoding of req to w.
func Encode(w io.Writer, req *Request) { ...
```

Struct field groups can share a comment:
```go
type Options struct {
    // General setup:
    Name  string
    Group *FooGroup

    // Dependencies:
    DB *sql.DB

    // Customization:
    LargeGroupThreshold int // optional; default: 10
    MinimumMembers      int // optional; default: 2
}
```

For unexported code, start doc comments with the unexported name — eases later export.

## Comment Sentences

Comments that are complete sentences should be capitalized and punctuated like standard English.
You may begin with an uncapitalized identifier name if clear.

Comments that are sentence fragments don't require capitalization or punctuation.

Doc comments should always be complete sentences. Simple end-of-line comments (especially for
struct fields) can be phrases assuming the field name is the subject.

```go
// A Server handles serving quotes from Shakespeare's collected works.
type Server struct {
    WelcomeMessage  string // displayed when user logs in
    ProtocolVersion string // checked against incoming requests
    PageLength      int    // lines per page when printing (optional; default: 20)
}
```

## Examples

Packages should clearly document intended usage. Provide runnable examples when possible —
they appear in Godoc and run as tests. Runnable examples belong in test files, not production
source.

If runnable examples aren't feasible, provide example code in comments following standard
formatting conventions.

## Named Result Parameters

Use named results when:
- Two or more results of the same type: `func (n *Node) Children() (left, right *Node, err error)`
- The name suggests an action the caller must take: `func WithTimeout(...) (ctx Context, cancel func())`

Don't use named results when:
- The name just repeats the type: `func (n *Node) Parent1() (node *Node)` — just return `*Node`
- You only want to enable naked returns in a medium+ function — be explicit

Naked returns are acceptable only in small functions. Once medium-sized, be explicit about
returned values.

It's always acceptable to name a result parameter if its value must be changed in a deferred
closure.

## Package Comments

Directly above `package` clause, no blank line between. One per package.

```go
// Package math provides basic constants and mathematical functions.
package math
```

For `main` packages, describe the command:
```go
// The seed_generator command generates a Finch seed file from JSON configs.
package main
```

If no obvious primary file exists or documentation is long, place the comment in `doc.go`.

## Documentation Best Practices

### Parameter Documentation
Focus on non-obvious or error-prone fields. Explain "why" when context adds value.

### Context Behavior
Don't document standard context cancellation. Only document divergent behavior.

### Concurrency Safety
Document whether operations are safe for concurrent use only when unclear.

### Cleanup Requirements
Explicitly state cleanup obligations: `defer resp.Body.Close()`.

### Error Documentation
Document significant error values returned. Specify pointer vs value for `errors.Is()`/`errors.As()`.

### Godoc Formatting
Blank lines between paragraphs. Indent code with two extra spaces. Include runnable examples
in test files.
