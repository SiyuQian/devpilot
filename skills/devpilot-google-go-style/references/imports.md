# Imports

## Grouping Order

1. Standard library
2. Third-party / project packages

Separate groups with blank lines.

```go
import (
    "fmt"
    "hash/adler32"
    "os"

    "github.com/dsnet/compress/flate"
    "golang.org/x/text/encoding"
)
```

## Renaming

Don't rename unless:
- Name collision with another import (rename the more local/project-specific one)
- Proto packages with underscores (rename with `pb` suffix): `foosvcpb "path/to/foo_service_go_proto"`
- Truly uninformative name like `v1`

Local names must follow package naming rules — no underscores or capital letters.

If importing a package whose name collides with a common local variable (e.g., `url`, `ssh`),
the preferred rename uses the `pkg` suffix: `urlpkg`.

## Dot Imports

Never use `import . "pkg"`. It obscures where things come from.

```go
// Bad:
import . "foo"
var myThing = Bar() // Where does Bar come from?

// Good:
import "foo"
var myThing = foo.Bar()
```

## Blank Imports

`import _ "pkg"` only in `main` or tests. Never in library packages — it creates hidden
dependencies and prevents tests from using different imports.

Exceptions:
- Bypassing nogo static checker's disallowed imports check
- The `embed` package with `//go:embed` directives
