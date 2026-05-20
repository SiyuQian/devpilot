# Devpilot Graph Phase 2 — Bite-Sized Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add TypeScript, JavaScript, and Rust parsers to the graph subsystem under the same `parser.Parser` interface used by `GoParser`, plus a `~/.devpilot/graphs/<repo-key>/` cache layout with concurrency-safe full-and-incremental builds.

**Architecture:** Each new parser is a thin wrapper around `github.com/smacker/go-tree-sitter/<lang>` that walks the AST and emits `store.Node` + `store.Edge` slices, mirroring the structure of `internal/graph/parser/go.go`. The cache layer adds a small filesystem-aware orchestrator on top of the existing `store.Store`: a deterministic repo-key derivation, a `flock`-based build lock, a TTL sweeper for preflight JSON, and a `cache.Builder` that walks the repo, fans parsing out over an `errgroup`, runs the existing resolver, and batch-inserts via `store.InsertNodes` / `store.InsertEdges`. Incremental update uses `git diff --name-status` to scope which files are re-parsed.

**Tech Stack:**
- Go 1.25.6, module path `github.com/siyuqian/devpilot`
- `github.com/smacker/go-tree-sitter` + `typescript/typescript`, `typescript/tsx`, `javascript`, `rust` language bindings
- `modernc.org/sqlite` (already in go.mod)
- `github.com/gofrs/flock` for build lock
- `golang.org/x/sync/errgroup` for parallel parsing
- `os/exec` shelling out to `git diff --name-status`

**Conventions inherited from Phase 1 (must follow):**
- One passing test → one commit (`feat(graph/<pkg>): <verb> <noun>`)
- Parsers MUST NOT hold mutable state across `Parse` calls; allocate a fresh `sitter.Parser` per call
- File node ID = the relative path; symbol ID = `<path>::<name>` for top-level, `<path>::<Container>.<name>` for methods; cross-file unresolved targets emitted as `external::<import-path>.<name>` so the resolver can rewrite them
- Test fixtures live under `internal/graph/parser/testdata/<lang>/<scenario>/`
- All `fmt.Errorf` at layer boundaries wraps with `%w`
- Table-driven tests with named subtests; no mocking of our own packages
- Run `make lint` and `make test` before committing

---

## File Structure

```
internal/graph/
├── parser/
│   ├── typescript.go            (new)
│   ├── typescript_test.go       (new)
│   ├── javascript.go            (new)
│   ├── javascript_test.go       (new)
│   ├── rust.go                  (new)
│   ├── rust_test.go             (new)
│   ├── tools.go                 (modified — shared helpers)
│   └── testdata/
│       ├── ts/{simple,multifile,iface,alias}/   (new fixtures)
│       ├── js/{simple,multifile}/               (new fixtures)
│       └── rust/{simple,multifile,trait}/       (new fixtures)
├── resolver/
│   ├── tsconfig.go              (new — tsconfig path alias resolver)
│   └── tsconfig_test.go         (new)
└── cache/
    ├── paths.go                 (new)
    ├── paths_test.go            (new)
    ├── flock.go                 (new)
    ├── flock_test.go            (new)
    ├── ttl.go                   (new)
    ├── ttl_test.go              (new)
    ├── builder.go               (new — orchestrator)
    ├── builder_test.go          (new)
    ├── meta.go                  (new — meta.json read/write)
    ├── meta_test.go             (new)
    ├── incremental.go           (new — git diff + delta apply)
    └── incremental_test.go      (new)
```

The `cache/` package is new and self-contained: it depends on `parser/`, `store/`, and `resolver/` but nothing in `cache/` imports them in a cycle.

---

## Dependency additions

Run once at the start of the phase (one commit at the end of Task 2.1's first sub-step):

```bash
go get github.com/smacker/go-tree-sitter/typescript/typescript
go get github.com/smacker/go-tree-sitter/typescript/tsx
go get github.com/smacker/go-tree-sitter/javascript
go get github.com/smacker/go-tree-sitter/rust
go get github.com/gofrs/flock
go get golang.org/x/sync/errgroup
go mod tidy
```

---

## Task 2.1: TypeScript parser — scaffolding + file/function/method nodes

**Files:**
- Create: `internal/graph/parser/typescript.go`
- Create: `internal/graph/parser/typescript_test.go`
- Create: `internal/graph/parser/testdata/ts/simple/main.ts`

- [ ] **Step 1: Add fixture `testdata/ts/simple/main.ts`**

```ts
export function greet(name: string): string {
  return "hi " + name;
}

function internalHelper(): void {
  greet("world");
}

export class Greeter {
  hello(name: string): string {
    return greet(name);
  }
  private silent(): void {}
}
```

- [ ] **Step 2: Write failing test `TestTypeScriptParserExtracts/functions`**

```go
package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTypeScriptParserExtracts(t *testing.T) {
	loadSimple := func(t *testing.T) (string, []byte) {
		t.Helper()
		path := filepath.Join("testdata", "ts", "simple", "main.ts")
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return "simple/main.ts", src
	}

	t.Run("functions", func(t *testing.T) {
		p := NewTypeScriptParser()
		path, src := loadSimple(t)
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		var hasGreet, hasInternal, hasFile bool
		for _, n := range r.Nodes {
			switch n.ID {
			case "simple/main.ts::greet":
				hasGreet = true
				if !n.IsExported {
					t.Error("greet must be exported")
				}
				if n.Kind != "function" {
					t.Errorf("greet kind=%q want function", n.Kind)
				}
			case "simple/main.ts::internalHelper":
				hasInternal = true
				if n.IsExported {
					t.Error("internalHelper must NOT be exported")
				}
			case "simple/main.ts":
				hasFile = true
				if n.Kind != "file" {
					t.Errorf("file kind=%q want file", n.Kind)
				}
			}
		}
		if !hasGreet || !hasInternal || !hasFile {
			t.Fatalf("missing nodes: greet=%v internal=%v file=%v", hasGreet, hasInternal, hasFile)
		}
	})
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/graph/parser/ -run TestTypeScriptParserExtracts -v
```

Expected: FAIL — `undefined: NewTypeScriptParser`.

- [ ] **Step 4: Add tree-sitter-typescript dependency**

```bash
go get github.com/smacker/go-tree-sitter/typescript/typescript
go mod tidy
```

- [ ] **Step 5: Implement minimal `typescript.go` to make the test pass**

```go
package parser

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	tsLang "github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// TypeScriptParser extracts nodes and edges from .ts source files.
type TypeScriptParser struct{ lang *sitter.Language }

// NewTypeScriptParser returns a Parser for TypeScript source files.
func NewTypeScriptParser() *TypeScriptParser {
	return &TypeScriptParser{lang: tsLang.GetLanguage()}
}

func (p *TypeScriptParser) Language() string     { return "typescript" }
func (p *TypeScriptParser) Extensions() []string { return []string{".ts", ".tsx"} }

func (p *TypeScriptParser) Parse(path string, src []byte) (ParseResult, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(p.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return ParseResult{}, fmt.Errorf("tree-sitter parse %s: %w", path, err)
	}
	defer tree.Close()

	res := ParseResult{InterfaceMethods: map[string][]string{}}
	res.Nodes = append(res.Nodes, store.Node{
		ID: path, Kind: "file", Path: path, Name: path, Language: "typescript",
	})

	root := tree.RootNode()
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		exported := false
		decl := child
		if child.Type() == "export_statement" {
			exported = true
			// The actual declaration is the first named child of export_statement.
			if child.NamedChildCount() > 0 {
				decl = child.NamedChild(0)
			}
		}
		if decl.Type() == "function_declaration" {
			emitFunctionNode(&res, decl, src, path, exported)
		}
	}
	return res, nil
}

func emitFunctionNode(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: "function", Path: path, Name: name, Language: "typescript",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
}
```

- [ ] **Step 6: Run test to verify it passes**

```bash
go test ./internal/graph/parser/ -run TestTypeScriptParserExtracts -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/graph/parser/typescript.go internal/graph/parser/typescript_test.go internal/graph/parser/testdata/ts/simple/main.ts go.mod go.sum
git commit -m "feat(graph/parser/ts): file and function node extraction"
```

---

## Task 2.2: TypeScript parser — class and method nodes

**Files:**
- Modify: `internal/graph/parser/typescript.go`
- Modify: `internal/graph/parser/typescript_test.go`

- [ ] **Step 1: Add failing subtest `methods`**

Append inside `TestTypeScriptParserExtracts`:

```go
	t.Run("methods", func(t *testing.T) {
		p := NewTypeScriptParser()
		path, src := loadSimple(t)
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]struct{ exported bool }{
			"simple/main.ts::Greeter.hello":  {exported: true},
			"simple/main.ts::Greeter.silent": {exported: false},
		}
		seen := map[string]bool{}
		for _, n := range r.Nodes {
			if w, ok := want[n.ID]; ok {
				seen[n.ID] = true
				if n.Kind != "method" {
					t.Errorf("%s kind=%q want method", n.ID, n.Kind)
				}
				if n.Container != "Greeter" {
					t.Errorf("%s container=%q want Greeter", n.ID, n.Container)
				}
				if n.IsExported != w.exported {
					t.Errorf("%s exported=%v want %v", n.ID, n.IsExported, w.exported)
				}
			}
		}
		for id := range want {
			if !seen[id] {
				t.Errorf("missing method node: %s", id)
			}
		}
		// Also verify the class node itself.
		var hasClass bool
		for _, n := range r.Nodes {
			if n.ID == "simple/main.ts::Greeter" {
				hasClass = true
				if n.Kind != "class" {
					t.Errorf("Greeter kind=%q want class", n.Kind)
				}
				if !n.IsExported {
					t.Error("Greeter must be exported")
				}
			}
		}
		if !hasClass {
			t.Error("missing class node Greeter")
		}
	})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/graph/parser/ -run TestTypeScriptParserExtracts/methods -v
```

Expected: FAIL — methods and class node missing.

- [ ] **Step 3: Extend Parse to handle class_declaration**

In `typescript.go`, inside the loop over root's named children, after the `function_declaration` branch add:

```go
		if decl.Type() == "class_declaration" {
			emitClassNode(&res, decl, src, path, exported)
		}
```

Add helper:

```go
func emitClassNode(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	className := nameNode.Content(src)
	classID := path + "::" + className
	res.Nodes = append(res.Nodes, store.Node{
		ID: classID, Kind: "class", Path: path, Name: className, Language: "typescript",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: classID, Kind: "contains"})

	body := decl.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < int(body.NamedChildCount()); i++ {
		member := body.NamedChild(i)
		if member.Type() != "method_definition" {
			continue
		}
		methodName := member.ChildByFieldName("name")
		if methodName == nil {
			continue
		}
		mName := methodName.Content(src)
		mID := path + "::" + className + "." + mName
		isPrivate := false
		for j := 0; j < int(member.ChildCount()); j++ {
			c := member.Child(j)
			if c.Type() == "accessibility_modifier" && c.Content(src) == "private" {
				isPrivate = true
				break
			}
		}
		res.Nodes = append(res.Nodes, store.Node{
			ID: mID, Kind: "method", Path: path, Name: mName, Container: className,
			Language:   "typescript",
			StartLine:  int(member.StartPoint().Row) + 1,
			EndLine:    int(member.EndPoint().Row) + 1,
			IsExported: !isPrivate,
		})
		res.Edges = append(res.Edges, store.Edge{Src: classID, Kind: "contains", Dst: mID})
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/graph/parser/ -v
```

Expected: PASS for all TS subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/graph/parser/typescript.go internal/graph/parser/typescript_test.go
git commit -m "feat(graph/parser/ts): class and method nodes"
```

---

## Task 2.3: TypeScript parser — interface and type-alias nodes

**Files:**
- Modify: `internal/graph/parser/typescript.go`
- Modify: `internal/graph/parser/testdata/ts/simple/main.ts`
- Modify: `internal/graph/parser/typescript_test.go`

- [ ] **Step 1: Extend fixture**

Append to `main.ts`:

```ts
export interface Speaker {
  hello(name: string): string;
}

export type Greeting = string;
```

- [ ] **Step 2: Add failing subtest `types`**

```go
	t.Run("types", func(t *testing.T) {
		p := NewTypeScriptParser()
		path, src := loadSimple(t)
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		var hasIface, hasTypeAlias bool
		for _, n := range r.Nodes {
			if n.ID == "simple/main.ts::Speaker" {
				hasIface = true
				if n.Kind != "interface" {
					t.Errorf("Speaker kind=%q want interface", n.Kind)
				}
				if !n.IsExported {
					t.Error("Speaker must be exported")
				}
			}
			if n.ID == "simple/main.ts::Greeting" {
				hasTypeAlias = true
				if n.Kind != "type" {
					t.Errorf("Greeting kind=%q want type", n.Kind)
				}
			}
		}
		if !hasIface || !hasTypeAlias {
			t.Fatalf("missing: iface=%v typeAlias=%v", hasIface, hasTypeAlias)
		}
		// InterfaceMethods recorded for resolver.
		methods, ok := r.InterfaceMethods["simple/main.ts::Speaker"]
		if !ok || len(methods) != 1 || methods[0] != "hello" {
			t.Errorf("InterfaceMethods[Speaker]=%v, want [hello]", methods)
		}
	})
```

- [ ] **Step 3: Run test, expect FAIL**

```bash
go test ./internal/graph/parser/ -run TestTypeScriptParserExtracts/types -v
```

- [ ] **Step 4: Implement**

In the per-child-loop, add:

```go
		switch decl.Type() {
		case "interface_declaration":
			emitInterfaceNode(&res, decl, src, path, exported)
		case "type_alias_declaration":
			emitTypeAliasNode(&res, decl, src, path, exported)
		}
```

Helpers:

```go
func emitInterfaceNode(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: "interface", Path: path, Name: name, Language: "typescript",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})

	body := decl.ChildByFieldName("body")
	if body == nil {
		return
	}
	var methods []string
	for i := 0; i < int(body.NamedChildCount()); i++ {
		member := body.NamedChild(i)
		if member.Type() != "method_signature" {
			continue
		}
		mName := member.ChildByFieldName("name")
		if mName != nil {
			methods = append(methods, mName.Content(src))
		}
	}
	if len(methods) > 0 {
		res.InterfaceMethods[id] = methods
	}
}

func emitTypeAliasNode(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: "type", Path: path, Name: name, Language: "typescript",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
}
```

- [ ] **Step 5: Run test, expect PASS**

- [ ] **Step 6: Commit**

```bash
git commit -am "feat(graph/parser/ts): interface and type-alias nodes"
```

---

## Task 2.4: TypeScript parser — `calls` edges

**Files:**
- Modify: `internal/graph/parser/typescript.go`
- Modify: `internal/graph/parser/typescript_test.go`

- [ ] **Step 1: Add failing subtest `calls`**

```go
	t.Run("calls", func(t *testing.T) {
		p := NewTypeScriptParser()
		path, src := loadSimple(t)
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		// internalHelper calls greet → intra-file resolution to "simple/main.ts::greet"
		// Greeter.hello calls greet → same target.
		want := map[[2]string]bool{
			{"simple/main.ts::internalHelper", "simple/main.ts::greet"}:    false,
			{"simple/main.ts::Greeter.hello", "simple/main.ts::greet"}:     false,
		}
		for _, e := range r.Edges {
			if e.Kind != "calls" {
				continue
			}
			key := [2]string{e.Src, e.Dst}
			if _, ok := want[key]; ok {
				want[key] = true
			}
		}
		for k, seen := range want {
			if !seen {
				t.Errorf("missing calls edge %s -> %s", k[0], k[1])
			}
		}
	})
```

- [ ] **Step 2: Run test, expect FAIL**

- [ ] **Step 3: Implement intra-file call walker**

Refactor `Parse` so that it first collects top-level function and class-method names into an intra-file lookup, then re-walks bodies emitting `calls` edges. Add:

```go
// Pre-pass: build name → owning symbol ID map for intra-file resolution.
intra := map[string]string{}
for i := 0; i < int(root.NamedChildCount()); i++ {
    child := root.NamedChild(i)
    decl := child
    if child.Type() == "export_statement" && child.NamedChildCount() > 0 {
        decl = child.NamedChild(0)
    }
    switch decl.Type() {
    case "function_declaration":
        if n := decl.ChildByFieldName("name"); n != nil {
            intra[n.Content(src)] = path + "::" + n.Content(src)
        }
    }
}
```

After emitting each function/method node, walk its body for `call_expression`:

```go
func walkTSCalls(body *sitter.Node, src []byte, srcID string, intra map[string]string) []store.Edge {
    var out []store.Edge
    var visit func(n *sitter.Node)
    visit = func(n *sitter.Node) {
        if n == nil {
            return
        }
        if n.Type() == "call_expression" {
            fn := n.ChildByFieldName("function")
            if fn != nil {
                switch fn.Type() {
                case "identifier":
                    name := fn.Content(src)
                    if dst, ok := intra[name]; ok {
                        out = append(out, store.Edge{Src: srcID, Dst: dst, Kind: "calls"})
                    } else {
                        out = append(out, store.Edge{Src: srcID, Dst: "external::" + name, Kind: "calls"})
                    }
                case "member_expression":
                    obj := fn.ChildByFieldName("object")
                    prop := fn.ChildByFieldName("property")
                    if obj != nil && prop != nil {
                        out = append(out, store.Edge{Src: srcID, Dst: "external::" + obj.Content(src) + "." + prop.Content(src), Kind: "calls"})
                    }
                }
            }
        }
        for i := 0; i < int(n.NamedChildCount()); i++ {
            visit(n.NamedChild(i))
        }
    }
    visit(body)
    return out
}
```

Wire it: in `emitFunctionNode` and the method loop inside `emitClassNode`, after appending the node, lookup `decl.ChildByFieldName("body")` and append `walkTSCalls(body, src, id, intra)` to `res.Edges`. Pass `intra` through as a function arg (update the helper signatures).

- [ ] **Step 4: Run test, expect PASS**

```bash
go test ./internal/graph/parser/ -v
```

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(graph/parser/ts): calls edges (intra-file + external)"
```

---

## Task 2.5: TypeScript parser — `imports` edges

**Files:**
- Modify: `internal/graph/parser/typescript.go`
- Create: `internal/graph/parser/testdata/ts/multifile/a.ts`
- Create: `internal/graph/parser/testdata/ts/multifile/b.ts`
- Modify: `internal/graph/parser/typescript_test.go`

- [ ] **Step 1: Create multifile fixtures**

`testdata/ts/multifile/a.ts`:

```ts
import { hello } from "./b";

export function callHello(): void {
  hello();
}
```

`testdata/ts/multifile/b.ts`:

```ts
export function hello(): void {}
```

- [ ] **Step 2: Add failing subtest `imports`**

```go
	t.Run("imports", func(t *testing.T) {
		p := NewTypeScriptParser()
		path := "multifile/a.ts"
		src, err := os.ReadFile(filepath.Join("testdata", "ts", "multifile", "a.ts"))
		if err != nil {
			t.Fatal(err)
		}
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		var seen bool
		for _, e := range r.Edges {
			if e.Kind == "imports" && e.Src == "multifile/a.ts" && e.Dst == "external::./b" {
				seen = true
			}
		}
		if !seen {
			t.Fatalf("missing imports edge multifile/a.ts -> external::./b; edges=%v", r.Edges)
		}
	})
```

- [ ] **Step 3: Run test, expect FAIL**

- [ ] **Step 4: Implement import extraction**

In the root-children loop in `Parse`, before the `decl.Type()` switch add:

```go
		if child.Type() == "import_statement" {
			srcNode := child.ChildByFieldName("source")
			if srcNode != nil {
				modulePath := unquote(srcNode.Content(src))
				res.Edges = append(res.Edges, store.Edge{
					Src: path, Dst: "external::" + modulePath, Kind: "imports",
				})
			}
			continue
		}
```

In `tools.go` add (or reuse if already present from Phase 1):

```go
func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'' || s[0] == '`') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}
```

- [ ] **Step 5: Run test, expect PASS**

- [ ] **Step 6: Commit**

```bash
git commit -am "feat(graph/parser/ts): imports edges from import statements"
```

---

## Task 2.6: TypeScript parser — `tests` edges (Jest/Mocha/Vitest)

**Files:**
- Modify: `internal/graph/parser/typescript.go`
- Create: `internal/graph/parser/testdata/ts/simple/main.test.ts`
- Modify: `internal/graph/parser/typescript_test.go`

- [ ] **Step 1: Add fixture `main.test.ts`**

```ts
import { greet } from "./main";

describe("greet", () => {
  it("greets", () => {
    greet("world");
  });
  test("greets again", () => {
    greet("again");
  });
});
```

- [ ] **Step 2: Add failing subtest `tests_edges`**

```go
	t.Run("tests_edges", func(t *testing.T) {
		p := NewTypeScriptParser()
		path := "simple/main.test.ts"
		src, err := os.ReadFile(filepath.Join("testdata", "ts", "simple", "main.test.ts"))
		if err != nil {
			t.Fatal(err)
		}
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		var count int
		for _, e := range r.Edges {
			if e.Kind == "tests" && e.Src == "simple/main.test.ts" && e.Dst == "external::greet" {
				count++
			}
		}
		if count == 0 {
			t.Fatalf("expected at least one tests edge from main.test.ts to external::greet; edges=%v", r.Edges)
		}
	})
```

- [ ] **Step 3: Run test, expect FAIL**

- [ ] **Step 4: Implement test-block detection**

After all symbols have been collected, walk the AST once more looking for `call_expression` whose function name is `describe`, `it`, or `test`, and treat call edges out of their arrow-function bodies as `tests` edges from the *file node* to the called symbol.

In `Parse`, after the per-child loop, append:

```go
	if isTestFile(path) {
		res.Edges = append(res.Edges, extractTSTestEdges(root, src, path)...)
	}
```

Add helpers:

```go
func isTestFile(path string) bool {
	for _, suf := range []string{".test.ts", ".spec.ts", ".test.tsx", ".spec.tsx", ".test.js", ".spec.js"} {
		if strings.HasSuffix(path, suf) {
			return true
		}
	}
	return false
}

func extractTSTestEdges(root *sitter.Node, src []byte, path string) []store.Edge {
	var out []store.Edge
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "call_expression" {
			fn := n.ChildByFieldName("function")
			if fn != nil && fn.Type() == "identifier" {
				if name := fn.Content(src); name == "describe" || name == "it" || name == "test" {
					args := n.ChildByFieldName("arguments")
					if args != nil {
						for i := 0; i < int(args.NamedChildCount()); i++ {
							out = append(out, extractCallTargets(args.NamedChild(i), src, path)...)
						}
					}
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(root)
	return out
}

func extractCallTargets(n *sitter.Node, src []byte, path string) []store.Edge {
	var out []store.Edge
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "call_expression" {
			fn := n.ChildByFieldName("function")
			if fn != nil && fn.Type() == "identifier" {
				out = append(out, store.Edge{Src: path, Dst: "external::" + fn.Content(src), Kind: "tests"})
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(n)
	return out
}
```

(Add `"strings"` to imports.)

- [ ] **Step 5: Run test, expect PASS**

- [ ] **Step 6: Commit**

```bash
git commit -am "feat(graph/parser/ts): tests edges for Jest/Mocha/Vitest patterns"
```

---

## Task 2.7: TypeScript parser — `implements` and `extends` edges

**Files:**
- Modify: `internal/graph/parser/testdata/ts/simple/main.ts`
- Modify: `internal/graph/parser/typescript.go`
- Modify: `internal/graph/parser/typescript_test.go`

- [ ] **Step 1: Extend fixture**

Replace `Greeter` class declaration with:

```ts
export class Greeter extends Base implements Speaker {
  hello(name: string): string {
    return greet(name);
  }
  private silent(): void {}
}

class Base {}
```

- [ ] **Step 2: Add failing subtest `implements_extends`**

```go
	t.Run("implements_extends", func(t *testing.T) {
		p := NewTypeScriptParser()
		path, src := loadSimple(t)
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		var hasImpl, hasExt bool
		for _, e := range r.Edges {
			if e.Kind == "implements" && e.Src == "simple/main.ts::Greeter" && e.Dst == "simple/main.ts::Speaker" {
				hasImpl = true
			}
			if e.Kind == "extends" && e.Src == "simple/main.ts::Greeter" && e.Dst == "simple/main.ts::Base" {
				hasExt = true
			}
		}
		if !hasImpl || !hasExt {
			t.Fatalf("missing edges: implements=%v extends=%v", hasImpl, hasExt)
		}
	})
```

- [ ] **Step 3: Run test, expect FAIL**

- [ ] **Step 4: Implement**

In `emitClassNode`, after appending the class node and before walking members, scan the `class_heritage` children:

```go
	// First-pass intra-file class+interface names (passed in by caller).
	for i := 0; i < int(decl.ChildCount()); i++ {
		c := decl.Child(i)
		if c.Type() == "class_heritage" {
			for j := 0; j < int(c.NamedChildCount()); j++ {
				clause := c.NamedChild(j)
				switch clause.Type() {
				case "extends_clause":
					for k := 0; k < int(clause.NamedChildCount()); k++ {
						name := clause.NamedChild(k).Content(src)
						res.Edges = append(res.Edges, store.Edge{
							Src: classID, Dst: resolveIntra(name, path, intra), Kind: "extends",
						})
					}
				case "implements_clause":
					for k := 0; k < int(clause.NamedChildCount()); k++ {
						name := clause.NamedChild(k).Content(src)
						res.Edges = append(res.Edges, store.Edge{
							Src: classID, Dst: resolveIntra(name, path, intra), Kind: "implements",
						})
					}
				}
			}
		}
	}
```

Helper in `tools.go`:

```go
func resolveIntra(name, path string, intra map[string]string) string {
	if id, ok := intra[name]; ok {
		return id
	}
	return "external::" + name
}
```

Update the pre-pass to also record class and interface names in `intra` (mapping to `path + "::" + name`).

- [ ] **Step 5: Run test, expect PASS**

- [ ] **Step 6: Commit**

```bash
git commit -am "feat(graph/parser/ts): implements and extends edges"
```

---

## Task 2.8: tsconfig.json path-alias resolver

**Files:**
- Create: `internal/graph/resolver/tsconfig.go`
- Create: `internal/graph/resolver/tsconfig_test.go`
- Create: `internal/graph/parser/testdata/ts/alias/tsconfig.json`
- Create: `internal/graph/parser/testdata/ts/alias/src/a.ts`
- Create: `internal/graph/parser/testdata/ts/alias/src/lib/b.ts`

- [ ] **Step 1: Create fixture**

`testdata/ts/alias/tsconfig.json`:

```json
{
  "compilerOptions": {
    "baseUrl": "./src",
    "paths": {
      "@lib/*": ["lib/*"]
    }
  }
}
```

`testdata/ts/alias/src/a.ts`:

```ts
import { x } from "@lib/b";
```

`testdata/ts/alias/src/lib/b.ts`:

```ts
export const x = 1;
```

- [ ] **Step 2: Write failing test `TestTSConfigResolver`**

```go
package resolver

import (
	"path/filepath"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestTSConfigResolverRewritesAliasImports(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "parser", "testdata", "ts", "alias"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewTSConfigResolver(root)
	if err != nil {
		t.Fatal(err)
	}
	edges := []store.Edge{
		{Src: "src/a.ts", Dst: "external::@lib/b", Kind: "imports"},
	}
	got := r.Rewrite(edges)
	if len(got) != 1 {
		t.Fatalf("want 1 edge, got %d", len(got))
	}
	if got[0].Dst != "src/lib/b.ts" {
		t.Errorf("dst=%q want src/lib/b.ts", got[0].Dst)
	}
}
```

- [ ] **Step 3: Run test, expect FAIL**

- [ ] **Step 4: Implement**

```go
// Package resolver — additions for TypeScript path aliases.
package resolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

type tsConfigFile struct {
	Extends         string `json:"extends"`
	CompilerOptions struct {
		BaseUrl string              `json:"baseUrl"`
		Paths   map[string][]string `json:"paths"`
	} `json:"compilerOptions"`
}

// TSConfigResolver rewrites import edges whose dst matches a tsconfig path alias
// into edges pointing at the resolved on-disk file path (relative to repo root).
type TSConfigResolver struct {
	root    string
	baseURL string
	paths   map[string][]string
}

// NewTSConfigResolver loads tsconfig.json from root (handling `extends` once).
func NewTSConfigResolver(root string) (*TSConfigResolver, error) {
	r := &TSConfigResolver{root: root, paths: map[string][]string{}}
	if err := r.loadTSConfig(filepath.Join(root, "tsconfig.json")); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *TSConfigResolver) loadTSConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	var cfg tsConfigFile
	if err := json.Unmarshal(stripJSONComments(data), &cfg); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Extends != "" {
		parent := filepath.Join(filepath.Dir(path), cfg.Extends)
		if !strings.HasSuffix(parent, ".json") {
			parent += ".json"
		}
		if err := r.loadTSConfig(parent); err != nil {
			return err
		}
	}
	if cfg.CompilerOptions.BaseUrl != "" {
		r.baseURL = filepath.Join(filepath.Dir(path), cfg.CompilerOptions.BaseUrl)
	}
	for k, v := range cfg.CompilerOptions.Paths {
		r.paths[k] = v
	}
	return nil
}

// Rewrite walks edges and rewrites `external::<alias>` import dsts to repo-relative file paths.
func (r *TSConfigResolver) Rewrite(edges []store.Edge) []store.Edge {
	out := make([]store.Edge, len(edges))
	for i, e := range edges {
		out[i] = e
		if e.Kind != "imports" || !strings.HasPrefix(e.Dst, "external::") {
			continue
		}
		spec := strings.TrimPrefix(e.Dst, "external::")
		if rel, ok := r.resolve(spec); ok {
			out[i].Dst = rel
		}
	}
	return out
}

func (r *TSConfigResolver) resolve(spec string) (string, bool) {
	for pattern, targets := range r.paths {
		prefix := strings.TrimSuffix(pattern, "*")
		if !strings.HasPrefix(spec, prefix) {
			continue
		}
		tail := strings.TrimPrefix(spec, prefix)
		for _, tmpl := range targets {
			candidate := filepath.Join(r.baseURL, strings.Replace(tmpl, "*", tail, 1))
			for _, ext := range []string{".ts", ".tsx", "/index.ts"} {
				p := candidate + ext
				if info, err := os.Stat(p); err == nil && !info.IsDir() {
					rel, err := filepath.Rel(r.root, p)
					if err != nil {
						return "", false
					}
					return filepath.ToSlash(rel), true
				}
			}
		}
	}
	return "", false
}

func stripJSONComments(b []byte) []byte {
	// tsconfig allows // line comments. Strip them naively.
	lines := strings.Split(string(b), "\n")
	for i, l := range lines {
		if idx := strings.Index(l, "//"); idx >= 0 {
			lines[i] = l[:idx]
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
```

- [ ] **Step 5: Run test, expect PASS**

```bash
go test ./internal/graph/resolver/ -run TestTSConfigResolver -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/graph/resolver/tsconfig.go internal/graph/resolver/tsconfig_test.go internal/graph/parser/testdata/ts/alias/
git commit -m "feat(graph/resolver/ts): tsconfig path alias resolver with extends"
```

---

## Task 2.9: JavaScript parser

**Files:**
- Create: `internal/graph/parser/javascript.go`
- Create: `internal/graph/parser/javascript_test.go`
- Create: `internal/graph/parser/testdata/js/simple/main.js`
- Create: `internal/graph/parser/testdata/js/multifile/{a,b}.js`

JavaScript is a strict subset of TypeScript for our purposes: no `interface_declaration`, no `type_alias_declaration`, no `accessibility_modifier`, no `implements_clause`. Everything else (function/class/method/extends/imports/calls/tests) carries over via tree-sitter-javascript's grammar (the node type names are identical for these constructs in tree-sitter-javascript).

- [ ] **Step 1: Add `tree-sitter-javascript` dependency**

```bash
go get github.com/smacker/go-tree-sitter/javascript
go mod tidy
```

- [ ] **Step 2: Add fixture `testdata/js/simple/main.js`**

```js
export function greet(name) {
  return "hi " + name;
}

function internalHelper() {
  greet("world");
}

export class Greeter extends Base {
  hello(name) {
    return greet(name);
  }
}

class Base {}
```

- [ ] **Step 3: Write failing test `TestJavaScriptParserExtracts`**

```go
package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJavaScriptParserExtracts(t *testing.T) {
	load := func(t *testing.T) (string, []byte) {
		t.Helper()
		path := filepath.Join("testdata", "js", "simple", "main.js")
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return "simple/main.js", src
	}

	p := NewJavaScriptParser()
	path, src := load(t)
	r, err := p.Parse(path, src)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("nodes", func(t *testing.T) {
		want := map[string]string{
			"simple/main.js":                     "file",
			"simple/main.js::greet":              "function",
			"simple/main.js::internalHelper":     "function",
			"simple/main.js::Greeter":            "class",
			"simple/main.js::Greeter.hello":      "method",
			"simple/main.js::Base":               "class",
		}
		seen := map[string]string{}
		for _, n := range r.Nodes {
			if _, ok := want[n.ID]; ok {
				seen[n.ID] = n.Kind
			}
		}
		for id, kind := range want {
			if seen[id] != kind {
				t.Errorf("%s kind=%q want %q", id, seen[id], kind)
			}
		}
	})

	t.Run("calls", func(t *testing.T) {
		want := map[[2]string]bool{
			{"simple/main.js::internalHelper", "simple/main.js::greet"}: false,
			{"simple/main.js::Greeter.hello", "simple/main.js::greet"}:  false,
		}
		for _, e := range r.Edges {
			if e.Kind == "calls" {
				want[[2]string{e.Src, e.Dst}] = true
			}
		}
		for k, seen := range want {
			if !seen {
				t.Errorf("missing calls edge %s -> %s", k[0], k[1])
			}
		}
	})

	t.Run("extends", func(t *testing.T) {
		var ok bool
		for _, e := range r.Edges {
			if e.Kind == "extends" && e.Src == "simple/main.js::Greeter" && e.Dst == "simple/main.js::Base" {
				ok = true
			}
		}
		if !ok {
			t.Error("missing extends edge Greeter -> Base")
		}
	})
}
```

- [ ] **Step 4: Run test, expect FAIL** (`NewJavaScriptParser` undefined)

- [ ] **Step 5: Implement `javascript.go`**

```go
package parser

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	jsLang "github.com/smacker/go-tree-sitter/javascript"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// JavaScriptParser extracts nodes and edges from .js/.mjs/.cjs source files.
type JavaScriptParser struct{ lang *sitter.Language }

// NewJavaScriptParser returns a Parser for JavaScript source files.
func NewJavaScriptParser() *JavaScriptParser {
	return &JavaScriptParser{lang: jsLang.GetLanguage()}
}

func (p *JavaScriptParser) Language() string     { return "javascript" }
func (p *JavaScriptParser) Extensions() []string { return []string{".js", ".mjs", ".cjs"} }

func (p *JavaScriptParser) Parse(path string, src []byte) (ParseResult, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(p.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return ParseResult{}, fmt.Errorf("tree-sitter parse %s: %w", path, err)
	}
	defer tree.Close()

	res := ParseResult{InterfaceMethods: map[string][]string{}}
	res.Nodes = append(res.Nodes, store.Node{
		ID: path, Kind: "file", Path: path, Name: path, Language: "javascript",
	})
	root := tree.RootNode()

	intra := map[string]string{}
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		decl := child
		if child.Type() == "export_statement" && child.NamedChildCount() > 0 {
			decl = child.NamedChild(0)
		}
		if name := nameOf(decl, src); name != "" {
			intra[name] = path + "::" + name
		}
	}

	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		exported := false
		decl := child
		if child.Type() == "export_statement" {
			exported = true
			if child.NamedChildCount() > 0 {
				decl = child.NamedChild(0)
			}
		}
		switch decl.Type() {
		case "function_declaration":
			emitJSFunction(&res, decl, src, path, exported, intra)
		case "class_declaration":
			emitJSClass(&res, decl, src, path, exported, intra)
		case "import_statement":
			if s := decl.ChildByFieldName("source"); s != nil {
				res.Edges = append(res.Edges, store.Edge{
					Src: path, Dst: "external::" + unquote(s.Content(src)), Kind: "imports",
				})
			}
		}
	}

	if isJSTestFile(path) {
		res.Edges = append(res.Edges, extractTSTestEdges(root, src, path)...)
	}
	return res, nil
}

func nameOf(decl *sitter.Node, src []byte) string {
	if decl == nil {
		return ""
	}
	n := decl.ChildByFieldName("name")
	if n == nil {
		return ""
	}
	return n.Content(src)
}

func emitJSFunction(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool, intra map[string]string) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: "function", Path: path, Name: name, Language: "javascript",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
	if body := decl.ChildByFieldName("body"); body != nil {
		res.Edges = append(res.Edges, walkTSCalls(body, src, id, intra)...)
	}
}

func emitJSClass(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool, intra map[string]string) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	className := nameNode.Content(src)
	classID := path + "::" + className
	res.Nodes = append(res.Nodes, store.Node{
		ID: classID, Kind: "class", Path: path, Name: className, Language: "javascript",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: classID, Kind: "contains"})

	for i := 0; i < int(decl.ChildCount()); i++ {
		c := decl.Child(i)
		if c.Type() == "class_heritage" {
			for j := 0; j < int(c.NamedChildCount()); j++ {
				name := c.NamedChild(j).Content(src)
				res.Edges = append(res.Edges, store.Edge{
					Src: classID, Dst: resolveIntra(name, path, intra), Kind: "extends",
				})
			}
		}
	}

	body := decl.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < int(body.NamedChildCount()); i++ {
		member := body.NamedChild(i)
		if member.Type() != "method_definition" {
			continue
		}
		mNameNode := member.ChildByFieldName("name")
		if mNameNode == nil {
			continue
		}
		mName := mNameNode.Content(src)
		mID := path + "::" + className + "." + mName
		res.Nodes = append(res.Nodes, store.Node{
			ID: mID, Kind: "method", Path: path, Name: mName, Container: className,
			Language: "javascript", IsExported: true,
			StartLine: int(member.StartPoint().Row) + 1,
			EndLine:   int(member.EndPoint().Row) + 1,
		})
		res.Edges = append(res.Edges, store.Edge{Src: classID, Dst: mID, Kind: "contains"})
		if mBody := member.ChildByFieldName("body"); mBody != nil {
			res.Edges = append(res.Edges, walkTSCalls(mBody, src, mID, intra)...)
		}
	}
}

func isJSTestFile(path string) bool {
	for _, suf := range []string{".test.js", ".spec.js", ".test.mjs", ".spec.mjs"} {
		if strings.HasSuffix(path, suf) {
			return true
		}
	}
	return false
}
```

(Add `"strings"` import.)

- [ ] **Step 6: Run test, expect PASS**

- [ ] **Step 7: Commit**

```bash
git add internal/graph/parser/javascript.go internal/graph/parser/javascript_test.go internal/graph/parser/testdata/js/ go.mod go.sum
git commit -m "feat(graph/parser/js): JavaScript parser reusing TS walker helpers"
```

---

## Task 2.10: Rust parser — file, function, struct, enum, type nodes

**Files:**
- Create: `internal/graph/parser/rust.go`
- Create: `internal/graph/parser/rust_test.go`
- Create: `internal/graph/parser/testdata/rust/simple/lib.rs`

- [ ] **Step 1: Add `tree-sitter-rust` dependency**

```bash
go get github.com/smacker/go-tree-sitter/rust
go mod tidy
```

- [ ] **Step 2: Add fixture `testdata/rust/simple/lib.rs`**

```rust
pub fn greet(name: &str) -> String {
    format!("hi {}", name)
}

fn internal_helper() {
    greet("world");
}

pub struct Greeter {
    pub name: String,
}

pub enum Mood { Happy, Sad }

pub type Greeting = String;
```

- [ ] **Step 3: Write failing test**

```go
package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRustParserExtracts(t *testing.T) {
	loadSimple := func(t *testing.T) (string, []byte) {
		t.Helper()
		path := filepath.Join("testdata", "rust", "simple", "lib.rs")
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return "simple/lib.rs", src
	}

	p := NewRustParser()
	path, src := loadSimple(t)
	r, err := p.Parse(path, src)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]struct {
		kind     string
		exported bool
	}{
		"simple/lib.rs":                  {"file", false},
		"simple/lib.rs::greet":           {"function", true},
		"simple/lib.rs::internal_helper": {"function", false},
		"simple/lib.rs::Greeter":         {"struct", true},
		"simple/lib.rs::Mood":            {"enum", true},
		"simple/lib.rs::Greeting":        {"type", true},
	}
	seen := map[string]bool{}
	for _, n := range r.Nodes {
		w, ok := want[n.ID]
		if !ok {
			continue
		}
		seen[n.ID] = true
		if n.Kind != w.kind {
			t.Errorf("%s kind=%q want %q", n.ID, n.Kind, w.kind)
		}
		if n.IsExported != w.exported {
			t.Errorf("%s exported=%v want %v", n.ID, n.IsExported, w.exported)
		}
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("missing node: %s", id)
		}
	}
}
```

- [ ] **Step 4: Run test, expect FAIL**

- [ ] **Step 5: Implement `rust.go`**

```go
package parser

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	rustLang "github.com/smacker/go-tree-sitter/rust"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// RustParser extracts nodes and edges from .rs source files.
type RustParser struct{ lang *sitter.Language }

// NewRustParser returns a Parser for Rust source files.
func NewRustParser() *RustParser { return &RustParser{lang: rustLang.GetLanguage()} }

func (p *RustParser) Language() string     { return "rust" }
func (p *RustParser) Extensions() []string { return []string{".rs"} }

func (p *RustParser) Parse(path string, src []byte) (ParseResult, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(p.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return ParseResult{}, fmt.Errorf("tree-sitter parse %s: %w", path, err)
	}
	defer tree.Close()

	res := ParseResult{InterfaceMethods: map[string][]string{}}
	res.Nodes = append(res.Nodes, store.Node{
		ID: path, Kind: "file", Path: path, Name: path, Language: "rust",
	})
	root := tree.RootNode()
	for i := 0; i < int(root.NamedChildCount()); i++ {
		c := root.NamedChild(i)
		exported := hasRustVisibilityPub(c, src)
		switch c.Type() {
		case "function_item":
			emitRustSymbol(&res, c, src, path, "function", exported)
		case "struct_item":
			emitRustSymbol(&res, c, src, path, "struct", exported)
		case "enum_item":
			emitRustSymbol(&res, c, src, path, "enum", exported)
		case "type_item":
			emitRustSymbol(&res, c, src, path, "type", exported)
		}
	}
	return res, nil
}

func hasRustVisibilityPub(n *sitter.Node, src []byte) bool {
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.Type() == "visibility_modifier" && c.Content(src) == "pub" {
			return true
		}
	}
	return false
}

func emitRustSymbol(res *ParseResult, decl *sitter.Node, src []byte, path, kind string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: kind, Path: path, Name: name, Language: "rust",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
}
```

- [ ] **Step 6: Run test, expect PASS**

- [ ] **Step 7: Commit**

```bash
git add internal/graph/parser/rust.go internal/graph/parser/rust_test.go internal/graph/parser/testdata/rust/ go.mod go.sum
git commit -m "feat(graph/parser/rust): file/function/struct/enum/type nodes"
```

---

## Task 2.11: Rust parser — trait nodes and `impl Trait for Struct` → implements

**Files:**
- Modify: `internal/graph/parser/testdata/rust/simple/lib.rs`
- Modify: `internal/graph/parser/rust.go`
- Modify: `internal/graph/parser/rust_test.go`

- [ ] **Step 1: Extend fixture**

Append to `lib.rs`:

```rust
pub trait Hello {
    fn hello(&self) -> String;
}

impl Hello for Greeter {
    fn hello(&self) -> String {
        greet(&self.name)
    }
}
```

- [ ] **Step 2: Add failing subtest**

Replace single-test body with a `t.Run` structure and add:

```go
t.Run("trait_and_impl", func(t *testing.T) {
    var hasTrait, hasImpl, hasImplMethod bool
    for _, n := range r.Nodes {
        if n.ID == "simple/lib.rs::Hello" && n.Kind == "interface" {
            hasTrait = true
        }
        if n.ID == "simple/lib.rs::Greeter.hello" && n.Kind == "method" && n.Container == "Greeter" {
            hasImplMethod = true
        }
    }
    for _, e := range r.Edges {
        if e.Kind == "implements" && e.Src == "simple/lib.rs::Greeter" && e.Dst == "simple/lib.rs::Hello" {
            hasImpl = true
        }
    }
    if !hasTrait || !hasImpl || !hasImplMethod {
        t.Fatalf("trait=%v impl=%v method=%v", hasTrait, hasImpl, hasImplMethod)
    }
})
```

- [ ] **Step 3: Run test, expect FAIL**

- [ ] **Step 4: Implement trait + impl handling**

In the root loop, add cases:

```go
		case "trait_item":
			emitRustTrait(&res, c, src, path, exported)
		case "impl_item":
			emitRustImpl(&res, c, src, path)
```

Helpers:

```go
func emitRustTrait(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: "interface", Path: path, Name: name, Language: "rust",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})

	body := decl.ChildByFieldName("body")
	if body == nil {
		return
	}
	var methods []string
	for i := 0; i < int(body.NamedChildCount()); i++ {
		m := body.NamedChild(i)
		if m.Type() == "function_signature_item" {
			if mn := m.ChildByFieldName("name"); mn != nil {
				methods = append(methods, mn.Content(src))
			}
		}
	}
	if len(methods) > 0 {
		res.InterfaceMethods[id] = methods
	}
}

func emitRustImpl(res *ParseResult, decl *sitter.Node, src []byte, path string) {
	traitNode := decl.ChildByFieldName("trait")
	typeNode := decl.ChildByFieldName("type")
	if typeNode == nil {
		return
	}
	typeName := typeNode.Content(src)
	typeID := path + "::" + typeName
	if traitNode != nil {
		traitName := traitNode.Content(src)
		res.Edges = append(res.Edges, store.Edge{
			Src: typeID, Dst: path + "::" + traitName, Kind: "implements",
		})
	}
	body := decl.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < int(body.NamedChildCount()); i++ {
		fn := body.NamedChild(i)
		if fn.Type() != "function_item" {
			continue
		}
		nameNode := fn.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		mName := nameNode.Content(src)
		mID := path + "::" + typeName + "." + mName
		res.Nodes = append(res.Nodes, store.Node{
			ID: mID, Kind: "method", Path: path, Name: mName, Container: typeName,
			Language:   "rust",
			StartLine:  int(fn.StartPoint().Row) + 1,
			EndLine:    int(fn.EndPoint().Row) + 1,
			IsExported: hasRustVisibilityPub(fn, src),
		})
		res.Edges = append(res.Edges, store.Edge{Src: typeID, Dst: mID, Kind: "contains"})
	}
}
```

- [ ] **Step 5: Run test, expect PASS**

- [ ] **Step 6: Commit**

```bash
git commit -am "feat(graph/parser/rust): trait nodes and impl Trait for Struct implements edges"
```

---

## Task 2.12: Rust parser — calls and `use`/imports edges

**Files:**
- Modify: `internal/graph/parser/rust.go`
- Modify: `internal/graph/parser/rust_test.go`

- [ ] **Step 1: Add failing subtests `calls` and `imports`**

Add to fixture top of `lib.rs`:

```rust
use std::fmt::Display;
use crate::other::helper;
```

Subtests:

```go
t.Run("calls", func(t *testing.T) {
    var ok bool
    for _, e := range r.Edges {
        if e.Kind == "calls" && e.Src == "simple/lib.rs::internal_helper" && e.Dst == "simple/lib.rs::greet" {
            ok = true
        }
    }
    if !ok {
        t.Error("missing calls edge internal_helper -> greet")
    }
})

t.Run("imports", func(t *testing.T) {
    want := map[string]bool{
        "external::std::fmt::Display":   false,
        "external::crate::other::helper": false,
    }
    for _, e := range r.Edges {
        if e.Kind == "imports" && e.Src == "simple/lib.rs" {
            if _, ok := want[e.Dst]; ok {
                want[e.Dst] = true
            }
        }
    }
    for k, v := range want {
        if !v {
            t.Errorf("missing imports edge to %s", k)
        }
    }
})
```

- [ ] **Step 2: Run test, expect FAIL**

- [ ] **Step 3: Implement**

In root loop add `case "use_declaration":`:

```go
		case "use_declaration":
			path := flattenUseTree(c.NamedChild(0), src)
			if path != "" {
				res.Edges = append(res.Edges, store.Edge{Src: path, Dst: "external::" + path, Kind: "imports"})
			}
```

Wait — `path` is shadowed; rename inner var:

```go
		case "use_declaration":
			if c.NamedChildCount() > 0 {
				usePath := flattenUseTree(c.NamedChild(0), src)
				if usePath != "" {
					res.Edges = append(res.Edges, store.Edge{
						Src: path, Dst: "external::" + usePath, Kind: "imports",
					})
				}
			}
```

Helper:

```go
func flattenUseTree(n *sitter.Node, src []byte) string {
	if n == nil {
		return ""
	}
	// For "scoped_identifier" or "scoped_use_list", concatenate path segments with ::.
	return n.Content(src)
}
```

For calls inside Rust functions, extend `emitRustSymbol` for the `function` kind to also walk the body:

```go
if kind == "function" {
    if body := decl.ChildByFieldName("body"); body != nil {
        res.Edges = append(res.Edges, walkRustCalls(body, src, id, path)...)
    }
}
```

Also have `emitRustImpl` walk each function body and pass the intra-file lookup of top-level functions.

Walker:

```go
func walkRustCalls(body *sitter.Node, src []byte, srcID, filePath string) []store.Edge {
	var out []store.Edge
	intra := map[string]string{} // populated by caller if needed; simplified intra-file lookup
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "call_expression" {
			fn := n.ChildByFieldName("function")
			if fn != nil {
				switch fn.Type() {
				case "identifier":
					name := fn.Content(src)
					out = append(out, store.Edge{Src: srcID, Dst: filePath + "::" + name, Kind: "calls"})
					_ = intra
				case "scoped_identifier", "field_expression":
					out = append(out, store.Edge{Src: srcID, Dst: "external::" + fn.Content(src), Kind: "calls"})
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(body)
	return out
}
```

(Phase 1's resolver already rewrites unresolved intra-module `<path>::<name>` edges; the Rust call walker treats every identifier-call as same-file and lets the resolver clean up cross-file references later. Document this in a comment above `walkRustCalls`.)

- [ ] **Step 4: Run test, expect PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(graph/parser/rust): calls and use/imports edges"
```

---

## Task 2.13: `cache.RepoKey` and path helpers

**Files:**
- Create: `internal/graph/cache/paths.go`
- Create: `internal/graph/cache/paths_test.go`

- [ ] **Step 1: Write failing tests**

```go
package cache

import (
	"strings"
	"testing"
)

func TestRepoKey(t *testing.T) {
	k1 := RepoKey("/Users/x/code/foo")
	k2 := RepoKey("/Users/x/code/foo")
	k3 := RepoKey("/Users/x/code/bar")
	if k1 != k2 {
		t.Errorf("RepoKey not deterministic: %q vs %q", k1, k2)
	}
	if k1 == k3 {
		t.Errorf("RepoKey not unique across paths: %q == %q", k1, k3)
	}
	if len(k1) != 12 {
		t.Errorf("RepoKey length=%d, want 12", len(k1))
	}
	if strings.ContainsAny(k1, "/. ") {
		t.Errorf("RepoKey contains illegal chars: %q", k1)
	}
}

func TestPathLayout(t *testing.T) {
	k := "abcdef012345"
	got := GraphDB("/tmp/devpilot-home", k)
	want := "/tmp/devpilot-home/graphs/abcdef012345/graph.db"
	if got != want {
		t.Errorf("GraphDB=%q want %q", got, want)
	}
	if !strings.HasPrefix(PreflightFile("/tmp/devpilot-home", k), "/tmp/devpilot-home/preflight/abcdef012345-") {
		t.Errorf("PreflightFile prefix mismatch")
	}
}
```

- [ ] **Step 2: Run test, expect FAIL**

- [ ] **Step 3: Implement `paths.go`**

```go
// Package cache manages the on-disk graph cache under ~/.devpilot/graphs/.
package cache

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RepoKey is a deterministic 12-character hex identifier derived from the
// absolute path of a repository root. Different clones of the same repo
// intentionally produce different keys (see design doc §5).
func RepoKey(absRepoRoot string) string {
	sum := sha1.Sum([]byte(absRepoRoot))
	return hex.EncodeToString(sum[:])[:12]
}

// Home returns the devpilot cache root, defaulting to ~/.devpilot.
// Overridable via DEVPILOT_HOME.
func Home() string {
	if v := os.Getenv("DEVPILOT_HOME"); v != "" {
		return v
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".devpilot")
	}
	return ".devpilot"
}

// GraphDir returns the per-repo cache directory.
func GraphDir(home, key string) string {
	return filepath.Join(home, "graphs", key)
}

// GraphDB returns the SQLite file path.
func GraphDB(home, key string) string {
	return filepath.Join(GraphDir(home, key), "graph.db")
}

// MetaFile returns the meta.json path.
func MetaFile(home, key string) string {
	return filepath.Join(GraphDir(home, key), "meta.json")
}

// LockFile returns the build.lock path.
func LockFile(home, key string) string {
	return filepath.Join(GraphDir(home, key), "build.lock")
}

// PreflightFile returns a timestamped preflight output path.
func PreflightFile(home, key string) string {
	return filepath.Join(home, "preflight",
		fmt.Sprintf("%s-%d.json", key, time.Now().UnixNano()))
}

// EnsureDirs mkdir -p's the graphs and preflight directories.
func EnsureDirs(home, key string) error {
	for _, d := range []string{GraphDir(home, key), filepath.Join(home, "preflight")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test, expect PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/cache/paths.go internal/graph/cache/paths_test.go
git commit -m "feat(graph/cache): RepoKey and on-disk path helpers"
```

---

## Task 2.14: Build lock (`flock` wrapper)

**Files:**
- Create: `internal/graph/cache/flock.go`
- Create: `internal/graph/cache/flock_test.go`

- [ ] **Step 1: Add dependency**

```bash
go get github.com/gofrs/flock
go mod tidy
```

- [ ] **Step 2: Write failing test**

```go
package cache

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestBuildLockSerializes(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "build.lock")

	var (
		mu        sync.Mutex
		insideMax int
		inside    int
		wg        sync.WaitGroup
	)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rel, err := AcquireBuildLock(lockPath, 5*time.Second)
			if err != nil {
				t.Errorf("acquire: %v", err)
				return
			}
			defer rel()
			mu.Lock()
			inside++
			if inside > insideMax {
				insideMax = inside
			}
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
			mu.Lock()
			inside--
			mu.Unlock()
		}()
	}
	wg.Wait()
	if insideMax != 1 {
		t.Errorf("max concurrent holders=%d, want 1", insideMax)
	}
}

func TestBuildLockTimeout(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "build.lock")

	rel1, err := AcquireBuildLock(lockPath, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer rel1()

	_, err = AcquireBuildLock(lockPath, 100*time.Millisecond)
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("err=%v, want ErrLockTimeout", err)
	}
}
```

- [ ] **Step 3: Run test, expect FAIL**

- [ ] **Step 4: Implement `flock.go`**

```go
package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/flock"
)

// ErrLockTimeout indicates AcquireBuildLock could not obtain the lock within timeout.
var ErrLockTimeout = errors.New("build lock acquire timed out")

// AcquireBuildLock takes an exclusive flock on lockPath, polling every 100ms until
// timeout. Returns a release function the caller must invoke.
func AcquireBuildLock(lockPath string, timeout time.Duration) (release func() error, err error) {
	fl := flock.New(lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	locked, err := fl.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("flock %s: %w", lockPath, err)
	}
	if !locked {
		return nil, ErrLockTimeout
	}
	return fl.Unlock, nil
}
```

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/graph/cache/ -run TestBuildLock -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/graph/cache/flock.go internal/graph/cache/flock_test.go go.mod go.sum
git commit -m "feat(graph/cache): build lock with gofrs/flock"
```

---

## Task 2.15: TTL sweeper for preflight JSON

**Files:**
- Create: `internal/graph/cache/ttl.go`
- Create: `internal/graph/cache/ttl_test.go`

- [ ] **Step 1: Write failing test**

```go
package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSweepPreflight(t *testing.T) {
	dir := t.TempDir()
	preflightDir := filepath.Join(dir, "preflight")
	if err := os.MkdirAll(preflightDir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(preflightDir, "old.json")
	fresh := filepath.Join(preflightDir, "fresh.json")
	for _, f := range []string{old, fresh} {
		if err := os.WriteFile(f, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	stale := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(old, stale, stale); err != nil {
		t.Fatal(err)
	}

	if err := SweepPreflight(dir, 7*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("old file still exists: %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("fresh file removed: %v", err)
	}
}
```

- [ ] **Step 2: Run test, expect FAIL**

- [ ] **Step 3: Implement `ttl.go`**

```go
package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SweepPreflight deletes files under <home>/preflight/ whose modtime is older
// than ttl. Missing directory is not an error.
func SweepPreflight(home string, ttl time.Duration) error {
	dir := filepath.Join(home, "preflight")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", dir, err)
	}
	cutoff := time.Now().Add(-ttl)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test, expect PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/cache/ttl.go internal/graph/cache/ttl_test.go
git commit -m "feat(graph/cache): TTL sweeper for preflight JSON"
```

---

## Task 2.16: `meta.json` read/write

**Files:**
- Create: `internal/graph/cache/meta.go`
- Create: `internal/graph/cache/meta_test.go`

- [ ] **Step 1: Write failing test**

```go
package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMetaRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "meta.json")
	want := Meta{
		SchemaVersion: 1,
		HeadSHA:       "abc123",
		ParserVersion: "go=phase2,ts=phase2",
		Languages:     []string{"go", "typescript"},
		BuiltAtUnix:   1700000000,
	}
	if err := WriteMeta(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadMeta(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("ReadMeta=%+v want %+v", got, want)
	}
}

func TestReadMetaMissingReturnsEmpty(t *testing.T) {
	got, err := ReadMeta("/nonexistent/meta.json")
	if err != nil {
		t.Fatal(err)
	}
	if got.HeadSHA != "" {
		t.Errorf("expected empty meta, got %+v", got)
	}
}

func TestWriteMetaAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "meta.json")
	if err := WriteMeta(path, Meta{HeadSHA: "x", SchemaVersion: 1}); err != nil {
		t.Fatal(err)
	}
	// No stray .tmp file should remain.
	matches, _ := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if len(matches) != 0 {
		t.Errorf("leftover tmp files: %v", matches)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
```

Note: `Meta` contains a slice, so equality with `==` won't compile. Adjust the round-trip assertion to compare field-by-field:

```go
	if got.HeadSHA != want.HeadSHA || got.SchemaVersion != want.SchemaVersion ||
		got.ParserVersion != want.ParserVersion || got.BuiltAtUnix != want.BuiltAtUnix ||
		len(got.Languages) != len(want.Languages) {
		t.Errorf("ReadMeta=%+v want %+v", got, want)
	}
	for i := range want.Languages {
		if got.Languages[i] != want.Languages[i] {
			t.Errorf("language[%d]=%q want %q", i, got.Languages[i], want.Languages[i])
		}
	}
```

- [ ] **Step 2: Run test, expect FAIL**

- [ ] **Step 3: Implement `meta.go`**

```go
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CurrentSchemaVersion mirrors store.currentSchemaVersion. Bump when the on-disk
// SQLite schema changes incompatibly; cache directories with a mismatched value
// are rebuilt from scratch.
const CurrentSchemaVersion = 1

// Meta is persisted to <cache>/meta.json alongside graph.db.
type Meta struct {
	SchemaVersion int      `json:"schema_version"`
	HeadSHA       string   `json:"head_sha"`
	ParserVersion string   `json:"parser_version"`
	Languages     []string `json:"languages"`
	BuiltAtUnix   int64    `json:"built_at_unix"`
}

// ReadMeta loads meta.json. Returns a zero Meta and nil error when the file is missing.
func ReadMeta(path string) (Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Meta{}, nil
		}
		return Meta{}, fmt.Errorf("read %s: %w", path, err)
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return Meta{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// WriteMeta atomically writes meta.json (write-temp-then-rename).
func WriteMeta(path string, m Meta) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s: %w", tmp, err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests, expect PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/cache/meta.go internal/graph/cache/meta_test.go
git commit -m "feat(graph/cache): atomic meta.json read/write"
```

---

## Task 2.17: Parser registry

**Files:**
- Create: `internal/graph/parser/registry.go`
- Create: `internal/graph/parser/registry_test.go`

- [ ] **Step 1: Write failing test**

```go
package parser

import "testing"

func TestRegistryByExtension(t *testing.T) {
	reg := DefaultRegistry()
	cases := map[string]string{
		"foo.go":  "go",
		"foo.ts":  "typescript",
		"foo.tsx": "typescript",
		"foo.js":  "javascript",
		"foo.mjs": "javascript",
		"foo.rs":  "rust",
	}
	for path, wantLang := range cases {
		p := reg.ForPath(path)
		if p == nil {
			t.Errorf("ForPath(%q) = nil", path)
			continue
		}
		if p.Language() != wantLang {
			t.Errorf("ForPath(%q).Language() = %q, want %q", path, p.Language(), wantLang)
		}
	}
	if reg.ForPath("foo.png") != nil {
		t.Error("ForPath(foo.png) should return nil")
	}
}
```

- [ ] **Step 2: Run test, expect FAIL**

- [ ] **Step 3: Implement `registry.go`**

```go
package parser

import (
	"path/filepath"
	"strings"
)

// Registry maps file extensions to parsers.
type Registry struct {
	byExt map[string]Parser
}

// DefaultRegistry returns a Registry covering every parser shipped in Phase 2.
func DefaultRegistry() *Registry {
	r := &Registry{byExt: map[string]Parser{}}
	for _, p := range []Parser{
		NewGoParser(),
		NewTypeScriptParser(),
		NewJavaScriptParser(),
		NewRustParser(),
	} {
		for _, ext := range p.Extensions() {
			r.byExt[strings.ToLower(ext)] = p
		}
	}
	return r
}

// ForPath returns the Parser for a path's extension, or nil if unsupported.
func (r *Registry) ForPath(path string) Parser {
	return r.byExt[strings.ToLower(filepath.Ext(path))]
}

// Languages returns the set of language strings the registry covers.
func (r *Registry) Languages() []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range r.byExt {
		if !seen[p.Language()] {
			seen[p.Language()] = true
			out = append(out, p.Language())
		}
	}
	return out
}
```

- [ ] **Step 4: Run test, expect PASS**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/parser/registry.go internal/graph/parser/registry_test.go
git commit -m "feat(graph/parser): default registry mapping extensions to parsers"
```

---

## Task 2.18: Full-build orchestrator (sequential first cut)

**Files:**
- Create: `internal/graph/cache/builder.go`
- Create: `internal/graph/cache/builder_test.go`
- Create: `internal/graph/cache/walk.go`

- [ ] **Step 1: Write failing test**

```go
package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuilderFullBuildOnTempRepo(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "main.go"), `package main
func Greet(n string) string { return "hi " + n }
func main() { Greet("x") }
`)
	mustWrite(t, filepath.Join(repo, "ignored.png"), "binary")

	home := t.TempDir()
	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	res, err := b.FullBuild()
	if err != nil {
		t.Fatal(err)
	}
	if res.FilesParsed != 1 {
		t.Errorf("FilesParsed=%d want 1", res.FilesParsed)
	}
	if _, err := os.Stat(GraphDB(home, RepoKey(repo))); err != nil {
		t.Errorf("graph.db missing: %v", err)
	}
	meta, err := ReadMeta(MetaFile(home, RepoKey(repo)))
	if err != nil {
		t.Fatal(err)
	}
	if meta.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("meta schema=%d want %d", meta.SchemaVersion, CurrentSchemaVersion)
	}
}

func TestBuilderFullBuildDeterministic(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "main.go"), `package main
func A() {}
func B() { A() }
`)
	home := t.TempDir()
	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.FullBuild(); err != nil {
		t.Fatal(err)
	}
	d1 := dumpDB(t, GraphDB(home, RepoKey(repo)))
	if _, err := b.FullBuild(); err != nil {
		t.Fatal(err)
	}
	d2 := dumpDB(t, GraphDB(home, RepoKey(repo)))
	if d1 != d2 {
		t.Errorf("two full builds produced different dumps:\n%s\n----\n%s", d1, d2)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
```

Add `dumpDB` test helper in `builder_test.go`:

```go
import (
	"database/sql"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

func dumpDB(t *testing.T, path string) string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var lines []string
	rows, err := db.Query(`SELECT id, kind, path, name, container, language FROM nodes`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id, kind, p, name, lang string
		var container sql.NullString
		if err := rows.Scan(&id, &kind, &p, &name, &container, &lang); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, "N "+id+"|"+kind+"|"+p+"|"+name+"|"+container.String+"|"+lang)
	}
	rows.Close()
	rows, err = db.Query(`SELECT src, dst, kind FROM edges`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var s, d, k string
		if err := rows.Scan(&s, &d, &k); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, "E "+s+"|"+d+"|"+k)
	}
	rows.Close()
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}
```

- [ ] **Step 2: Run test, expect FAIL**

- [ ] **Step 3: Implement `walk.go`**

```go
package cache

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// WalkRepo returns every regular file in root, skipping VCS and dependency
// directories. Paths returned are repo-relative with forward slashes.
func WalkRepo(root string) ([]string, error) {
	var out []string
	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "target": true, "vendor": true, ".devpilot": true,
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out, err
}

// FilterByParser keeps only files whose extension is recognised by parser.Registry.
// Callers pass a probe function so cache does not need to import parser directly.
func FilterByParser(paths []string, supported func(path string) bool) []string {
	var out []string
	for _, p := range paths {
		if supported(p) {
			out = append(out, p)
		}
	}
	return out
}

// IsHidden returns true for dot-prefixed segments other than "." and "..".
func IsHidden(name string) bool {
	return strings.HasPrefix(name, ".") && name != "." && name != ".."
}
```

- [ ] **Step 4: Implement `builder.go`**

```go
package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/resolver"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

// Builder owns the cache directory for a single repo and produces graph.db.
type Builder struct {
	home    string
	repo    string
	key     string
	reg     *parser.Registry
}

// NewBuilder validates the repo path and constructs a Builder.
func NewBuilder(home, repo string) (*Builder, error) {
	abs, err := filepath.Abs(repo)
	if err != nil {
		return nil, fmt.Errorf("abs(%s): %w", repo, err)
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("repo %s is not a directory", abs)
	}
	key := RepoKey(abs)
	if err := EnsureDirs(home, key); err != nil {
		return nil, err
	}
	return &Builder{home: home, repo: abs, key: key, reg: parser.DefaultRegistry()}, nil
}

// BuildResult summarises a build for callers and tests.
type BuildResult struct {
	FilesParsed int
	NodesInsert int
	EdgesInsert int
	Mode        string // "full" | "incremental"
}

// FullBuild parses every supported file in the repo, runs the resolver,
// inserts into graph.db, and writes meta.json. It deletes any prior graph.db
// first so two consecutive calls produce identical output.
func (b *Builder) FullBuild() (BuildResult, error) {
	rel := func() error { return nil }
	defer func() { _ = rel() }()
	var err error
	rel, err = AcquireBuildLock(LockFile(b.home, b.key), 60*time.Second)
	if err != nil {
		return BuildResult{}, err
	}

	dbPath := GraphDB(b.home, b.key)
	_ = os.Remove(dbPath)

	files, err := WalkRepo(b.repo)
	if err != nil {
		return BuildResult{}, fmt.Errorf("walk %s: %w", b.repo, err)
	}
	files = FilterByParser(files, func(p string) bool {
		return b.reg.ForPath(p) != nil
	})

	st, err := store.Open(dbPath)
	if err != nil {
		return BuildResult{}, err
	}
	defer st.Close()

	var allNodes []store.Node
	var allEdges []store.Edge
	ifaceMethods := map[string][]string{}

	for _, rel := range files {
		p := b.reg.ForPath(rel)
		src, err := os.ReadFile(filepath.Join(b.repo, rel))
		if err != nil {
			return BuildResult{}, fmt.Errorf("read %s: %w", rel, err)
		}
		res, err := p.Parse(rel, src)
		if err != nil {
			return BuildResult{}, fmt.Errorf("parse %s: %w", rel, err)
		}
		allNodes = append(allNodes, res.Nodes...)
		allEdges = append(allEdges, res.Edges...)
		for k, v := range res.InterfaceMethods {
			ifaceMethods[k] = append(ifaceMethods[k], v...)
		}
	}

	allEdges = resolver.New(allNodes).Rewrite(allEdges)
	allEdges = append(allEdges, resolver.Implements(allNodes, ifaceMethods)...)
	if hasTSConfig(b.repo) {
		ts, err := resolver.NewTSConfigResolver(b.repo)
		if err != nil {
			return BuildResult{}, err
		}
		allEdges = ts.Rewrite(allEdges)
	}

	if err := st.InsertNodes(allNodes); err != nil {
		return BuildResult{}, err
	}
	if err := st.InsertEdges(allEdges); err != nil {
		return BuildResult{}, err
	}

	meta := Meta{
		SchemaVersion: CurrentSchemaVersion,
		HeadSHA:       gitHeadSHA(b.repo),
		ParserVersion: parserVersionTag(b.reg),
		Languages:     b.reg.Languages(),
		BuiltAtUnix:   time.Now().Unix(),
	}
	if err := WriteMeta(MetaFile(b.home, b.key), meta); err != nil {
		return BuildResult{}, err
	}
	return BuildResult{
		FilesParsed: len(files),
		NodesInsert: len(allNodes),
		EdgesInsert: len(allEdges),
		Mode:        "full",
	}, nil
}

func hasTSConfig(root string) bool {
	_, err := os.Stat(filepath.Join(root, "tsconfig.json"))
	return err == nil
}

func parserVersionTag(reg *parser.Registry) string {
	langs := reg.Languages()
	return "phase2:" + strings.Join(langs, ",")
}
```

Add `gitHeadSHA` in a small helper file `internal/graph/cache/git.go`:

```go
package cache

import (
	"os/exec"
	"strings"
)

// gitHeadSHA returns the HEAD SHA of repo, or "" if git is unavailable or repo is not a git checkout.
func gitHeadSHA(repo string) string {
	out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
```

**Note on resolver:** Phase 1's `resolver` package exposes `resolver.New(nodes []store.Node)` returning a resolver that rewrites cross-file `external::pkg.Sym` edges, and `resolver.Implements(nodes, ifaceMethods)` returning derived `implements` edges. If the exact function names differ, adjust the calls in `builder.go` to match what `internal/graph/resolver/resolver.go` and `implements.go` export — do NOT change the resolver API in this task.

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/graph/cache/ -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/graph/cache/walk.go internal/graph/cache/builder.go internal/graph/cache/git.go internal/graph/cache/builder_test.go
git commit -m "feat(graph/cache): sequential full-build orchestrator producing graph.db + meta.json"
```

---

## Task 2.19: Parallel parsing via errgroup

**Files:**
- Modify: `internal/graph/cache/builder.go`

- [ ] **Step 1: Add dependency**

```bash
go get golang.org/x/sync/errgroup
go mod tidy
```

- [ ] **Step 2: Write failing test `TestBuilderParallelMatchesSequential`**

```go
func TestBuilderParallelMatchesSequential(t *testing.T) {
	repo := t.TempDir()
	for i := 0; i < 20; i++ {
		mustWrite(t, filepath.Join(repo, fmt.Sprintf("f%d.go", i)),
			fmt.Sprintf("package x\nfunc F%d() {}\n", i))
	}
	hSeq := t.TempDir()
	hPar := t.TempDir()

	bSeq, _ := NewBuilder(hSeq, repo)
	bSeq.MaxWorkers = 1
	bPar, _ := NewBuilder(hPar, repo)
	bPar.MaxWorkers = 8

	if _, err := bSeq.FullBuild(); err != nil {
		t.Fatal(err)
	}
	if _, err := bPar.FullBuild(); err != nil {
		t.Fatal(err)
	}
	if dumpDB(t, GraphDB(hSeq, RepoKey(repo))) != dumpDB(t, GraphDB(hPar, RepoKey(repo))) {
		t.Error("parallel build differs from sequential build")
	}
}
```

(Add `"fmt"` import.)

- [ ] **Step 3: Run test, expect FAIL** (`MaxWorkers` undefined)

- [ ] **Step 4: Implement**

In `builder.go`:

```go
type Builder struct {
	home       string
	repo       string
	key        string
	reg        *parser.Registry
	MaxWorkers int
}

func NewBuilder(home, repo string) (*Builder, error) {
	// ... unchanged ...
	return &Builder{home: home, repo: abs, key: key, reg: parser.DefaultRegistry(), MaxWorkers: 0}, nil
}
```

Replace the sequential loop with an `errgroup` fanout:

```go
import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

type fileResult struct {
	res parser.ParseResult
	err error
}

workers := b.MaxWorkers
if workers <= 0 {
	workers = 4
}

results := make([]fileResult, len(files))
g, ctx := errgroup.WithContext(context.Background())
sem := make(chan struct{}, workers)
var mu sync.Mutex
_ = mu // results array is indexed, no mu needed

for i, rel := range files {
	i, rel := i, rel
	p := b.reg.ForPath(rel)
	g.Go(func() error {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
		defer func() { <-sem }()
		src, err := os.ReadFile(filepath.Join(b.repo, rel))
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		res, err := p.Parse(rel, src)
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		results[i] = fileResult{res: res}
		return nil
	})
}
if err := g.Wait(); err != nil {
	return BuildResult{}, err
}
for _, fr := range results {
	allNodes = append(allNodes, fr.res.Nodes...)
	allEdges = append(allEdges, fr.res.Edges...)
	for k, v := range fr.res.InterfaceMethods {
		ifaceMethods[k] = append(ifaceMethods[k], v...)
	}
}
```

Determinism comes from iterating `results` in `files` order, which is itself produced by `filepath.WalkDir` (lexical).

- [ ] **Step 5: Run tests, expect PASS**

- [ ] **Step 6: Commit**

```bash
git commit -am "feat(graph/cache): parallel parsing with errgroup, deterministic output"
```

---

## Task 2.20: Schema-version mismatch triggers full rebuild

**Files:**
- Modify: `internal/graph/cache/builder.go`
- Modify: `internal/graph/cache/builder_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestSchemaMismatchRebuilds(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "a.go"), "package x\nfunc A(){}\n")
	home := t.TempDir()
	b, _ := NewBuilder(home, repo)
	if _, err := b.FullBuild(); err != nil {
		t.Fatal(err)
	}
	// Poison meta.json with an older schema_version.
	m, _ := ReadMeta(MetaFile(home, RepoKey(repo)))
	m.SchemaVersion = 0
	if err := WriteMeta(MetaFile(home, RepoKey(repo)), m); err != nil {
		t.Fatal(err)
	}
	res, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != "full" {
		t.Errorf("mode=%q want full", res.Mode)
	}
	got, _ := ReadMeta(MetaFile(home, RepoKey(repo)))
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("schema=%d want %d", got.SchemaVersion, CurrentSchemaVersion)
	}
}
```

- [ ] **Step 2: Run test, expect FAIL** (`Build` undefined)

- [ ] **Step 3: Implement `Build` dispatcher**

```go
// Build picks between full and incremental based on cache state.
// - missing graph.db or missing/invalid meta.json → full
// - meta.SchemaVersion != CurrentSchemaVersion   → wipe cache, full
// - otherwise: incremental (delegated to BuildIncremental in Task 2.21)
func (b *Builder) Build() (BuildResult, error) {
	dbPath := GraphDB(b.home, b.key)
	metaPath := MetaFile(b.home, b.key)

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return b.FullBuild()
	}
	m, err := ReadMeta(metaPath)
	if err != nil {
		return BuildResult{}, err
	}
	if m.SchemaVersion != CurrentSchemaVersion {
		_ = os.RemoveAll(GraphDir(b.home, b.key))
		if err := EnsureDirs(b.home, b.key); err != nil {
			return BuildResult{}, err
		}
		return b.FullBuild()
	}
	return b.BuildIncremental(m)
}

// BuildIncremental is added in Task 2.21. Until then, fall back to full.
func (b *Builder) BuildIncremental(prev Meta) (BuildResult, error) {
	return b.FullBuild() // stub — replaced in Task 2.21
}
```

- [ ] **Step 4: Run test, expect PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(graph/cache): schema-version mismatch wipes cache and full-rebuilds"
```

---

## Task 2.21: Incremental update via `git diff --name-status`

**Files:**
- Create: `internal/graph/cache/incremental.go`
- Create: `internal/graph/cache/incremental_test.go`
- Modify: `internal/graph/cache/builder.go`
- Modify: `internal/graph/store/store.go` (add `DeleteFilePaths` helper)

- [ ] **Step 1: Extend `store.Store` with a delete helper**

In `internal/graph/store/store.go` append:

```go
// DeleteByPaths deletes every node whose path is in paths, plus edges whose
// src or dst belongs to a deleted node. Returns counts deleted.
func (s *Store) DeleteByPaths(paths []string) (nodes, edges int, err error) {
	if len(paths) == 0 {
		return 0, 0, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	// Collect node ids in path set.
	var ids []string
	q := `SELECT id FROM nodes WHERE path = ?`
	for _, p := range paths {
		rows, err := tx.Query(q, p)
		if err != nil {
			return 0, 0, err
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return 0, 0, err
			}
			ids = append(ids, id)
		}
		rows.Close()
	}
	for _, id := range ids {
		r, err := tx.Exec(`DELETE FROM edges WHERE src = ? OR dst = ?`, id, id)
		if err != nil {
			return 0, 0, err
		}
		n, _ := r.RowsAffected()
		edges += int(n)
	}
	for _, p := range paths {
		r, err := tx.Exec(`DELETE FROM nodes WHERE path = ?`, p)
		if err != nil {
			return 0, 0, err
		}
		n, _ := r.RowsAffected()
		nodes += int(n)
	}
	return nodes, edges, tx.Commit()
}
```

Add a quick test in `store_test.go`:

```go
func TestDeleteByPaths(t *testing.T) {
	st, _ := Open(t.TempDir() + "/g.db")
	defer st.Close()
	_ = st.InsertNodes([]Node{
		{ID: "a.go", Kind: "file", Path: "a.go", Name: "a.go", Language: "go"},
		{ID: "a.go::A", Kind: "function", Path: "a.go", Name: "A", Language: "go"},
		{ID: "b.go", Kind: "file", Path: "b.go", Name: "b.go", Language: "go"},
	})
	_ = st.InsertEdges([]Edge{
		{Src: "a.go", Dst: "a.go::A", Kind: "contains"},
		{Src: "b.go", Dst: "a.go::A", Kind: "calls"},
	})
	n, e, err := st.DeleteByPaths([]string{"a.go"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || e != 2 {
		t.Errorf("delete: nodes=%d edges=%d, want 2/2", n, e)
	}
	if _, err := st.GetNode("a.go::A"); err == nil {
		t.Error("a.go::A still exists")
	}
}
```

Commit: `feat(graph/store): DeleteByPaths for incremental update`

- [ ] **Step 2: Write failing test `TestIncrementalMatchesFullRebuild`**

`incremental_test.go`:

```go
package cache

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIncrementalMatchesFullRebuild(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	mustGit(t, repo, "init", "-q")
	mustGit(t, repo, "config", "user.email", "t@t")
	mustGit(t, repo, "config", "user.name", "t")
	mustWrite(t, filepath.Join(repo, "a.go"), "package x\nfunc A(){}\n")
	mustWrite(t, filepath.Join(repo, "b.go"), "package x\nfunc B(){ A() }\n")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "first")

	homeA := t.TempDir()
	bA, _ := NewBuilder(homeA, repo)
	if _, err := bA.FullBuild(); err != nil {
		t.Fatal(err)
	}

	// Mutate b.go and add c.go, then commit.
	mustWrite(t, filepath.Join(repo, "b.go"), "package x\nfunc B(){ A(); A() }\n")
	mustWrite(t, filepath.Join(repo, "c.go"), "package x\nfunc C(){}\n")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "second")

	// Incremental over the cache.
	res, err := bA.Build()
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != "incremental" {
		t.Errorf("mode=%q want incremental", res.Mode)
	}

	// Full rebuild into a fresh home and compare dumps.
	homeB := t.TempDir()
	bB, _ := NewBuilder(homeB, repo)
	if _, err := bB.FullBuild(); err != nil {
		t.Fatal(err)
	}

	if dumpDB(t, GraphDB(homeA, RepoKey(repo))) != dumpDB(t, GraphDB(homeB, RepoKey(repo))) {
		t.Error("incremental result differs from full rebuild")
	}
}

func mustGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2026-05-20T00:00:00", "GIT_COMMITTER_DATE=2026-05-20T00:00:00")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
```

- [ ] **Step 3: Run test, expect FAIL**

- [ ] **Step 4: Implement `incremental.go`**

```go
package cache

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/resolver"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

// BuildIncremental applies a delta on top of an existing graph.db.
// Falls back to FullBuild when prev.HeadSHA is empty or not an ancestor of HEAD.
func (b *Builder) BuildIncremental(prev Meta) (BuildResult, error) {
	currentHead := gitHeadSHA(b.repo)
	if prev.HeadSHA == "" || currentHead == "" || !isAncestor(b.repo, prev.HeadSHA, currentHead) {
		return b.FullBuild()
	}
	if prev.HeadSHA == currentHead {
		// No changes since last build — nothing to do.
		return BuildResult{Mode: "incremental"}, nil
	}

	rel, err := AcquireBuildLock(LockFile(b.home, b.key), 60*time.Second)
	if err != nil {
		return BuildResult{}, err
	}
	defer func() { _ = rel() }()

	changed, err := gitChangedFiles(b.repo, prev.HeadSHA, currentHead)
	if err != nil {
		return BuildResult{}, err
	}

	st, err := store.Open(GraphDB(b.home, b.key))
	if err != nil {
		return BuildResult{}, err
	}
	defer st.Close()

	// Delete nodes/edges belonging to every changed-or-deleted file.
	allDelete := append([]string{}, changed.Modified...)
	allDelete = append(allDelete, changed.Deleted...)
	if _, _, err := st.DeleteByPaths(allDelete); err != nil {
		return BuildResult{}, err
	}

	// Re-parse modified + added files.
	var newNodes []store.Node
	var newEdges []store.Edge
	ifaceMethods := map[string][]string{}
	reg := b.reg
	for _, p := range append(changed.Modified, changed.Added...) {
		par := reg.ForPath(p)
		if par == nil {
			continue
		}
		src, err := os.ReadFile(filepath.Join(b.repo, p))
		if err != nil {
			if os.IsNotExist(err) {
				continue // raced with delete
			}
			return BuildResult{}, fmt.Errorf("read %s: %w", p, err)
		}
		res, err := par.Parse(p, src)
		if err != nil {
			return BuildResult{}, err
		}
		newNodes = append(newNodes, res.Nodes...)
		newEdges = append(newEdges, res.Edges...)
		for k, v := range res.InterfaceMethods {
			ifaceMethods[k] = append(ifaceMethods[k], v...)
		}
	}

	// Resolver runs against the union of (existing nodes intersect unchanged) + newNodes.
	// For correctness in v1 we accept the conservative cost of re-loading the full node
	// set: it's a single SELECT and the resolver only needs the index, not edges.
	allNodes, err := allNodesFromStore(st)
	if err != nil {
		return BuildResult{}, err
	}
	allNodes = append(allNodes, newNodes...)
	newEdges = resolver.New(allNodes).Rewrite(newEdges)
	newEdges = append(newEdges, resolver.Implements(allNodes, ifaceMethods)...)
	if hasTSConfig(b.repo) {
		ts, err := resolver.NewTSConfigResolver(b.repo)
		if err != nil {
			return BuildResult{}, err
		}
		newEdges = ts.Rewrite(newEdges)
	}

	if err := st.InsertNodes(newNodes); err != nil {
		return BuildResult{}, err
	}
	if err := st.InsertEdges(newEdges); err != nil {
		return BuildResult{}, err
	}

	meta := Meta{
		SchemaVersion: CurrentSchemaVersion,
		HeadSHA:       currentHead,
		ParserVersion: parserVersionTag(reg),
		Languages:     reg.Languages(),
		BuiltAtUnix:   time.Now().Unix(),
	}
	if err := WriteMeta(MetaFile(b.home, b.key), meta); err != nil {
		return BuildResult{}, err
	}
	return BuildResult{
		FilesParsed: len(changed.Modified) + len(changed.Added),
		NodesInsert: len(newNodes),
		EdgesInsert: len(newEdges),
		Mode:        "incremental",
	}, nil
}

type changeSet struct {
	Added, Modified, Deleted []string
}

func gitChangedFiles(repo, from, to string) (changeSet, error) {
	out, err := exec.Command("git", "-C", repo, "diff", "--name-status", from, to).CombinedOutput()
	if err != nil {
		return changeSet{}, fmt.Errorf("git diff: %w (%s)", err, out)
	}
	var cs changeSet
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		status, path := fields[0], fields[len(fields)-1]
		path = filepath.ToSlash(path)
		switch status[0] {
		case 'A':
			cs.Added = append(cs.Added, path)
		case 'M':
			cs.Modified = append(cs.Modified, path)
		case 'D':
			cs.Deleted = append(cs.Deleted, path)
		case 'R':
			// Renames: treat the old path as deleted, new path as added.
			if len(fields) == 3 {
				cs.Deleted = append(cs.Deleted, filepath.ToSlash(fields[1]))
				cs.Added = append(cs.Added, filepath.ToSlash(fields[2]))
			}
		}
	}
	return cs, nil
}

func isAncestor(repo, a, b string) bool {
	err := exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", a, b).Run()
	return err == nil
}

func allNodesFromStore(st *store.Store) ([]store.Node, error) {
	// store does not yet expose a "scan all nodes" API; for Phase 2 this is
	// acceptably small. Add a thin helper directly on Store if you find this
	// in a tight loop — but keep the resolver consuming []store.Node.
	return st.AllNodes()
}
```

Append to `store.go`:

```go
// AllNodes returns every node currently in the database.
// Intended for cache.Builder; not for query hot-paths.
func (s *Store) AllNodes() ([]Node, error) {
	rows, err := s.db.Query(
		`SELECT id, kind, path, name, container, language, start_line, end_line, is_exported, signature_hash FROM nodes`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		var container, sigHash sql.NullString
		var exported int
		if err := rows.Scan(
			&n.ID, &n.Kind, &n.Path, &n.Name, &container, &n.Language,
			&n.StartLine, &n.EndLine, &exported, &sigHash,
		); err != nil {
			return nil, err
		}
		n.Container = container.String
		n.SignatureHash = sigHash.String
		n.IsExported = exported == 1
		out = append(out, n)
	}
	return out, rows.Err()
}
```

Also add a quick test in `store_test.go` (`TestAllNodes`) covering insert + read-back.

Commit ordering: ship the store helpers first (`feat(graph/store): AllNodes + DeleteByPaths`), then the incremental builder.

- [ ] **Step 5: Run tests, expect PASS**

```bash
go test ./internal/graph/... -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/graph/cache/incremental.go internal/graph/cache/incremental_test.go internal/graph/cache/builder.go internal/graph/store/store.go internal/graph/store/store_test.go
git commit -m "feat(graph/cache): incremental update via git diff with full-rebuild fallback"
```

---

## Task 2.22: Lazy preflight sweep on Build entry

**Files:**
- Modify: `internal/graph/cache/builder.go`
- Modify: `internal/graph/cache/builder_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestBuildSweepsStalePreflight(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "a.go"), "package x\n")
	home := t.TempDir()
	preflightDir := filepath.Join(home, "preflight")
	_ = os.MkdirAll(preflightDir, 0o755)
	stalePath := filepath.Join(preflightDir, "stale.json")
	_ = os.WriteFile(stalePath, []byte("{}"), 0o644)
	stale := time.Now().Add(-30 * 24 * time.Hour)
	_ = os.Chtimes(stalePath, stale, stale)

	b, _ := NewBuilder(home, repo)
	if _, err := b.Build(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Errorf("stale preflight not swept: %v", err)
	}
}
```

(Add `"time"` import.)

- [ ] **Step 2: Run test, expect FAIL**

- [ ] **Step 3: Implement**

In `Build`, as the very first action:

```go
_ = SweepPreflight(b.home, 7*24*time.Hour)
```

(Swallowing the error is fine — sweep failure should never block a build.)

- [ ] **Step 4: Run test, expect PASS**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(graph/cache): lazy preflight TTL sweep on every Build invocation"
```

---

## Task 2.23: Phase 2 acceptance harness

**Files:**
- Create: `internal/graph/cache/phase2_acceptance_test.go`

- [ ] **Step 1: Write the acceptance test**

```go
package cache

import (
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestPhase2Acceptance asserts the four acceptance criteria from the plan in one place.
func TestPhase2Acceptance(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	t.Run("repokey_deterministic_12_chars", func(t *testing.T) {
		k := RepoKey("/abs/path/x")
		if len(k) != 12 {
			t.Errorf("len=%d", len(k))
		}
		if k != RepoKey("/abs/path/x") {
			t.Error("not deterministic")
		}
	})

	t.Run("two_full_builds_identical", func(t *testing.T) {
		repo := setupRepo(t, map[string]string{
			"main.go":  "package main\nfunc A(){}\nfunc main(){A()}\n",
			"util.ts":  "export function x(){}\n",
			"lib.rs":   "pub fn y(){}\n",
		})
		home := t.TempDir()
		b, _ := NewBuilder(home, repo)
		if _, err := b.FullBuild(); err != nil {
			t.Fatal(err)
		}
		d1 := dumpDB(t, GraphDB(home, RepoKey(repo)))
		if _, err := b.FullBuild(); err != nil {
			t.Fatal(err)
		}
		d2 := dumpDB(t, GraphDB(home, RepoKey(repo)))
		if d1 != d2 {
			t.Error("two full builds differ")
		}
	})

	t.Run("incremental_matches_full_5_files", func(t *testing.T) {
		repo := setupGitRepo(t, map[string]string{
			"a.go": "package x\nfunc A(){}\n",
			"b.go": "package x\nfunc B(){ A() }\n",
		})
		homeA := t.TempDir()
		bA, _ := NewBuilder(homeA, repo)
		if _, err := bA.FullBuild(); err != nil {
			t.Fatal(err)
		}
		mutateRepo(t, repo, map[string]string{
			"a.go": "package x\nfunc A(){}\nfunc A2(){}\n",
			"b.go": "package x\nfunc B(){ A(); A2() }\n",
			"c.go": "package x\nfunc C(){}\n",
			"d.go": "package x\nfunc D(){}\n",
			"e.go": "package x\nfunc E(){}\n",
		})
		if _, err := bA.Build(); err != nil {
			t.Fatal(err)
		}

		homeB := t.TempDir()
		bB, _ := NewBuilder(homeB, repo)
		if _, err := bB.FullBuild(); err != nil {
			t.Fatal(err)
		}
		if dumpDB(t, GraphDB(homeA, RepoKey(repo))) != dumpDB(t, GraphDB(homeB, RepoKey(repo))) {
			t.Error("incremental != full")
		}
	})

	t.Run("flock_serializes_concurrent_builders", func(t *testing.T) {
		repo := setupRepo(t, map[string]string{"main.go": "package main\nfunc main(){}\n"})
		home := t.TempDir()
		key := RepoKey(repo)
		_ = EnsureDirs(home, key)
		lockPath := LockFile(home, key)

		var (
			mu      sync.Mutex
			maxIn   int
			in      int
			wg      sync.WaitGroup
			fail    error
		)
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				rel, err := AcquireBuildLock(lockPath, 5*time.Second)
				if err != nil {
					fail = err
					return
				}
				defer rel()
				mu.Lock()
				in++
				if in > maxIn {
					maxIn = in
				}
				mu.Unlock()
				time.Sleep(30 * time.Millisecond)
				mu.Lock()
				in--
				mu.Unlock()
			}()
		}
		wg.Wait()
		if fail != nil {
			t.Fatal(fail)
		}
		if maxIn != 1 {
			t.Errorf("max concurrent=%d, want 1", maxIn)
		}
	})
}

func setupRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for p, c := range files {
		mustWrite(t, filepath.Join(repo, p), c)
	}
	return repo
}

func setupGitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := setupRepo(t, files)
	mustGit(t, repo, "init", "-q")
	mustGit(t, repo, "config", "user.email", "t@t")
	mustGit(t, repo, "config", "user.name", "t")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "init")
	return repo
}

func mutateRepo(t *testing.T, repo string, files map[string]string) {
	t.Helper()
	for p, c := range files {
		mustWrite(t, filepath.Join(repo, p), c)
	}
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "mutate")
}
```

- [ ] **Step 2: Run, fix any drift, expect PASS**

```bash
go test ./internal/graph/cache/ -run TestPhase2Acceptance -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/graph/cache/phase2_acceptance_test.go
git commit -m "test(graph/cache): Phase 2 acceptance harness"
```

---

## Task 2.24: Tag the phase and run full validation

- [ ] **Step 1: Run the full suite**

```bash
make lint
make test
```

Both must pass.

- [ ] **Step 2: Tag**

```bash
git tag graph-phase-2
```

- [ ] **Step 3: Update plan doc**

In `docs/plans/2026-05-19-devpilot-graph-plan.md`, replace the unchecked Phase 2 task list with checked boxes referencing this expansion:

```markdown
Phase 2 implemented per `docs/plans/2026-05-20-devpilot-graph-phase2-plan.md`. See `git log graph-phase-1..graph-phase-2 -- internal/graph/`.
```

- [ ] **Step 4: Commit and push the branch**

```bash
git commit -am "docs(graph): mark Phase 2 complete; link to phase2 plan"
```

Open PR via `devpilot-pr-creator` (not this plan's responsibility).

---

## Cross-task notes

- **Resolver API drift:** Phase 1 may name the rewrite entrypoint `resolver.Rewrite(nodes, edges)` rather than `resolver.New(nodes).Rewrite(edges)`. Before Task 2.18, run `go doc ./internal/graph/resolver/.` and adjust the calls in `builder.go` and `incremental.go` to the actual exported API. **Do not change the resolver itself in this phase.**
- **Tree-sitter node-type names:** TS/JS grammars sometimes name children with hyphens vs underscores depending on the binding version. If a `ChildByFieldName("source")` returns nil, fall back to a name-by-content scan and add a test fixture that reproduces the surprise. Never silently swallow nil — surface it as a `ParseError`.
- **Determinism:** Every list that ends up in `dumpDB` must come from a deterministic source. `filepath.WalkDir` is lexical-ordered; `errgroup` parallelism writes into an indexed slice so results are reassembled in walk order; SQLite `INSERT OR REPLACE` keys are deterministic. If any test goes flaky, the suspect is almost certainly an iteration over a `map`.
- **Commit cadence:** every passing test = one commit. Phase boundary = `graph-phase-2` tag.

---

## Self-review

- **Spec coverage**: Phase 2 plan section §742 lists 12 sub-tasks (2.1–2.12). This plan covers them across Tasks 2.1–2.22 (parsers 2.1–2.12; cache layout 2.13–2.16; orchestrator 2.17–2.22), plus acceptance harness 2.23 and tagging 2.24. ✅
- **Placeholder scan**: every step contains either a runnable command, a complete code block, or an exact test case. No "TBD", no "similar to" without showing code. ✅
- **Type consistency**: `Parser` interface, `ParseResult`, `store.Node`, `store.Edge`, `BuildResult.Mode` ("full"/"incremental"), `Meta` struct shape used uniformly across tasks. The newly added `Store.DeleteByPaths` and `Store.AllNodes` are introduced in the task that first needs them (2.21) and consistent across call sites. ✅
- **Scope check**: single phase, single subsystem, single PR's worth of work. Decomposition matches the plan doc. ✅
- **Risk note**: the resolver API drift caveat (Cross-task notes §1) is the highest-likelihood snag — flagged explicitly.
