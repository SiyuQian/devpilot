# Formatting (Clean Code, Ch. 5)

Formatting is communication, and communication is the professional developer's first order of
business.

**Team rules trump personal preference.** Once agreed, enforce with an automatic formatter.

## The Newspaper Metaphor

A source file should read like a newspaper article:
- **Headline** (name) tells you if you're in the right place.
- **Top** — high-level summary (public interface, main flow).
- **Details** grow as you descend.

## Vertical Formatting

### File Size

Small is better. 200 lines is easy to reason about. 500 lines starts to hurt. Files over 1000 lines
are a smell.

### Vertical Openness Between Concepts

Blank lines between groups of related lines signal transitions of thought:

```go
package widgets

import "regexp"

const boldRegexp = `'''.+?'''`

var boldPattern = regexp.MustCompile(`'''(.+?)'''`)

type BoldWidget struct{ parent *ParentWidget }

func NewBoldWidget(parent *ParentWidget, text string) *BoldWidget {
    w := &BoldWidget{parent: parent}
    if m := boldPattern.FindStringSubmatch(text); m != nil {
        w.addChildWidgets(m[1])
    }
    return w
}
```

Notice the blank lines between `package`/`import`/const/var/type/func — each section is its own
thought. TypeScript follows the same convention with blank lines between imports, top-level
constants, and class/function declarations.

### Vertical Density

Lines of code that are tightly related should appear vertically dense. Don't break them up with
blank lines or useless comments.

### Vertical Distance

Concepts closely related should be kept close. Things that aren't related shouldn't be in the same
file.

- **Variable declarations** should appear as close to their usage as possible. Local variables at the
  top of the function. Loop variables in the loop.
- **Instance variables** at the top of the class (or bottom — pick one, consistently).
- **Dependent functions**: if one function calls another, put the caller above the callee so readers
  can scroll down to drill in.
- **Conceptual affinity**: functions that perform similar operations belong together, even without
  direct calls.

### Vertical Ordering

General → specific. Top → bottom. The reader shouldn't need to jump around.

## Horizontal Formatting

### Line Width

80–120 characters. Longer lines hurt scanability.

### Horizontal Openness and Density

Space around operators for separation:
```ts
const lineSize = line.length;
totalChars += lineSize;
lineWidthHistogram.addLine(lineSize, lineCount);
```

No space between function name and its open parenthesis:
```ts
measureLine(line);  // function is a unit with its args
```

Tighter binding for higher-precedence operators:
```ts
b*b - 4*a*c
```

**In Go, `gofmt` decides horizontal formatting for you — trust it.** In TypeScript, Prettier /
ESLint plays the same role.

### Horizontal Alignment

Don't align variable names or right-hand sides — the alignment draws the eye away from meaning and
becomes a maintenance burden. Just use single spaces.

### Indentation

Make the hierarchy visible. Never collapse scopes onto one line:
```ts
// ❌ Bad
class CommentWidget extends TextWidget { static REGEXP = "..."; constructor() { super(); } }

// ✅ Actually indent.
```

### Empty Loops

If you genuinely need an empty loop body, make it explicit:

```ts
while (await stream.read(buf) !== -1) { /* drain */ }
```

```go
for dis.Read(buf) != io.EOF {
    // drain
}
```
Don't leave an empty-looking line that readers might misread.

## Team Rules

Every programmer has formatting preferences, but professionals subordinate theirs to the team's.
Consistency across the codebase matters more than any single rule. Configure the formatter once and
let it decide forever.
