# Devpilot Graph Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a Go-native code graph subsystem (`devpilot graph`) into the devpilot CLI, with `devpilot-pr-review` as the first composite consumer via a preflight JSON contract.

**Architecture:** Tree-sitter parsers for Go/TS/JS/Rust feed a SQLite store (pure-Go driver) keyed under `~/.devpilot/graphs/<repo-key>/`. Queries flow through `internal/graph/query/` and surface via the `devpilot graph` cobra subcommand emitting a uniform JSON envelope. Skill scripts under `skills/devpilot-pr-review/scripts/` call `preflight` and fall back to grep on failure.

**Tech Stack:** Go 1.25, cobra, `modernc.org/sqlite`, `github.com/smacker/go-tree-sitter` (with the official grammars for Go/TypeScript/JavaScript/Rust), bash for skill scripts.

**Reference spec:** [`docs/plans/2026-05-19-devpilot-graph-design.md`](./2026-05-19-devpilot-graph-design.md)

---

## Plan structure

The project is sized at ~13 weeks. This plan is laid out as seven phases with explicit dependencies and acceptance criteria. **Phase 1 is bite-sized TDD** because it executes immediately. **Phases 2–7 are concrete task lists at half-day granularity.** When each later phase begins, re-invoke `superpowers:writing-plans` against just that phase to expand into bite-sized steps with the latest context.

| Phase | Weeks | Goal | Depends on |
|---|---|---|---|
| 1 | 1–3 | Go parser + minimal SQLite store, end-to-end on one language | — |
| 2 | 4–5 | TS / JS / Rust parsers + cache layout + incremental update | Phase 1 |
| 3 | 6–7 | Query layer (callers, tests, impact, hubs, implementors, preflight composite) | Phase 2 |
| 4 | 8–9 | CLI surface + JSON envelope + schemas | Phase 3 |
| 5 | 10–11 | LSP cross-check test infrastructure | Phase 4 |
| 6 | 12 | Skill integration (`preflight.sh`, SKILL.md edits, fanout asserts) | Phase 4 |
| 7 | 13 | Release gate: perf, failure-mode walkthrough, real-PR dogfood | All |

---

## Phase 1: Go parser and minimal SQLite store

**Goal:** Build a Go-only graph end-to-end. Parse the devpilot repo, store nodes/edges in SQLite, query `callers_of` via raw SQL. No CLI yet.

**Files:**
- Create: `internal/graph/store/schema.go`
- Create: `internal/graph/store/migrations.go`
- Create: `internal/graph/store/store.go`
- Create: `internal/graph/store/store_test.go`
- Create: `internal/graph/parser/parser.go`
- Create: `internal/graph/parser/parser_test.go`
- Create: `internal/graph/parser/go.go`
- Create: `internal/graph/parser/go_test.go`
- Create: `internal/graph/parser/testdata/go/simple/main.go`
- Create: `internal/graph/resolver/imports.go`
- Create: `internal/graph/resolver/imports_test.go`
- Modify: `go.mod`, `go.sum`

### Task 1.1: Add dependencies

- [ ] **Step 1: Add modules to go.mod**

Run:
```bash
cd /Users/siyu/Works/github.com/siyuqian/devpilot
go get modernc.org/sqlite
go get github.com/smacker/go-tree-sitter
go get github.com/smacker/go-tree-sitter/golang
```

- [ ] **Step 2: Verify resolution**

Run: `go mod tidy && go build ./...`
Expected: clean build, no missing packages.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add tree-sitter + pure-Go sqlite deps for graph subsystem"
```

### Task 1.2: SQLite schema and migrations

**Files:** `internal/graph/store/schema.go`, `internal/graph/store/migrations.go`, `internal/graph/store/store_test.go`

- [ ] **Step 1: Write failing test for schema creation**

Create `internal/graph/store/store_test.go`:
```go
package store

import (
	"path/filepath"
	"testing"
)

func TestOpenCreatesSchema(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "graph.db")

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	tables := []string{"nodes", "edges", "schema_version"}
	for _, name := range tables {
		var exists int
		err := s.db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name,
		).Scan(&exists)
		if err != nil || exists != 1 {
			t.Errorf("table %s missing", name)
		}
	}
}
```

- [ ] **Step 2: Run test, expect FAIL (package does not exist)**

Run: `go test ./internal/graph/store/...`
Expected: build error referencing `Open`.

- [ ] **Step 3: Implement schema.go**

Create `internal/graph/store/schema.go`:
```go
package store

const schemaSQL = `
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS nodes (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  path TEXT NOT NULL,
  name TEXT NOT NULL,
  container TEXT,
  language TEXT NOT NULL,
  start_line INTEGER,
  end_line INTEGER,
  is_exported INTEGER NOT NULL DEFAULT 0,
  signature_hash TEXT
);

CREATE TABLE IF NOT EXISTS edges (
  src TEXT NOT NULL,
  dst TEXT NOT NULL,
  kind TEXT NOT NULL,
  PRIMARY KEY (src, dst, kind)
);

CREATE INDEX IF NOT EXISTS idx_edges_dst_kind ON edges(dst, kind);
CREATE INDEX IF NOT EXISTS idx_edges_src_kind ON edges(src, kind);
CREATE INDEX IF NOT EXISTS idx_nodes_path ON nodes(path);
`

const currentSchemaVersion = 1
```

- [ ] **Step 4: Implement store.go open path**

Create `internal/graph/store/store.go`:
```go
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	if _, err := db.Exec(`INSERT INTO schema_version (version) SELECT ? WHERE NOT EXISTS (SELECT 1 FROM schema_version)`, currentSchemaVersion); err != nil {
		db.Close()
		return nil, fmt.Errorf("seed schema_version: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }
```

- [ ] **Step 5: Run test, expect PASS**

Run: `go test ./internal/graph/store/... -run TestOpenCreatesSchema -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/graph/store/
git commit -m "feat(graph/store): initial sqlite schema and Open"
```

### Task 1.3: Node and edge insert/query

**Files:** `internal/graph/store/store.go`, `internal/graph/store/store_test.go`

- [ ] **Step 1: Write failing tests for InsertNode / GetNode / InsertEdge / CallersOf**

Append to `internal/graph/store/store_test.go`:
```go
func TestInsertAndGetNode(t *testing.T) {
	s := newTestStore(t)
	n := Node{
		ID: "internal/foo.go::Foo.Bar", Kind: "method", Path: "internal/foo.go",
		Name: "Bar", Container: "Foo", Language: "go", IsExported: true,
	}
	if err := s.InsertNodes([]Node{n}); err != nil {
		t.Fatalf("InsertNodes: %v", err)
	}
	got, err := s.GetNode(n.ID)
	if err != nil || got.Name != "Bar" {
		t.Fatalf("GetNode: %v %+v", err, got)
	}
}

func TestCallersOf(t *testing.T) {
	s := newTestStore(t)
	must := func(err error) { t.Helper(); if err != nil { t.Fatal(err) } }
	must(s.InsertNodes([]Node{
		{ID: "a.go::A", Kind: "function", Path: "a.go", Name: "A", Language: "go"},
		{ID: "b.go::B", Kind: "function", Path: "b.go", Name: "B", Language: "go"},
		{ID: "c.go::C", Kind: "function", Path: "c.go", Name: "C", Language: "go"},
	}))
	must(s.InsertEdges([]Edge{
		{Src: "a.go::A", Dst: "c.go::C", Kind: "calls"},
		{Src: "b.go::B", Dst: "c.go::C", Kind: "calls"},
	}))
	callers, err := s.CallersOf("c.go::C")
	must(err)
	if len(callers) != 2 {
		t.Fatalf("expected 2 callers, got %d: %v", len(callers), callers)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir() + "/graph.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
```

- [ ] **Step 2: Run tests, expect FAIL**

Run: `go test ./internal/graph/store/... -v`
Expected: undefined `Node`, `Edge`, `InsertNodes`, `InsertEdges`, `GetNode`, `CallersOf`.

- [ ] **Step 3: Implement Node, Edge, and methods**

Append to `internal/graph/store/store.go`:
```go
type Node struct {
	ID, Kind, Path, Name, Container, Language string
	StartLine, EndLine                        int
	IsExported                                bool
	SignatureHash                             string
}

type Edge struct {
	Src, Dst, Kind string
}

func (s *Store) InsertNodes(nodes []Node) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO nodes
		(id, kind, path, name, container, language, start_line, end_line, is_exported, signature_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, n := range nodes {
		var exported int
		if n.IsExported {
			exported = 1
		}
		if _, err := stmt.Exec(n.ID, n.Kind, n.Path, n.Name, n.Container, n.Language,
			n.StartLine, n.EndLine, exported, n.SignatureHash); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) InsertEdges(edges []Edge) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO edges (src, dst, kind) VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, e := range edges {
		if _, err := stmt.Exec(e.Src, e.Dst, e.Kind); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetNode(id string) (Node, error) {
	var n Node
	var exported int
	err := s.db.QueryRow(`SELECT id, kind, path, name, container, language, start_line, end_line, is_exported, signature_hash FROM nodes WHERE id = ?`, id).Scan(
		&n.ID, &n.Kind, &n.Path, &n.Name, &n.Container, &n.Language,
		&n.StartLine, &n.EndLine, &exported, &n.SignatureHash,
	)
	n.IsExported = exported == 1
	return n, err
}

func (s *Store) CallersOf(id string) ([]string, error) {
	rows, err := s.db.Query(`SELECT src FROM edges WHERE dst = ? AND kind = 'calls'`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var src string
		if err := rows.Scan(&src); err != nil {
			return nil, err
		}
		out = append(out, src)
	}
	return out, rows.Err()
}
```

- [ ] **Step 4: Run tests, expect PASS**

Run: `go test ./internal/graph/store/... -v`
Expected: 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/graph/store/
git commit -m "feat(graph/store): node/edge insert + callers_of query"
```

### Task 1.4: Parser interface and ParseResult type

**Files:** `internal/graph/parser/parser.go`, `internal/graph/parser/parser_test.go`

- [ ] **Step 1: Write failing test for ParseResult zero value**

Create `internal/graph/parser/parser_test.go`:
```go
package parser

import "testing"

func TestParseResultZero(t *testing.T) {
	var r ParseResult
	if r.Nodes != nil || r.Edges != nil || r.Errors != nil {
		t.Fatal("ParseResult zero value must be empty")
	}
}
```

- [ ] **Step 2: Run test, expect FAIL (undefined)**

Run: `go test ./internal/graph/parser/...`
Expected: undefined `ParseResult`.

- [ ] **Step 3: Implement parser.go**

Create `internal/graph/parser/parser.go`:
```go
package parser

import "github.com/siyuqian/devpilot/internal/graph/store"

type Parser interface {
	Language() string
	Extensions() []string
	Parse(path string, src []byte) (ParseResult, error)
}

type ParseResult struct {
	Nodes  []store.Node
	Edges  []store.Edge
	Errors []ParseError
}

type ParseError struct {
	Path    string
	Line    int
	Message string
}
```

- [ ] **Step 4: Run test, expect PASS**

Run: `go test ./internal/graph/parser/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/graph/parser/parser.go internal/graph/parser/parser_test.go
git commit -m "feat(graph/parser): common Parser interface and ParseResult"
```

### Task 1.5: Go parser — file and function nodes

**Files:** `internal/graph/parser/go.go`, `internal/graph/parser/go_test.go`, `internal/graph/parser/testdata/go/simple/main.go`

- [ ] **Step 1: Create fixture**

Create `internal/graph/parser/testdata/go/simple/main.go`:
```go
package main

import "fmt"

func Greet(name string) string {
	return fmt.Sprintf("hi %s", name)
}

func main() {
	fmt.Println(Greet("world"))
}
```

- [ ] **Step 2: Write failing test for Go function extraction**

Create `internal/graph/parser/go_test.go`:
```go
package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoParserExtractsFunctions(t *testing.T) {
	p := NewGoParser()
	path := filepath.Join("testdata", "go", "simple", "main.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.Parse("simple/main.go", src)
	if err != nil {
		t.Fatal(err)
	}

	var hasGreet, hasMain, hasFile bool
	for _, n := range r.Nodes {
		switch n.ID {
		case "simple/main.go::Greet":
			hasGreet = true
			if !n.IsExported {
				t.Error("Greet must be exported")
			}
		case "simple/main.go::main":
			hasMain = true
		case "simple/main.go":
			hasFile = true
			if n.Kind != "file" {
				t.Errorf("file node kind=%q", n.Kind)
			}
		}
	}
	if !hasGreet || !hasMain || !hasFile {
		t.Fatalf("missing nodes: greet=%v main=%v file=%v", hasGreet, hasMain, hasFile)
	}
}
```

- [ ] **Step 3: Run test, expect FAIL**

Run: `go test ./internal/graph/parser/... -run TestGoParserExtractsFunctions -v`
Expected: undefined `NewGoParser`.

- [ ] **Step 4: Implement Go parser file + function extraction**

Create `internal/graph/parser/go.go`:
```go
package parser

import (
	"context"
	"fmt"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
	goLang "github.com/smacker/go-tree-sitter/golang"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

type GoParser struct{ lang *sitter.Language }

func NewGoParser() *GoParser { return &GoParser{lang: goLang.GetLanguage()} }

func (p *GoParser) Language() string    { return "go" }
func (p *GoParser) Extensions() []string { return []string{".go"} }

func (p *GoParser) Parse(path string, src []byte) (ParseResult, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(p.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return ParseResult{}, fmt.Errorf("tree-sitter parse: %w", err)
	}
	defer tree.Close()

	res := ParseResult{}
	res.Nodes = append(res.Nodes, store.Node{
		ID: path, Kind: "file", Path: path, Name: path, Language: "go",
	})

	root := tree.RootNode()
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.Type() == "function_declaration" {
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			name := nameNode.Content(src)
			id := path + "::" + name
			res.Nodes = append(res.Nodes, store.Node{
				ID: id, Kind: "function", Path: path, Name: name, Language: "go",
				StartLine:  int(child.StartPoint().Row) + 1,
				EndLine:    int(child.EndPoint().Row) + 1,
				IsExported: isExportedGo(name),
			})
			res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
		}
	}
	return res, nil
}

func isExportedGo(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper([]rune(name)[0])
}
```

- [ ] **Step 5: Run test, expect PASS**

Run: `go test ./internal/graph/parser/... -run TestGoParserExtractsFunctions -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/graph/parser/go.go internal/graph/parser/go_test.go internal/graph/parser/testdata/
git commit -m "feat(graph/parser/go): file and function node extraction"
```

### Task 1.6: Go parser — method nodes (struct methods)

- [ ] **Step 1: Extend fixture**

Append to `internal/graph/parser/testdata/go/simple/main.go`:
```go
type Greeter struct{ prefix string }

func (g *Greeter) Hello(name string) string { return g.prefix + " " + name }
func (g Greeter) silent() string             { return "" }
```

- [ ] **Step 2: Write failing test for methods**

Append to `internal/graph/parser/go_test.go`:
```go
func TestGoParserExtractsMethods(t *testing.T) {
	p := NewGoParser()
	src, _ := os.ReadFile(filepath.Join("testdata", "go", "simple", "main.go"))
	r, _ := p.Parse("simple/main.go", src)

	wantIDs := map[string]bool{
		"simple/main.go::Greeter.Hello":  false,
		"simple/main.go::Greeter.silent": false,
	}
	for _, n := range r.Nodes {
		if _, ok := wantIDs[n.ID]; ok {
			wantIDs[n.ID] = true
			if n.Kind != "method" || n.Container != "Greeter" {
				t.Errorf("%s: kind=%q container=%q", n.ID, n.Kind, n.Container)
			}
		}
	}
	for id, found := range wantIDs {
		if !found {
			t.Errorf("missing %s", id)
		}
	}
}
```

- [ ] **Step 3: Run, expect FAIL**

Run: `go test ./internal/graph/parser/... -run TestGoParserExtractsMethods -v`

- [ ] **Step 4: Add method handling**

In `internal/graph/parser/go.go`, extend the `Parse` loop with:
```go
		if child.Type() == "method_declaration" {
			nameNode := child.ChildByFieldName("name")
			recvNode := child.ChildByFieldName("receiver")
			if nameNode == nil || recvNode == nil {
				continue
			}
			name := nameNode.Content(src)
			recvType := extractGoReceiverType(recvNode, src)
			id := fmt.Sprintf("%s::%s.%s", path, recvType, name)
			res.Nodes = append(res.Nodes, store.Node{
				ID: id, Kind: "method", Path: path, Name: name, Container: recvType, Language: "go",
				StartLine: int(child.StartPoint().Row) + 1, EndLine: int(child.EndPoint().Row) + 1,
				IsExported: isExportedGo(name),
			})
			res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
		}
```

And add helper:
```go
func extractGoReceiverType(recv *sitter.Node, src []byte) string {
	for i := 0; i < int(recv.NamedChildCount()); i++ {
		c := recv.NamedChild(i)
		if c.Type() == "parameter_declaration" {
			typeNode := c.ChildByFieldName("type")
			if typeNode == nil {
				continue
			}
			t := typeNode.Content(src)
			if len(t) > 0 && t[0] == '*' {
				t = t[1:]
			}
			return t
		}
	}
	return ""
}
```

- [ ] **Step 5: Run, expect PASS**

Run: `go test ./internal/graph/parser/... -v`

- [ ] **Step 6: Commit**

```bash
git add internal/graph/parser/
git commit -m "feat(graph/parser/go): method nodes with receiver types"
```

### Task 1.7: Go parser — struct, interface, type nodes

Apply the same TDD pattern. Add fixture additions (`type X struct`, `type Y interface`, `type Z = int`), write failing test asserting `kind: struct/interface/type` nodes are produced, implement `type_declaration` handler, pass, commit.

- [ ] **Step 1:** Extend fixture with `type Greeter2 struct {}`, `type Hello interface { Greet() string }`, `type Alias = string`.
- [ ] **Step 2:** Write `TestGoParserExtractsTypes` checking these three IDs exist with correct `kind`.
- [ ] **Step 3:** Run, expect FAIL.
- [ ] **Step 4:** Implement `type_declaration` branch handling `type_spec` children with `struct_type`, `interface_type`, anything else → `type`.
- [ ] **Step 5:** Run, expect PASS.
- [ ] **Step 6:** Commit `"feat(graph/parser/go): struct/interface/type nodes"`.

### Task 1.8: Go parser — `calls` edges

- [ ] **Step 1:** Extend fixture: a function that calls another (`Greet` calls `fmt.Sprintf`, `main` calls `Greet`).
- [ ] **Step 2:** Write `TestGoParserExtractsCalls` asserting edges `simple/main.go::main → simple/main.go::Greet` with kind `calls`. External calls (`fmt.Sprintf`) get edges to a synthetic node `external::fmt.Sprintf` for v1; resolution happens later.
- [ ] **Step 3:** Run, expect FAIL.
- [ ] **Step 4:** Walk the AST inside each function body, locate `call_expression` nodes, extract callee identifier (handle plain `Ident()` and `pkg.Sel`).
- [ ] **Step 5:** Run, expect PASS.
- [ ] **Step 6:** Commit `"feat(graph/parser/go): calls edges (intra-file + external)"`.

### Task 1.9: Go parser — `imports` edges

- [ ] **Step 1:** Extend fixture with `import "fmt"` and `import alias "strings"`.
- [ ] **Step 2:** Write test asserting edges `simple/main.go → external::fmt` and `simple/main.go → external::strings` with kind `imports`.
- [ ] **Step 3:** Run, expect FAIL.
- [ ] **Step 4:** Handle `import_declaration` nodes (both single and grouped) in the parser top-level walk.
- [ ] **Step 5:** Run, expect PASS.
- [ ] **Step 6:** Commit `"feat(graph/parser/go): imports edges"`.

### Task 1.10: Go parser — `tests` edges (heuristic)

- [ ] **Step 1:** Add fixture `simple/main_test.go` with `func TestGreet(t *testing.T) { Greet("x") }`.
- [ ] **Step 2:** Write test asserting `simple/main_test.go::TestGreet` exists AND `tests` edge from `TestGreet` to `simple/main.go::Greet`.
- [ ] **Step 3:** Run, expect FAIL.
- [ ] **Step 4:** In `Parse`, when a function name starts with `Test` and signature matches `*testing.T`, walk its body to find call_expressions; emit `tests` edges to each callee. (Calls edges are already emitted; `tests` is an additional edge for the same target.)
- [ ] **Step 5:** Run, expect PASS.
- [ ] **Step 6:** Commit `"feat(graph/parser/go): tests edges via TestXxx heuristic"`.

### Task 1.11: Cross-file import resolver (Go)

**Files:** `internal/graph/resolver/imports.go`, `internal/graph/resolver/imports_test.go`

- [ ] **Step 1:** Write test: given two Go fixture files in the same Go module (`a.go` imports relative path resolved via `go.mod`, calls `b.B()`), the resolver replaces the synthetic `external::b.B` edge with `b.go::B` and adds an `imports` edge between files.
- [ ] **Step 2:** Run, expect FAIL.
- [ ] **Step 3:** Implement `Resolve(parseResults []ParseResult, modulePath string) (resolved []ParseResult)`. Build a symbol-name → node-ID lookup keyed by `(package, name)`; rewrite calls/imports edges where the dst matches.
- [ ] **Step 4:** Run, expect PASS.
- [ ] **Step 5:** Commit `"feat(graph/resolver): cross-file import resolution for Go"`.

### Task 1.12: `implements` edges for Go interfaces

- [ ] **Step 1:** Fixture: `interface Greeter { Greet() string }` and `type Console struct{}` with `func (Console) Greet() string`.
- [ ] **Step 2:** Write test asserting an `implements` edge from `Console` to `Greeter` with exact method set match.
- [ ] **Step 3:** Run, expect FAIL.
- [ ] **Step 4:** In the resolver, after parsing all files, build interface → method-set map and struct → method-set map; emit `implements` for exact matches.
- [ ] **Step 5:** Run, expect PASS.
- [ ] **Step 6:** Commit `"feat(graph/resolver): implements edges (Go, exact method-set match)"`.

### Phase 1 acceptance

- [ ] `go test ./internal/graph/... -v` is green (all parser, store, resolver tests).
- [ ] A throwaway integration test in `internal/graph/integration_test.go` (not committed permanently) parses the devpilot repo's `internal/auth/` directory, persists to a `t.TempDir()` SQLite DB, and queries `CallersOf` for `internal/auth/...::Validate` returning the expected callers.
- [ ] Phase 1 produces ~1500 lines of code across parser/store/resolver. No public API surface yet (no CLI command).

---

## Phase 2: Remaining parsers + cache + incremental update

**Goal:** Add TypeScript / JavaScript / Rust parsers using the same interface, plus cache layout under `~/.devpilot/` with incremental update.

**File map:**
- Create: `internal/graph/parser/typescript.go` + test + `testdata/ts/simple/`
- Create: `internal/graph/parser/javascript.go` + test + `testdata/js/simple/`
- Create: `internal/graph/parser/rust.go` + test + `testdata/rust/simple/`
- Create: `internal/graph/resolver/tsconfig.go` + test
- Create: `internal/graph/cache/paths.go`
- Create: `internal/graph/cache/flock.go`
- Create: `internal/graph/cache/ttl.go`
- Create: `internal/graph/cache/builder.go` (orchestrates full + incremental build)

**Tasks (half-day granularity): COMPLETE — see `docs/plans/2026-05-20-devpilot-graph-phase2-plan.md` for the bite-sized expansion (24 tasks) and `git log graph-phase-1..graph-phase-2 -- internal/graph/` for the commits.**

- [x] **2.1** TS parser
- [x] **2.2** TS path-alias resolver
- [x] **2.3** TS `tests` edges
- [x] **2.4** TS `implements` / `extends`
- [x] **2.5** JS parser
- [x] **2.6** Rust parser
- [x] **2.7** Cache layout
- [x] **2.8** Flock helper
- [x] **2.9** TTL sweeper
- [x] **2.10** Builder orchestrator
- [x] **2.11** Incremental update
- [x] **2.12** Cache schema-version mismatch handling

**Phase 2 acceptance: PASS** (see `internal/graph/cache/phase2_acceptance_test.go`):
- [x] Parsers for all four languages green on their fixture suites.
- [x] `cache.RepoKey` deterministic and 12 chars.
- [x] Builder produces identical graph (snapshot match) on two consecutive full builds.
- [x] Incremental update on a 5-file change matches what a full rebuild would produce (snapshot equality on resulting graph.db dump).
- [x] Flock serializes two concurrent builders against the same repo-key.

---

## Phase 3: Query layer

**Goal:** All read-side operations that downstream consumers will use, including the `preflight` composite.

**File map:**
- Create: `internal/graph/query/callers.go` + test
- Create: `internal/graph/query/callees.go` + test
- Create: `internal/graph/query/tests.go` + test
- Create: `internal/graph/query/impact.go` + test
- Create: `internal/graph/query/hubs.go` + test
- Create: `internal/graph/query/implementors.go` + test
- Create: `internal/graph/query/context.go` + test
- Create: `internal/graph/query/detect_changes.go` + test
- Create: `internal/graph/query/preflight.go` + test
- Create: `internal/graph/query/risk.go` + test

**Tasks:**

- [ ] **3.1** `CallersOf(id, depth)` — BFS over `kind=calls` edges, returns nodes with hop count. Tested with a 3-hop chain.
- [ ] **3.2** `CalleesOf(id, depth)` — symmetrical.
- [ ] **3.3** `TestsFor(id)` — direct `kind=tests` lookup.
- [ ] **3.4** `ImpactRadius(files, depth)` — collect all symbol nodes contained in the file set, run `CallersOf` on each, union results.
- [ ] **3.5** `Hubs(threshold)` — `SELECT dst, COUNT(*) FROM edges WHERE kind='calls' GROUP BY dst HAVING COUNT(*) >= ?`.
- [ ] **3.6** `ImplementorsOf(interfaceID)` — direct `kind=implements` lookup with dst filter.
- [ ] **3.7** `Context(id, depth)` — return source snippet from `path` between `start_line` and `end_line`, optionally including caller snippets (depth=1).
- [ ] **3.8** `DetectChanges(base, head)` — parses git diff, joins with graph to produce list of changed-symbol metadata (kind, is_exported, change_type via signature_hash comparison).
- [ ] **3.9** `Preflight(base, head)` composite — calls `DetectChanges`, enriches each entry with `CallersOf` (limit 10), `TestsFor`, `ImplementorsOf` (where applicable), `Hubs` lookup, directory-based community, cross-community edges. Applies risk score and truncates to top 50.
- [ ] **3.10** `RiskScore(symbol)` — exact formula from spec §6.

**Phase 3 acceptance:**
- [ ] Each query function has at least one test demonstrating correctness on a small fixture graph (constructed directly via store.InsertNodes/InsertEdges, not via parser).
- [ ] `Preflight` round-trip test: build graph for `internal/auth/` fixture in devpilot, run preflight against two synthetic SHAs, assert JSON shape matches spec §6.

---

## Phase 4: CLI surface

**Goal:** Expose query layer as `devpilot graph <verb>` subcommands with uniform JSON envelope.

**File map:**
- Create: `cmd/devpilot/graph.go` (cobra registration, dispatches to subcommands)
- Create: `cmd/devpilot/graph_build.go`
- Create: `cmd/devpilot/graph_status.go`
- Create: `cmd/devpilot/graph_clean.go`
- Create: `cmd/devpilot/graph_query.go`
- Create: `cmd/devpilot/graph_impact.go`
- Create: `cmd/devpilot/graph_hubs.go`
- Create: `cmd/devpilot/graph_context.go`
- Create: `cmd/devpilot/graph_detect_changes.go`
- Create: `cmd/devpilot/graph_preflight.go`
- Create: `internal/graph/envelope/envelope.go` + test
- Create: `internal/graph/envelope/schemas/preflight.v1.json`
- Create: `internal/graph/envelope/schemas/query.v1.json`
- (etc., one schema per command output)
- Create: `cmd/devpilot/graph_e2e_test.go`

**Tasks:**

- [ ] **4.1** Envelope type + helpers: `New(cmd string)`, `OK(data)`, `Err(code, msg)`, `Suggest(cmds ...string)`, `Marshal() []byte`.
- [ ] **4.2** JSON schemas for each output type; `envelope.Validate(json, schema)` helper using `github.com/santhosh-tekuri/jsonschema/v5`.
- [ ] **4.3** `graph build <repo>` wires builder; auto-detects full vs incremental from `meta.json`.
- [ ] **4.4** `graph status <repo>` reads `meta.json` + counts rows; emits envelope.
- [ ] **4.5** `graph clean [--repo X | --all]` deletes cache dirs.
- [ ] **4.6** `graph query <pattern> <target>` dispatches to right query func; supports all six v1 patterns.
- [ ] **4.7** `graph impact`, `graph hubs`, `graph context` — thin wrappers over query layer.
- [ ] **4.8** `graph detect-changes`, `graph preflight` — composites, identical-shape envelope with `next_tool_suggestions` populated per spec.
- [ ] **4.9** E2e tests: for each subcommand, invoke compiled binary via `os/exec`, parse stdout, validate envelope shape via schema + assert specific fields.

**Phase 4 acceptance:**
- [ ] All nine subcommands and six `query` patterns have passing e2e tests.
- [ ] Every JSON output validates against its v1 schema.
- [ ] `devpilot graph preflight --base <sha> --head <sha>` against devpilot itself completes in < 10s on a warm cache.

---

## Phase 5: LSP cross-check test infrastructure

**Goal:** Validate parser/resolver accuracy against language servers. Runs nightly, not per-PR. Tagged `//go:build lsp_check`.

**N1.16 reframe for Go:** Phase 5 for Go is a **coverage check**, not a precision/recall gate. Because the native backend consumes `go/types` (same as `gopls`), agreement is near-tautological on the things both compute and diverges only on build-tag-gated or generated code. Concretely: assert every entry returned by `gopls workspace/symbol` appears in the native graph; log deltas; do not fail CI on the gap. Precision/recall gating with a ≥ 90% threshold remains for TS and Rust where the LSP is genuinely independent of our parser implementation.

**File map:**
- Create: `internal/graph/lsp/gopls.go` + test
- Create: `internal/graph/lsp/tsc.go` + test
- Create: `internal/graph/lsp/rust_analyzer.go` + test
- Create: `internal/graph/lsp/crosscheck_test.go` (master comparison suite)
- Create: `.github/workflows/lsp-nightly.yml`

**Tasks:**

- [ ] **5.1** `gopls` driver: spawn gopls in JSON-RPC mode, `initialize`, `textDocument/references`, parse results into a comparable map.
- [ ] **5.2** `tsc` driver: invoke `tsc --listFiles` + use `ts-morph` via subprocess or call `tsserver`'s `references` request via stdin.
- [ ] **5.3** `rust-analyzer` driver: similar JSON-RPC, `textDocument/references`.
- [ ] **5.4** Cross-check test: for each fixture repo, build graph; for a curated list of ~30 symbols across languages, compare graph's `CallersOf` with LSP's `references` result. Output a precision/recall report. For Go, verify coverage via `workspace/symbol` instead.
- [ ] **5.5** For TS and Rust: precision/recall ≥ 90% gate enforced in test (failure if below). For Go: coverage check, log deltas without CI gate.
- [ ] **5.6** GitHub Actions workflow runs the `lsp_check` build tag on a nightly cron.

**Phase 5 acceptance:**
- [ ] All three LSP drivers operational against their reference servers.
- [ ] Cross-check report on devpilot + one OSS fixture per language: coverage check passed for Go; ≥ 90% precision and recall for TS and Rust on `callers_of`, `tests_for`, `implementors_of`.
- [ ] Nightly workflow green for 5 consecutive nights.

---

## Phase 6: Skill integration

**Goal:** Wire `devpilot-pr-review` to consume preflight output; ship grep fallback.

**File map:**
- Create: `skills/devpilot-pr-review/scripts/preflight.sh`
- Create: `skills/devpilot-pr-review/scripts/grep_fallback.sh`
- Create: `skills/devpilot-pr-review/references/preflight.md`
- Modify: `skills/devpilot-pr-review/SKILL.md` (insert step 1.5)
- Modify: `skills/devpilot-pr-review/references/fanout.md` (`SHARED_PR_HEADER` paragraph)
- Modify: `skills/devpilot-pr-review/references/unknown-unknowns.md` (Agent A Q2 fast path)
- Modify: `skills/devpilot-pr-review/references/template.md` (Architecture Impact section)
- Create: `skills/devpilot-pr-review/tests/fanout_prompt_test.sh` (5+ assertions)

**Tasks:**

- [ ] **6.1** `grep_fallback.sh`: takes diff, extracts changed symbol names via simple regex (capitalized identifier preceded by `func` / `function` / `class` / `fn` / `export`), runs `git grep -n` per symbol, emits the minimum-viable JSON shape from spec §8.
- [ ] **6.2** `preflight.sh`: orchestrates cache check → `devpilot graph build` if needed → `devpilot graph preflight` → on any failure, invoke `grep_fallback.sh`. Implements exit codes 0/10/20 per spec §8.
- [ ] **6.3** Failure-mode walkthrough: manually trigger every scenario in spec §8 (kill `devpilot` binary mid-build, write-protect cache dir, unsupported language, etc.) and confirm exit code + mode field.
- [ ] **6.4** `references/preflight.md`: copy spec §6 schema + §8 failure matrix; this is the in-skill source of truth.
- [ ] **6.5** Insert step 1.5 in `SKILL.md` workflow per spec §7.
- [ ] **6.6** Modify `fanout.md` `SHARED_PR_HEADER`: paragraph instructing agents to prefer preflight `changed_symbols.callers` over manual grep when `mode != fallback_grep`.
- [ ] **6.7** Modify `unknown-unknowns.md` Agent A Q2: add fast-path reading from preflight; fall back to grep only when preflight unavailable for the symbol.
- [ ] **6.8** Modify `template.md`: add Architecture Impact section, populated from preflight `data.risk_summary` and `data.cross_community_edges`.
- [ ] **6.9** Five fanout-prompt assertions: shell test loads a mock preflight JSON, expands the fanout prompt template, greps the result for expected substrings (`Architecture Impact`, `New cross-module edges`, etc.).

**Phase 6 acceptance:**
- [ ] `preflight.sh` produces exit 0 on devpilot itself with a small synthetic PR.
- [ ] All Q7/§8 failure modes exit with expected code + JSON mode.
- [ ] All five+ fanout-prompt assertions pass.
- [ ] One manual end-to-end run: simulate a PR review on devpilot with the integrated skill; review body contains the Architecture Impact section.

---

## Phase 7: Polish + release gate

**Goal:** Final hardening and dogfooding before declaring v1.

**Tasks:**

- [ ] **7.1** Performance baseline: instrument `devpilot graph build` and `preflight` with timing; run on devpilot (~5k LOC), one external ~10k LOC Go repo (e.g., `go-redis`), one ~10k LOC TS repo (e.g., `typescript-eslint`), record times in `docs/plans/2026-05-19-devpilot-graph-bench.md`.
- [ ] **7.2** Regression gate: CI workflow asserts build/update/preflight stay within 25% of recorded baselines.
- [ ] **7.3** Dogfood three real PRs (suggested: most recent merged PR in devpilot, one in `code-review-graph`, one in an external OSS repo of choice). Manually run pr-review skill with preflight enabled and compare findings to a baseline run without preflight. Capture results in `docs/plans/2026-05-19-devpilot-graph-dogfood.md`.
- [ ] **7.4** Documentation: README section under devpilot project describing `devpilot graph` subcommands; brief usage docs in `docs/`.
- [ ] **7.5** Release gate sign-off: walk through spec §10 checklist; all seven items checked.
- [ ] **7.6** Tag release, update `CHANGELOG.md` / equivalent.

**Phase 7 acceptance:** All seven items in spec §10 release gate checked off in writing.

---

## Cross-phase notes

**Test fixtures live under `internal/graph/parser/testdata/`** organized by language. Each fixture should be runnable as an independent Go package or TS/JS project where applicable so that LSP drivers can operate on them.

**Snapshot tests live alongside the code they cover.** Use `github.com/sebdah/goldie/v2` or hand-rolled diff against `testdata/snapshots/`. Update snapshots only via explicit flag (`-update`) to prevent silent regression.

**Commit cadence:** every passing test = one commit. Phase boundary = a tag (e.g., `graph-phase-1`).

**Branching:** work on `feat/graph` branch; rebase against `main` weekly to avoid drift.

**Re-plan trigger:** before starting each new phase (2 through 7), re-invoke `superpowers:writing-plans` against the specific phase to expand its half-day tasks into bite-sized TDD steps with the most recent context. The half-day-resolution list here is the contract; the expansion is the execution.

---

## Self-review

- **Spec coverage:** every spec §2 in-scope item maps to a phase: parsers (P1/P2), SQLite store + cache (P1/P2), 9 CLI subcommands (P4), preflight JSON (P3/P4), grep fallback (P6), skill step 1.5 (P6). LSP testing (P5) covers spec §9 Layer 1; envelope/schema testing (P4) covers Layer 2; fanout assertions (P6) cover Layer 3.
- **Type consistency:** `Node`, `Edge`, `ParseResult`, `Store.CallersOf` are defined in Phase 1 and re-used by name in Phase 3 task descriptions.
- **No placeholders:** Phase 1 tasks have complete code blocks. Phase 2+ tasks are half-day work units, not placeholders — they have explicit file paths, behaviors, and acceptance criteria, and each will be expanded via re-plan before execution.
- **Scope:** This is one coherent subsystem (parser → store → query → CLI → skill). Phases are sequential and tightly coupled — decomposition into independent sub-projects is not viable. The phased structure handles the size.
