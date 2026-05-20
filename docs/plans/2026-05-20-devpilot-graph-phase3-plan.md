# Devpilot Graph Phase 3 — Query Layer Bite-Sized Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement every read-side query that downstream consumers (CLI in Phase 4, `devpilot-pr-review` skill in Phase 6) will use, including the `preflight` composite. All queries operate against a populated `*store.Store`; no parsing or git mutation happens here (except `DetectChanges`/`Preflight` which shell out to `git` read-only).

**Architecture:** A new `internal/graph/query` package. Each verb is a single file with one exported function plus tiny internal helpers. Queries take a `*store.Store` (or a narrower interface defined locally) and primitive args; they return typed Go structs that map 1:1 to the Phase 4 JSON envelope shape. Composite queries (`Preflight`) call other query functions in this same package — no cycles, since `Preflight` lives in its own file. `DetectChanges` shells out to `git diff --name-status` and `git show :path` to read base/head trees, then joins with the graph by file path.

**Tech Stack:**
- Go 1.25.6, module `github.com/siyuqian/devpilot`
- `modernc.org/sqlite` (via existing `internal/graph/store`)
- `os/exec` for `git diff --name-status` + `git show`
- Stdlib `crypto/sha256` for signature hashes

**Conventions inherited from Phases 1–2:**
- One passing test → one commit (`feat(graph/query): <verb> <noun>` for new behavior; `test(graph/query): ...` for tests-only).
- All `fmt.Errorf` at layer boundaries wraps with `%w`.
- Tests are table-driven with named subtests where >1 case; small fixtures are built via `store.InsertNodes` / `store.InsertEdges` directly (do NOT spin up parsers in query tests — keep them hermetic).
- No mocking of our own packages. The single exception is `detect_changes.go`: factor the `git` invocation through a function variable so tests can swap it.
- Queries MUST NOT mutate the store; any test that needs to assert no-write should use a `t.TempDir()`-backed store and re-open read-only after.
- Run `make lint` and `make test` before committing each task.

---

## File Structure

```
internal/graph/query/
├── reader.go               (new — narrow read-only interface satisfied by *store.Store)
├── reader_test.go          (new — interface compile check)
├── callers.go              (new — CallersOf with depth)
├── callers_test.go         (new)
├── callees.go              (new — CalleesOf with depth)
├── callees_test.go         (new)
├── tests.go                (new — TestsFor)
├── tests_test.go           (new)
├── impact.go               (new — ImpactRadius over file set)
├── impact_test.go          (new)
├── hubs.go                 (new — Hubs by inbound calls threshold)
├── hubs_test.go            (new)
├── implementors.go         (new — ImplementorsOf interface)
├── implementors_test.go    (new)
├── context.go              (new — Context source snippet + caller snippets)
├── context_test.go         (new)
├── detect_changes.go       (new — git diff + signature_hash compare)
├── detect_changes_test.go  (new)
├── risk.go                 (new — RiskScore formula)
├── risk_test.go            (new)
├── preflight.go            (new — composite)
├── preflight_test.go       (new)
└── testdata/
    ├── detect_changes/     (small temp-repo helper fixtures created at runtime)
    └── context/            (a single .go file used by Context tests)
```

Reader interface lives in `reader.go` so each query file can declare its dependency narrowly (e.g., `CallersOf` only needs `EdgesByDst` + `GetNode`); the concrete `*store.Store` satisfies all of them.

---

## Pre-Phase: extend the store with the few accessors the query layer needs

The Phase 1/2 store exposes `GetNode`, `AllNodes`, `InsertNodes`, `InsertEdges`, `CallersOf` (1-hop), `DeleteByPaths`. The query layer needs three additional cheap accessors. We add them here so every Phase 3 task can rely on them.

**Files:**
- Modify: `internal/graph/store/store.go`
- Modify: `internal/graph/store/store_test.go`

### Task 0.1: `EdgesByDst(dst, kind)` and `EdgesBySrc(src, kind)`

- [ ] **Step 1: Write failing test**

Append to `internal/graph/store/store_test.go`:

```go
func TestEdgesByDstAndBySrc(t *testing.T) {
	s := newTestStore(t)
	mustInsertNodes(t, s, []Node{
		{ID: "a", Kind: "function", Path: "a.go", Name: "A", Language: "go"},
		{ID: "b", Kind: "function", Path: "b.go", Name: "B", Language: "go"},
		{ID: "c", Kind: "function", Path: "c.go", Name: "C", Language: "go"},
	})
	mustInsertEdges(t, s, []Edge{
		{Src: "a", Dst: "c", Kind: "calls"},
		{Src: "b", Dst: "c", Kind: "calls"},
		{Src: "a", Dst: "b", Kind: "calls"},
		{Src: "a", Dst: "c", Kind: "tests"},
	})

	t.Run("by_dst_calls", func(t *testing.T) {
		got, err := s.EdgesByDst("c", "calls")
		if err != nil {
			t.Fatal(err)
		}
		want := []Edge{{Src: "a", Dst: "c", Kind: "calls"}, {Src: "b", Dst: "c", Kind: "calls"}}
		if !sameEdges(got, want) {
			t.Errorf("got=%v want=%v", got, want)
		}
	})

	t.Run("by_src_calls", func(t *testing.T) {
		got, err := s.EdgesBySrc("a", "calls")
		if err != nil {
			t.Fatal(err)
		}
		want := []Edge{{Src: "a", Dst: "b", Kind: "calls"}, {Src: "a", Dst: "c", Kind: "calls"}}
		if !sameEdges(got, want) {
			t.Errorf("got=%v want=%v", got, want)
		}
	})
}

func sameEdges(a, b []Edge) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[Edge]int{}
	for _, e := range a {
		m[e]++
	}
	for _, e := range b {
		m[e]--
	}
	for _, v := range m {
		if v != 0 {
			return false
		}
	}
	return true
}

func mustInsertNodes(t *testing.T, s *Store, n []Node) {
	t.Helper()
	if err := s.InsertNodes(n); err != nil {
		t.Fatal(err)
	}
}

func mustInsertEdges(t *testing.T, s *Store, e []Edge) {
	t.Helper()
	if err := s.InsertEdges(e); err != nil {
		t.Fatal(err)
	}
}
```

If `mustInsertNodes`/`mustInsertEdges`/`sameEdges` already exist from Phase 1/2, drop the duplicates; reuse the existing names.

- [ ] **Step 2: Run, expect FAIL** — `EdgesByDst`/`EdgesBySrc` undefined.

```bash
go test ./internal/graph/store/ -run TestEdgesByDstAndBySrc -v
```

- [ ] **Step 3: Implement**

Append to `internal/graph/store/store.go`:

```go
// EdgesByDst returns all edges of the given kind that end at dst.
func (s *Store) EdgesByDst(dst, kind string) ([]Edge, error) {
	rows, err := s.db.Query(`SELECT src, dst, kind FROM edges WHERE dst = ? AND kind = ?`, dst, kind)
	if err != nil {
		return nil, fmt.Errorf("EdgesByDst: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.Src, &e.Dst, &e.Kind); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// EdgesBySrc returns all edges of the given kind that start at src.
func (s *Store) EdgesBySrc(src, kind string) ([]Edge, error) {
	rows, err := s.db.Query(`SELECT src, dst, kind FROM edges WHERE src = ? AND kind = ?`, src, kind)
	if err != nil {
		return nil, fmt.Errorf("EdgesBySrc: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.Src, &e.Dst, &e.Kind); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/store/store.go internal/graph/store/store_test.go
git commit -m "feat(graph/store): EdgesByDst and EdgesBySrc accessors"
```

### Task 0.2: `NodesByPath(path)` and `CountEdgesByKind(dstID, kind)`

- [ ] **Step 1: Failing test**

Append:

```go
func TestNodesByPathAndCountEdges(t *testing.T) {
	s := newTestStore(t)
	mustInsertNodes(t, s, []Node{
		{ID: "a.go::A", Kind: "function", Path: "a.go", Name: "A", Language: "go"},
		{ID: "a.go::B", Kind: "function", Path: "a.go", Name: "B", Language: "go"},
		{ID: "b.go::C", Kind: "function", Path: "b.go", Name: "C", Language: "go"},
	})
	mustInsertEdges(t, s, []Edge{
		{Src: "a.go::A", Dst: "b.go::C", Kind: "calls"},
		{Src: "a.go::B", Dst: "b.go::C", Kind: "calls"},
	})

	t.Run("nodes_by_path", func(t *testing.T) {
		got, err := s.NodesByPath("a.go")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2 nodes, got %d: %+v", len(got), got)
		}
	})

	t.Run("count_edges", func(t *testing.T) {
		n, err := s.CountEdgesByKind("b.go::C", "calls")
		if err != nil {
			t.Fatal(err)
		}
		if n != 2 {
			t.Errorf("want 2, got %d", n)
		}
	})
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Implement**

```go
// NodesByPath returns all nodes whose path equals the given path (excluding the file node itself).
func (s *Store) NodesByPath(path string) ([]Node, error) {
	rows, err := s.db.Query(
		`SELECT id, kind, path, name, container, language, start_line, end_line, is_exported, signature_hash
		 FROM nodes WHERE path = ? AND kind != 'file'`, path)
	if err != nil {
		return nil, fmt.Errorf("NodesByPath: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Node
	for rows.Next() {
		var n Node
		var exp int
		if err := rows.Scan(&n.ID, &n.Kind, &n.Path, &n.Name, &n.Container, &n.Language,
			&n.StartLine, &n.EndLine, &exp, &n.SignatureHash); err != nil {
			return nil, err
		}
		n.IsExported = exp == 1
		out = append(out, n)
	}
	return out, rows.Err()
}

// CountEdgesByKind returns the inbound edge count toward dst of the given kind.
func (s *Store) CountEdgesByKind(dst, kind string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM edges WHERE dst = ? AND kind = ?`, dst, kind).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("CountEdgesByKind: %w", err)
	}
	return n, nil
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(graph/store): NodesByPath and CountEdgesByKind"
```

---

## Task 3.0: `query.Reader` interface

**Files:**
- Create: `internal/graph/query/reader.go`
- Create: `internal/graph/query/reader_test.go`

- [ ] **Step 1: Write failing compile-time check**

`internal/graph/query/reader_test.go`:

```go
package query

import (
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// TestStoreSatisfiesReader is a compile-time assertion that *store.Store implements Reader.
func TestStoreSatisfiesReader(t *testing.T) {
	var _ Reader = (*store.Store)(nil)
}
```

- [ ] **Step 2: Run, expect FAIL** (undefined `Reader`).

```bash
go test ./internal/graph/query/ -run TestStoreSatisfiesReader -v
```

- [ ] **Step 3: Implement `reader.go`**

```go
// Package query implements the read-side graph operations consumed by the
// devpilot CLI and the devpilot-pr-review skill.
//
// All queries are pure functions of a Reader plus primitive arguments; they
// never mutate the underlying store.
package query

import "github.com/siyuqian/devpilot/internal/graph/store"

// Reader is the narrow read-only surface that every query depends on. It is
// satisfied by *store.Store; tests construct in-memory stores to feed it.
type Reader interface {
	GetNode(id string) (store.Node, error)
	NodesByPath(path string) ([]store.Node, error)
	AllNodes() ([]store.Node, error)
	EdgesByDst(dst, kind string) ([]store.Edge, error)
	EdgesBySrc(src, kind string) ([]store.Edge, error)
	CountEdgesByKind(dst, kind string) (int, error)
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/query/reader.go internal/graph/query/reader_test.go
git commit -m "feat(graph/query): Reader interface satisfied by *store.Store"
```

---

## Task 3.1: `CallersOf(r, id, depth)` — BFS over `calls` edges

**Files:**
- Create: `internal/graph/query/callers.go`
- Create: `internal/graph/query/callers_test.go`

- [ ] **Step 1: Write failing test**

```go
package query

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// newStore returns an in-memory store seeded with the given nodes/edges.
func newStore(t *testing.T, nodes []store.Node, edges []store.Edge) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "graph.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.InsertNodes(nodes); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertEdges(edges); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCallersOf(t *testing.T) {
	// chain: a -> b -> c -> d ; also e -> c
	nodes := []store.Node{
		{ID: "a", Kind: "function", Path: "a.go", Name: "a", Language: "go"},
		{ID: "b", Kind: "function", Path: "b.go", Name: "b", Language: "go"},
		{ID: "c", Kind: "function", Path: "c.go", Name: "c", Language: "go"},
		{ID: "d", Kind: "function", Path: "d.go", Name: "d", Language: "go"},
		{ID: "e", Kind: "function", Path: "e.go", Name: "e", Language: "go"},
	}
	edges := []store.Edge{
		{Src: "a", Dst: "b", Kind: "calls"},
		{Src: "b", Dst: "c", Kind: "calls"},
		{Src: "c", Dst: "d", Kind: "calls"},
		{Src: "e", Dst: "c", Kind: "calls"},
	}
	r := newStore(t, nodes, edges)

	t.Run("depth_1", func(t *testing.T) {
		got, err := CallersOf(r, "d", 1)
		if err != nil {
			t.Fatal(err)
		}
		want := []Caller{{ID: "c", Hop: 1}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got=%v want=%v", got, want)
		}
	})

	t.Run("depth_3", func(t *testing.T) {
		got, err := CallersOf(r, "d", 3)
		if err != nil {
			t.Fatal(err)
		}
		sort.Slice(got, func(i, j int) bool {
			if got[i].Hop != got[j].Hop {
				return got[i].Hop < got[j].Hop
			}
			return got[i].ID < got[j].ID
		})
		want := []Caller{
			{ID: "c", Hop: 1},
			{ID: "b", Hop: 2},
			{ID: "e", Hop: 2},
			{ID: "a", Hop: 3},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got=%v want=%v", got, want)
		}
	})

	t.Run("nonexistent_target", func(t *testing.T) {
		got, err := CallersOf(r, "no_such_id", 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("want empty, got %v", got)
		}
	})

	t.Run("cycle_safe", func(t *testing.T) {
		// a -> b -> a forms a cycle; CallersOf("a", 5) must terminate.
		nodes := []store.Node{
			{ID: "a", Kind: "function", Path: "a.go", Name: "a", Language: "go"},
			{ID: "b", Kind: "function", Path: "b.go", Name: "b", Language: "go"},
		}
		edges := []store.Edge{
			{Src: "a", Dst: "b", Kind: "calls"},
			{Src: "b", Dst: "a", Kind: "calls"},
		}
		r := newStore(t, nodes, edges)
		got, err := CallersOf(r, "a", 5)
		if err != nil {
			t.Fatal(err)
		}
		// Expect only "b" (1 hop). "a" is the target itself and must be excluded.
		want := []Caller{{ID: "b", Hop: 1}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got=%v want=%v", got, want)
		}
	})
}
```

- [ ] **Step 2: Run, expect FAIL** (undefined `CallersOf`, `Caller`).

```bash
go test ./internal/graph/query/ -run TestCallersOf -v
```

- [ ] **Step 3: Implement `callers.go`**

```go
package query

import "fmt"

// Caller is a node that (possibly transitively) calls the queried target.
// Hop is the BFS distance from the target (1 = direct caller).
type Caller struct {
	ID  string
	Hop int
}

// CallersOf returns all transitive callers of id up to maxDepth hops via
// `calls` edges, in BFS order. The target itself is never returned.
func CallersOf(r Reader, id string, maxDepth int) ([]Caller, error) {
	if maxDepth < 1 {
		return nil, nil
	}
	seen := map[string]bool{id: true}
	frontier := []string{id}
	var out []Caller
	for hop := 1; hop <= maxDepth && len(frontier) > 0; hop++ {
		var next []string
		for _, cur := range frontier {
			edges, err := r.EdgesByDst(cur, "calls")
			if err != nil {
				return nil, fmt.Errorf("CallersOf at hop %d: %w", hop, err)
			}
			for _, e := range edges {
				if seen[e.Src] {
					continue
				}
				seen[e.Src] = true
				out = append(out, Caller{ID: e.Src, Hop: hop})
				next = append(next, e.Src)
			}
		}
		frontier = next
	}
	return out, nil
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/query/callers.go internal/graph/query/callers_test.go
git commit -m "feat(graph/query): CallersOf with depth-limited BFS"
```

---

## Task 3.2: `CalleesOf(r, id, depth)` — symmetrical BFS

**Files:**
- Create: `internal/graph/query/callees.go`
- Create: `internal/graph/query/callees_test.go`

- [ ] **Step 1: Failing test**

```go
package query

import (
	"reflect"
	"sort"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestCalleesOf(t *testing.T) {
	nodes := []store.Node{
		{ID: "a", Kind: "function", Path: "a.go", Name: "a", Language: "go"},
		{ID: "b", Kind: "function", Path: "b.go", Name: "b", Language: "go"},
		{ID: "c", Kind: "function", Path: "c.go", Name: "c", Language: "go"},
		{ID: "d", Kind: "function", Path: "d.go", Name: "d", Language: "go"},
	}
	edges := []store.Edge{
		{Src: "a", Dst: "b", Kind: "calls"},
		{Src: "b", Dst: "c", Kind: "calls"},
		{Src: "b", Dst: "d", Kind: "calls"},
	}
	r := newStore(t, nodes, edges)

	got, err := CalleesOf(r, "a", 2)
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(got, func(i, j int) bool {
		if got[i].Hop != got[j].Hop {
			return got[i].Hop < got[j].Hop
		}
		return got[i].ID < got[j].ID
	})
	want := []Callee{
		{ID: "b", Hop: 1},
		{ID: "c", Hop: 2},
		{ID: "d", Hop: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got=%v want=%v", got, want)
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Implement `callees.go`**

```go
package query

import "fmt"

// Callee is a node that the queried source (possibly transitively) calls.
type Callee struct {
	ID  string
	Hop int
}

// CalleesOf returns all transitive callees of id up to maxDepth hops via
// `calls` edges. The source itself is excluded.
func CalleesOf(r Reader, id string, maxDepth int) ([]Callee, error) {
	if maxDepth < 1 {
		return nil, nil
	}
	seen := map[string]bool{id: true}
	frontier := []string{id}
	var out []Callee
	for hop := 1; hop <= maxDepth && len(frontier) > 0; hop++ {
		var next []string
		for _, cur := range frontier {
			edges, err := r.EdgesBySrc(cur, "calls")
			if err != nil {
				return nil, fmt.Errorf("CalleesOf at hop %d: %w", hop, err)
			}
			for _, e := range edges {
				if seen[e.Dst] {
					continue
				}
				seen[e.Dst] = true
				out = append(out, Callee{ID: e.Dst, Hop: hop})
				next = append(next, e.Dst)
			}
		}
		frontier = next
	}
	return out, nil
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/query/callees.go internal/graph/query/callees_test.go
git commit -m "feat(graph/query): CalleesOf with depth-limited BFS"
```

---

## Task 3.3: `TestsFor(r, id)` — direct `tests` edge lookup

**Files:**
- Create: `internal/graph/query/tests.go`
- Create: `internal/graph/query/tests_test.go`

- [ ] **Step 1: Failing test**

```go
package query

import (
	"reflect"
	"sort"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestTestsFor(t *testing.T) {
	nodes := []store.Node{
		{ID: "pkg.go::Greet", Kind: "function", Path: "pkg.go", Name: "Greet", Language: "go", IsExported: true},
		{ID: "pkg_test.go::TestGreet", Kind: "function", Path: "pkg_test.go", Name: "TestGreet", Language: "go"},
		{ID: "pkg_test.go::TestGreetEdge", Kind: "function", Path: "pkg_test.go", Name: "TestGreetEdge", Language: "go"},
	}
	edges := []store.Edge{
		{Src: "pkg_test.go::TestGreet", Dst: "pkg.go::Greet", Kind: "tests"},
		{Src: "pkg_test.go::TestGreetEdge", Dst: "pkg.go::Greet", Kind: "tests"},
	}
	r := newStore(t, nodes, edges)

	got, err := TestsFor(r, "pkg.go::Greet")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	want := []string{"pkg_test.go::TestGreet", "pkg_test.go::TestGreetEdge"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got=%v want=%v", got, want)
	}

	t.Run("empty_when_untested", func(t *testing.T) {
		empty, err := TestsFor(r, "pkg_test.go::TestGreetEdge")
		if err != nil {
			t.Fatal(err)
		}
		if len(empty) != 0 {
			t.Errorf("want empty, got %v", empty)
		}
	})
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Implement `tests.go`**

```go
package query

import "fmt"

// TestsFor returns the IDs of test symbols that exercise id (i.e., nodes
// connected to id by a `tests` edge). Order is insertion order from SQLite.
func TestsFor(r Reader, id string) ([]string, error) {
	edges, err := r.EdgesByDst(id, "tests")
	if err != nil {
		return nil, fmt.Errorf("TestsFor: %w", err)
	}
	out := make([]string, 0, len(edges))
	for _, e := range edges {
		out = append(out, e.Src)
	}
	return out, nil
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/query/tests.go internal/graph/query/tests_test.go
git commit -m "feat(graph/query): TestsFor direct tests-edge lookup"
```

---

## Task 3.4: `ImpactRadius(r, files, depth)` — union of CallersOf over symbols in files

**Files:**
- Create: `internal/graph/query/impact.go`
- Create: `internal/graph/query/impact_test.go`

- [ ] **Step 1: Failing test**

```go
package query

import (
	"reflect"
	"sort"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestImpactRadius(t *testing.T) {
	// files: a.go contains A, b.go contains B; external callers x, y call A and B respectively.
	nodes := []store.Node{
		{ID: "a.go", Kind: "file", Path: "a.go", Name: "a.go", Language: "go"},
		{ID: "a.go::A", Kind: "function", Path: "a.go", Name: "A", Language: "go"},
		{ID: "b.go", Kind: "file", Path: "b.go", Name: "b.go", Language: "go"},
		{ID: "b.go::B", Kind: "function", Path: "b.go", Name: "B", Language: "go"},
		{ID: "x.go::X", Kind: "function", Path: "x.go", Name: "X", Language: "go"},
		{ID: "y.go::Y", Kind: "function", Path: "y.go", Name: "Y", Language: "go"},
		{ID: "z.go::Z", Kind: "function", Path: "z.go", Name: "Z", Language: "go"},
	}
	edges := []store.Edge{
		{Src: "x.go::X", Dst: "a.go::A", Kind: "calls"},
		{Src: "y.go::Y", Dst: "b.go::B", Kind: "calls"},
		{Src: "z.go::Z", Dst: "x.go::X", Kind: "calls"}, // 2-hop into A
	}
	r := newStore(t, nodes, edges)

	t.Run("depth_1", func(t *testing.T) {
		got, err := ImpactRadius(r, []string{"a.go", "b.go"}, 1)
		if err != nil {
			t.Fatal(err)
		}
		sort.Slice(got.Symbols, func(i, j int) bool { return got.Symbols[i].ID < got.Symbols[j].ID })
		want := Impact{
			ChangedSymbols: []string{"a.go::A", "b.go::B"},
			Symbols: []Caller{
				{ID: "x.go::X", Hop: 1},
				{ID: "y.go::Y", Hop: 1},
			},
		}
		sort.Strings(got.ChangedSymbols)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got=%+v want=%+v", got, want)
		}
	})

	t.Run("depth_2_picks_up_z", func(t *testing.T) {
		got, err := ImpactRadius(r, []string{"a.go"}, 2)
		if err != nil {
			t.Fatal(err)
		}
		ids := map[string]bool{}
		for _, s := range got.Symbols {
			ids[s.ID] = true
		}
		if !ids["x.go::X"] || !ids["z.go::Z"] {
			t.Errorf("want x and z in callers, got %v", got.Symbols)
		}
	})
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Implement `impact.go`**

```go
package query

import "fmt"

// Impact is the result of an impact-radius query: the symbols owned by the
// changed file set, plus the union of their transitive callers up to depth.
type Impact struct {
	ChangedSymbols []string
	Symbols        []Caller
}

// ImpactRadius returns the union of CallersOf for every symbol contained in
// the given files, up to maxDepth hops.
func ImpactRadius(r Reader, files []string, maxDepth int) (Impact, error) {
	out := Impact{}
	seen := map[string]int{} // id -> min hop
	for _, f := range files {
		nodes, err := r.NodesByPath(f)
		if err != nil {
			return Impact{}, fmt.Errorf("ImpactRadius: %w", err)
		}
		for _, n := range nodes {
			out.ChangedSymbols = append(out.ChangedSymbols, n.ID)
			callers, err := CallersOf(r, n.ID, maxDepth)
			if err != nil {
				return Impact{}, err
			}
			for _, c := range callers {
				if prev, ok := seen[c.ID]; !ok || c.Hop < prev {
					seen[c.ID] = c.Hop
				}
			}
		}
	}
	for id, hop := range seen {
		out.Symbols = append(out.Symbols, Caller{ID: id, Hop: hop})
	}
	return out, nil
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/query/impact.go internal/graph/query/impact_test.go
git commit -m "feat(graph/query): ImpactRadius union over changed-file symbols"
```

---

## Task 3.5: `Hubs(r, threshold)` — group inbound `calls` by dst

**Files:**
- Create: `internal/graph/query/hubs.go`
- Create: `internal/graph/query/hubs_test.go`

This query needs a slightly wider Reader than the interface offers — it scans across all edges. We add a `Hubs` method on `*store.Store` and have `query.Hubs` call into a small extension interface defined in `hubs.go`.

- [ ] **Step 1: Failing test**

```go
package query

import (
	"reflect"
	"sort"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestHubs(t *testing.T) {
	nodes := []store.Node{
		{ID: "h", Kind: "function", Path: "h.go", Name: "h", Language: "go"},
		{ID: "a", Kind: "function", Path: "a.go", Name: "a", Language: "go"},
		{ID: "b", Kind: "function", Path: "b.go", Name: "b", Language: "go"},
		{ID: "c", Kind: "function", Path: "c.go", Name: "c", Language: "go"},
		{ID: "d", Kind: "function", Path: "d.go", Name: "d", Language: "go"},
	}
	edges := []store.Edge{
		{Src: "a", Dst: "h", Kind: "calls"},
		{Src: "b", Dst: "h", Kind: "calls"},
		{Src: "c", Dst: "h", Kind: "calls"},
		{Src: "a", Dst: "d", Kind: "calls"}, // d gets 1 caller; below threshold
	}
	r := newStore(t, nodes, edges)

	got, err := Hubs(r, 3)
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })
	want := []Hub{{ID: "h", CallerCount: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got=%v want=%v", got, want)
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Add `HubsByCalls` to the store**

Append to `internal/graph/store/store.go`:

```go
// HubsByCalls returns dst IDs with at least minCallers inbound `calls` edges,
// ordered by caller count descending then id ascending for determinism.
func (s *Store) HubsByCalls(minCallers int) ([]struct {
	ID    string
	Count int
}, error) {
	rows, err := s.db.Query(
		`SELECT dst, COUNT(*) AS c FROM edges WHERE kind='calls'
		   GROUP BY dst HAVING c >= ?
		   ORDER BY c DESC, dst ASC`, minCallers)
	if err != nil {
		return nil, fmt.Errorf("HubsByCalls: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []struct {
		ID    string
		Count int
	}
	for rows.Next() {
		var e struct {
			ID    string
			Count int
		}
		if err := rows.Scan(&e.ID, &e.Count); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
```

- [ ] **Step 4: Implement `hubs.go`**

```go
package query

import "fmt"

// Hub is a high-fanin node — a frequent call target.
type Hub struct {
	ID          string
	CallerCount int
}

// hubReader extends Reader with the hub aggregation that callers/callees do
// not need. It is satisfied by *store.Store.
type hubReader interface {
	Reader
	HubsByCalls(minCallers int) ([]struct {
		ID    string
		Count int
	}, error)
}

// Hubs returns all nodes whose inbound `calls` edge count is >= threshold.
func Hubs(r Reader, threshold int) ([]Hub, error) {
	hr, ok := r.(hubReader)
	if !ok {
		return nil, fmt.Errorf("Hubs: reader does not implement HubsByCalls")
	}
	rows, err := hr.HubsByCalls(threshold)
	if err != nil {
		return nil, fmt.Errorf("Hubs: %w", err)
	}
	out := make([]Hub, 0, len(rows))
	for _, row := range rows {
		out = append(out, Hub{ID: row.ID, CallerCount: row.Count})
	}
	return out, nil
}
```

- [ ] **Step 5: Run, expect PASS.**

- [ ] **Step 6: Commit**

```bash
git add internal/graph/store/store.go internal/graph/query/hubs.go internal/graph/query/hubs_test.go
git commit -m "feat(graph/query): Hubs by inbound calls threshold"
```

---

## Task 3.6: `ImplementorsOf(r, interfaceID)`

**Files:**
- Create: `internal/graph/query/implementors.go`
- Create: `internal/graph/query/implementors_test.go`

- [ ] **Step 1: Failing test**

```go
package query

import (
	"reflect"
	"sort"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestImplementorsOf(t *testing.T) {
	nodes := []store.Node{
		{ID: "p.go::Speaker", Kind: "interface", Path: "p.go", Name: "Speaker", Language: "go", IsExported: true},
		{ID: "p.go::Console", Kind: "struct", Path: "p.go", Name: "Console", Language: "go", IsExported: true},
		{ID: "p.go::Silent", Kind: "struct", Path: "p.go", Name: "Silent", Language: "go", IsExported: true},
	}
	edges := []store.Edge{
		{Src: "p.go::Console", Dst: "p.go::Speaker", Kind: "implements"},
		{Src: "p.go::Silent", Dst: "p.go::Speaker", Kind: "implements"},
	}
	r := newStore(t, nodes, edges)

	got, err := ImplementorsOf(r, "p.go::Speaker")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	want := []string{"p.go::Console", "p.go::Silent"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got=%v want=%v", got, want)
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Implement `implementors.go`**

```go
package query

import "fmt"

// ImplementorsOf returns the IDs of types that implement the given interface
// (i.e., nodes connected to ifaceID by an `implements` edge).
func ImplementorsOf(r Reader, ifaceID string) ([]string, error) {
	edges, err := r.EdgesByDst(ifaceID, "implements")
	if err != nil {
		return nil, fmt.Errorf("ImplementorsOf: %w", err)
	}
	out := make([]string, 0, len(edges))
	for _, e := range edges {
		out = append(out, e.Src)
	}
	return out, nil
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/query/implementors.go internal/graph/query/implementors_test.go
git commit -m "feat(graph/query): ImplementorsOf interface lookup"
```

---

## Task 3.7: `Context(r, id, depth, repoRoot)` — source snippet + caller snippets

**Files:**
- Create: `internal/graph/query/context.go`
- Create: `internal/graph/query/context_test.go`
- Create: `internal/graph/query/testdata/context/sample.go`

- [ ] **Step 1: Create fixture**

`internal/graph/query/testdata/context/sample.go`:

```go
package sample

func Greet(name string) string {
	return "hi " + name
}

func CallGreet() string {
	return Greet("world")
}
```

- [ ] **Step 2: Failing test**

`context_test.go`:

```go
package query

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestContext(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("testdata", "context"))
	if err != nil {
		t.Fatal(err)
	}
	nodes := []store.Node{
		{ID: "sample.go::Greet", Kind: "function", Path: "sample.go", Name: "Greet",
			Language: "go", IsExported: true, StartLine: 3, EndLine: 5},
		{ID: "sample.go::CallGreet", Kind: "function", Path: "sample.go", Name: "CallGreet",
			Language: "go", IsExported: true, StartLine: 7, EndLine: 9},
	}
	edges := []store.Edge{
		{Src: "sample.go::CallGreet", Dst: "sample.go::Greet", Kind: "calls"},
	}
	r := newStore(t, nodes, edges)

	t.Run("depth_0_target_only", func(t *testing.T) {
		ctx, err := Context(r, "sample.go::Greet", 0, root)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(ctx.Target.Source, "return \"hi \" + name") {
			t.Errorf("target snippet missing body, got %q", ctx.Target.Source)
		}
		if len(ctx.Callers) != 0 {
			t.Errorf("want no callers at depth 0, got %v", ctx.Callers)
		}
	})

	t.Run("depth_1_includes_caller", func(t *testing.T) {
		ctx, err := Context(r, "sample.go::Greet", 1, root)
		if err != nil {
			t.Fatal(err)
		}
		if len(ctx.Callers) != 1 {
			t.Fatalf("want 1 caller snippet, got %d", len(ctx.Callers))
		}
		if !strings.Contains(ctx.Callers[0].Source, "return Greet(\"world\")") {
			t.Errorf("caller snippet wrong: %q", ctx.Callers[0].Source)
		}
	})

	t.Run("unknown_id", func(t *testing.T) {
		_, err := Context(r, "nope", 0, root)
		if err == nil {
			t.Error("want error for unknown id")
		}
	})
}
```

- [ ] **Step 3: Run, expect FAIL.**

- [ ] **Step 4: Implement `context.go`**

```go
package query

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContextResult is the return type of Context: the target's source snippet
// plus, optionally, snippets for its direct callers.
type ContextResult struct {
	Target  Snippet
	Callers []Snippet
}

// Snippet captures a node's source-line range from disk.
type Snippet struct {
	ID        string
	Path      string
	StartLine int
	EndLine   int
	Source    string
}

// Context returns the source snippet for id, optionally including caller
// snippets when depth >= 1. repoRoot is the absolute directory that node
// paths are relative to.
func Context(r Reader, id string, depth int, repoRoot string) (ContextResult, error) {
	node, err := r.GetNode(id)
	if err != nil {
		return ContextResult{}, fmt.Errorf("Context: %w", err)
	}
	tgt, err := snippetFromNode(node, repoRoot)
	if err != nil {
		return ContextResult{}, err
	}
	res := ContextResult{Target: tgt}
	if depth < 1 {
		return res, nil
	}
	callers, err := CallersOf(r, id, 1)
	if err != nil {
		return ContextResult{}, err
	}
	for _, c := range callers {
		n, err := r.GetNode(c.ID)
		if err != nil {
			continue // caller without recorded node (synthetic external::...)
		}
		s, err := snippetFromNode(n, repoRoot)
		if err != nil {
			continue
		}
		res.Callers = append(res.Callers, s)
	}
	return res, nil
}

func snippetFromNode(n store.Node, repoRoot string) (Snippet, error) {
	if n.StartLine <= 0 || n.EndLine < n.StartLine {
		return Snippet{ID: n.ID, Path: n.Path, StartLine: n.StartLine, EndLine: n.EndLine}, nil
	}
	abs := filepath.Join(repoRoot, n.Path)
	data, err := os.ReadFile(abs)
	if err != nil {
		return Snippet{}, fmt.Errorf("read %s: %w", abs, err)
	}
	lines := strings.Split(string(data), "\n")
	if n.EndLine > len(lines) {
		return Snippet{}, fmt.Errorf("snippet out of range for %s: end=%d len=%d", n.ID, n.EndLine, len(lines))
	}
	src := strings.Join(lines[n.StartLine-1:n.EndLine], "\n")
	return Snippet{
		ID: n.ID, Path: n.Path,
		StartLine: n.StartLine, EndLine: n.EndLine,
		Source: src,
	}, nil
}
```

Add the `store` import to `context.go`:

```go
import "github.com/siyuqian/devpilot/internal/graph/store"
```

- [ ] **Step 5: Run, expect PASS.**

- [ ] **Step 6: Commit**

```bash
git add internal/graph/query/context.go internal/graph/query/context_test.go internal/graph/query/testdata/context/
git commit -m "feat(graph/query): Context snippet extractor with caller depth"
```

---

## Task 3.8: `DetectChanges(r, repoRoot, base, head)` — git diff + signature compare

**Files:**
- Create: `internal/graph/query/detect_changes.go`
- Create: `internal/graph/query/detect_changes_test.go`

This is the only query that shells out. We isolate the `git` invocation through a package-level function variable (`gitRun`) so tests can supply canned diff output.

- [ ] **Step 1: Failing test**

```go
package query

import (
	"reflect"
	"sort"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestDetectChanges(t *testing.T) {
	// Graph reflects HEAD state.
	nodes := []store.Node{
		{ID: "api/checkout.go::handleCheckout", Kind: "function", Path: "api/checkout.go",
			Name: "handleCheckout", Language: "go", IsExported: true, SignatureHash: "newhash"},
		{ID: "internal/auth/session.go::Validate", Kind: "function", Path: "internal/auth/session.go",
			Name: "Validate", Language: "go", IsExported: true, SignatureHash: "same"},
	}
	r := newStore(t, nodes, nil)

	// Pretend git returns: M api/checkout.go, A internal/new/file.go, D internal/old/gone.go
	prevGitRun := gitRun
	t.Cleanup(func() { gitRun = prevGitRun })
	gitRun = func(repo string, args ...string) ([]byte, error) {
		switch {
		case len(args) > 0 && args[0] == "diff" && contains(args, "--name-status"):
			return []byte("M\tapi/checkout.go\nA\tinternal/new/file.go\nD\tinternal/old/gone.go\n"), nil
		case len(args) > 0 && args[0] == "show":
			// `git show base:path` and `git show head:path`
			// Return a stub body whose signature hash will differ for the M file
			// and be identical for the U entry (none here).
			if contains(args, "BASE:api/checkout.go") {
				return []byte("old-body"), nil
			}
			if contains(args, "HEAD:api/checkout.go") {
				return []byte("new-body"), nil
			}
			return nil, nil
		}
		return nil, nil
	}

	got, err := DetectChanges(r, "/fake/repo", "BASE", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })

	want := []ChangedSymbol{
		{
			ID:         "api/checkout.go::handleCheckout",
			Kind:       "function",
			IsExported: true,
			IsNew:      false,
			ChangeType: "modified",
		},
		// New file's symbols cannot be enumerated from the graph (they'd be in HEAD).
		// Phase 3 keeps DetectChanges focused on graph-known symbols; a new file
		// surfaces as a file-level entry instead.
		{
			ID:         "internal/new/file.go",
			Kind:       "file",
			ChangeType: "added",
			IsNew:      true,
		},
		{
			ID:         "internal/old/gone.go",
			Kind:       "file",
			ChangeType: "removed",
			IsNew:      false,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got=%+v\nwant=%+v", got, want)
	}
}

func contains(s []string, target string) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Implement `detect_changes.go`**

```go
package query

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
)

// ChangedSymbol describes one entry in the diff between base and head.
type ChangedSymbol struct {
	ID         string
	Kind       string
	IsExported bool
	IsNew      bool
	ChangeType string // "added" | "removed" | "modified" | "renamed"
}

// gitRun is the shell-out hook. Tests replace it.
var gitRun = func(repo string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %v: %w", args, err)
	}
	return out, nil
}

// DetectChanges enumerates symbols (or files) changed between base..head.
// For modified files, graph-known symbols are emitted as `modified` if their
// signature_hash differs between the base and head blobs; otherwise no symbol
// entry is produced for that file. Added/removed files surface as file-level
// entries because the in-graph state only reflects head.
func DetectChanges(r Reader, repoRoot, base, head string) ([]ChangedSymbol, error) {
	rangeArg := base + ".." + head
	out, err := gitRun(repoRoot, "diff", "--name-status", "-M", rangeArg)
	if err != nil {
		return nil, fmt.Errorf("DetectChanges: %w", err)
	}

	var changes []ChangedSymbol
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		path := parts[len(parts)-1]
		switch status[0] {
		case 'A':
			changes = append(changes, ChangedSymbol{ID: path, Kind: "file", ChangeType: "added", IsNew: true})
		case 'D':
			changes = append(changes, ChangedSymbol{ID: path, Kind: "file", ChangeType: "removed"})
		case 'R':
			changes = append(changes, ChangedSymbol{ID: path, Kind: "file", ChangeType: "renamed"})
		case 'M':
			modified, err := modifiedSymbols(r, repoRoot, base, head, path)
			if err != nil {
				return nil, err
			}
			changes = append(changes, modified...)
		}
	}
	return changes, nil
}

func modifiedSymbols(r Reader, repoRoot, base, head, path string) ([]ChangedSymbol, error) {
	nodes, err := r.NodesByPath(path)
	if err != nil {
		return nil, err
	}
	baseBlob, err := gitRun(repoRoot, "show", base+":"+path)
	if err != nil {
		// File didn't exist at base — treat as added even though status was M
		// (can happen with rename detection edge cases).
		baseBlob = nil
	}
	headBlob, err := gitRun(repoRoot, "show", head+":"+path)
	if err != nil {
		headBlob = nil
	}
	if hashBytes(baseBlob) == hashBytes(headBlob) {
		return nil, nil
	}
	out := make([]ChangedSymbol, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, ChangedSymbol{
			ID:         n.ID,
			Kind:       n.Kind,
			IsExported: n.IsExported,
			ChangeType: "modified",
		})
	}
	return out, nil
}

func hashBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/query/detect_changes.go internal/graph/query/detect_changes_test.go
git commit -m "feat(graph/query): DetectChanges via git diff + signature hash"
```

---

## Task 3.9: `RiskScore(s)` — pure formula from spec §6

**Files:**
- Create: `internal/graph/query/risk.go`
- Create: `internal/graph/query/risk_test.go`

- [ ] **Step 1: Failing test**

```go
package query

import "testing"

func TestRiskScore(t *testing.T) {
	cases := []struct {
		name string
		in   RiskInputs
		want int
	}{
		{"none", RiskInputs{}, 0},
		{"exported_only", RiskInputs{IsExported: true}, 2},
		{"hub_only", RiskInputs{InHub: true}, 3},
		{"interface_change_only", RiskInputs{InterfaceChange: true}, 3},
		{"untested_only", RiskInputs{Untested: true}, 1},
		{"all_factors", RiskInputs{IsExported: true, InHub: true, InterfaceChange: true, Untested: true}, 9},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RiskScore(c.in); got != c.want {
				t.Errorf("got=%d want=%d", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Implement `risk.go`**

```go
package query

// RiskInputs are the four boolean factors that feed RiskScore. The formula
// matches §6 of the graph design doc: 2*exported + 3*hub + 3*iface + 1*untested.
type RiskInputs struct {
	IsExported      bool
	InHub           bool
	InterfaceChange bool
	Untested        bool
}

// RiskScore is the deterministic risk weight used by Preflight to rank
// changed symbols. Higher means higher review priority.
func RiskScore(in RiskInputs) int {
	score := 0
	if in.IsExported {
		score += 2
	}
	if in.InHub {
		score += 3
	}
	if in.InterfaceChange {
		score += 3
	}
	if in.Untested {
		score += 1
	}
	return score
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/query/risk.go internal/graph/query/risk_test.go
git commit -m "feat(graph/query): RiskScore formula per spec §6"
```

---

## Task 3.10: `Preflight(r, opts)` — composite producing the spec §6 payload

**Files:**
- Create: `internal/graph/query/preflight.go`
- Create: `internal/graph/query/preflight_test.go`

The composite output mirrors the JSON contract in `docs/plans/2026-05-19-devpilot-graph-design.md` §6 exactly. The Phase 4 CLI command will marshal this struct into the envelope without further transformation.

### Task 3.10a: types + community derivation + per-symbol enrichment

- [ ] **Step 1: Failing test for community heuristic**

`preflight_test.go`:

```go
package query

import (
	"reflect"
	"sort"
	"testing"
)

func TestCommunityFromPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"internal/payment/processor.go", "internal/payment"},
		{"api/checkout.go", "api"},
		{"cmd/devpilot/main.go", "cmd/devpilot"},
		{"main.go", ""},
		{"internal/a/b/c/d/e.go", "internal/a/b"}, // depth cap 3
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := communityFromPath(c.in); got != c.want {
				t.Errorf("got=%q want=%q", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Stub out `preflight.go` with types + helper**

```go
package query

import (
	"sort"
	"strings"
)

// PreflightInput configures Preflight.
type PreflightInput struct {
	RepoRoot      string
	Base, Head    string
	HubThreshold  int    // default 10
	CallerSample  int    // default 10
	SymbolBudget  int    // default 50 (top-N by risk)
}

// PreflightResult mirrors §6 of the design doc; field names map to JSON
// keys when emitted via the envelope.
type PreflightResult struct {
	Mode              string                  `json:"mode"`
	Graph             GraphMeta               `json:"graph"`
	ChangedSymbols    []ChangedSymbolDetail   `json:"changed_symbols"`
	CrossCommunity    []CrossCommunityEdge    `json:"cross_community_edges"`
	RiskSummary       RiskSummary             `json:"risk_summary"`
	TruncatedSymbols  []string                `json:"truncated_symbols"`
}

type GraphMeta struct {
	Freshness    Freshness `json:"freshness"`
	Languages    []string  `json:"languages"`
	SkippedFiles []string  `json:"skipped_files"`
}

type Freshness struct {
	CoversBaseSHA bool `json:"covers_base_sha"`
	StaleFiles    int  `json:"stale_files"`
}

type ChangedSymbolDetail struct {
	ID              string         `json:"id"`
	Kind            string         `json:"kind"`
	IsExported      bool           `json:"is_exported"`
	IsNew           bool           `json:"is_new"`
	ChangeType      string         `json:"change_type"`
	Callers         CallerSummary  `json:"callers"`
	CalleesChanged  []string       `json:"callees_changed"`
	Tests           TestSummary    `json:"tests"`
	ImplementorsOf  []string       `json:"implementors_of"`
	Implements      []string       `json:"implements"`
	Community       string         `json:"community"`
	RiskFactors     []string       `json:"risk_factors"`
	Risk            int            `json:"-"` // used for sorting/truncation; not in §6 schema
}

type CallerSummary struct {
	Count  int      `json:"count"`
	InHub  bool     `json:"in_hub"`
	Sample []string `json:"sample"`
}

type TestSummary struct {
	HasTests    bool     `json:"has_tests"`
	TestSymbols []string `json:"test_symbols"`
}

type CrossCommunityEdge struct {
	From       string   `json:"from"`
	To         string   `json:"to"`
	CountAdded int      `json:"count_added"`
	Samples    []string `json:"samples"`
}

type RiskSummary struct {
	HubNodesModified         int `json:"hub_nodes_modified"`
	UntestedPublicChanges    int `json:"untested_public_changes"`
	InterfaceChanges         int `json:"interface_changes"`
	NewCrossCommunityEdges   int `json:"new_cross_community_edges"`
}

// communityFromPath returns the shallowest directory containing the file,
// capped at depth 3, per design §6 "Community definition".
func communityFromPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) <= 1 {
		return ""
	}
	dirs := parts[:len(parts)-1]
	if len(dirs) > 3 {
		dirs = dirs[:3]
	}
	return strings.Join(dirs, "/")
}
```

Add the missing import:

```go
import "path/filepath"
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add internal/graph/query/preflight.go internal/graph/query/preflight_test.go
git commit -m "feat(graph/query): Preflight types and community heuristic"
```

### Task 3.10b: per-symbol enrichment

- [ ] **Step 1: Failing test**

Append to `preflight_test.go`:

```go
import (
	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestEnrichChangedSymbol(t *testing.T) {
	nodes := []store.Node{
		{ID: "internal/payment/p.go::Charge", Kind: "method", Path: "internal/payment/p.go",
			Name: "Charge", Container: "PaymentProcessor", Language: "go", IsExported: true},
		{ID: "api/checkout.go::handleCheckout", Kind: "function", Path: "api/checkout.go",
			Name: "handleCheckout", Language: "go", IsExported: true},
		{ID: "internal/payment/p_test.go::TestCharge", Kind: "function", Path: "internal/payment/p_test.go",
			Name: "TestCharge", Language: "go"},
	}
	edges := []store.Edge{
		{Src: "api/checkout.go::handleCheckout", Dst: "internal/payment/p.go::Charge", Kind: "calls"},
		{Src: "internal/payment/p_test.go::TestCharge", Dst: "internal/payment/p.go::Charge", Kind: "tests"},
	}
	r := newStore(t, nodes, edges)

	in := ChangedSymbol{
		ID: "internal/payment/p.go::Charge", Kind: "method",
		IsExported: true, ChangeType: "modified",
	}
	got, err := enrichChangedSymbol(r, in, hubSet{}, 10)
	if err != nil {
		t.Fatal(err)
	}

	if got.Community != "internal/payment" {
		t.Errorf("community=%q", got.Community)
	}
	if got.Callers.Count != 1 || len(got.Callers.Sample) != 1 ||
		got.Callers.Sample[0] != "api/checkout.go::handleCheckout" {
		t.Errorf("callers=%+v", got.Callers)
	}
	if !got.Tests.HasTests || len(got.Tests.TestSymbols) != 1 {
		t.Errorf("tests=%+v", got.Tests)
	}
	wantFactors := []string{} // exported but tested, not in hub set, not interface
	sort.Strings(got.RiskFactors)
	if !reflect.DeepEqual(got.RiskFactors, wantFactors) && len(got.RiskFactors) != 0 {
		t.Errorf("risk factors=%v", got.RiskFactors)
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Implement enrichment**

Append to `preflight.go`:

```go
// hubSet is a quick membership lookup of hub-node IDs.
type hubSet map[string]bool

func (h hubSet) contains(id string) bool { return h[id] }

func enrichChangedSymbol(r Reader, ch ChangedSymbol, hubs hubSet, callerSample int) (ChangedSymbolDetail, error) {
	out := ChangedSymbolDetail{
		ID:         ch.ID,
		Kind:       ch.Kind,
		IsExported: ch.IsExported,
		IsNew:      ch.IsNew,
		ChangeType: ch.ChangeType,
	}
	node, err := r.GetNode(ch.ID)
	if err == nil {
		out.Community = communityFromPath(node.Path)
	} else {
		out.Community = communityFromPath(ch.ID)
	}

	count, err := r.CountEdgesByKind(ch.ID, "calls")
	if err != nil {
		return out, err
	}
	out.Callers.Count = count
	out.Callers.InHub = hubs.contains(ch.ID)

	callerEdges, err := r.EdgesByDst(ch.ID, "calls")
	if err != nil {
		return out, err
	}
	sample := pickCallerSample(r, callerEdges, callerSample)
	out.Callers.Sample = sample

	tests, err := TestsFor(r, ch.ID)
	if err != nil {
		return out, err
	}
	out.Tests = TestSummary{HasTests: len(tests) > 0, TestSymbols: tests}

	impls, err := ImplementorsOf(r, ch.ID)
	if err != nil {
		return out, err
	}
	if len(impls) > 0 {
		out.ImplementorsOf = impls
	}

	// `implements` edges originating from this symbol (struct/class) toward interfaces.
	implEdges, err := r.EdgesBySrc(ch.ID, "implements")
	if err != nil {
		return out, err
	}
	for _, e := range implEdges {
		out.Implements = append(out.Implements, e.Dst)
	}

	// Risk factors + score.
	factors := riskFactors(out, hubs)
	out.RiskFactors = factors
	out.Risk = RiskScore(RiskInputs{
		IsExported:      out.IsExported,
		InHub:           out.Callers.InHub,
		InterfaceChange: containsString(factors, "interface_change"),
		Untested:        !out.Tests.HasTests && out.IsExported,
	})
	return out, nil
}

func pickCallerSample(r Reader, edges []store.Edge, limit int) []string {
	type entry struct {
		id       string
		exported bool
	}
	pool := make([]entry, 0, len(edges))
	for _, e := range edges {
		n, err := r.GetNode(e.Src)
		if err != nil {
			pool = append(pool, entry{id: e.Src})
			continue
		}
		pool = append(pool, entry{id: e.Src, exported: n.IsExported})
	}
	sort.Slice(pool, func(i, j int) bool {
		if pool[i].exported != pool[j].exported {
			return pool[i].exported // true first
		}
		return pool[i].id < pool[j].id
	})
	if limit > 0 && len(pool) > limit {
		pool = pool[:limit]
	}
	out := make([]string, len(pool))
	for i, e := range pool {
		out[i] = e.id
	}
	return out
}

func riskFactors(d ChangedSymbolDetail, hubs hubSet) []string {
	var f []string
	if d.IsExported && !d.Tests.HasTests {
		f = append(f, "untested_public")
	}
	if hubs.contains(d.ID) {
		f = append(f, "hub")
	}
	if d.Kind == "interface" || len(d.ImplementorsOf) > 0 {
		f = append(f, "interface_change")
	}
	return f
}

func containsString(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
```

Add the `store` import to `preflight.go` if not already imported.

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(graph/query): preflight enrichChangedSymbol with callers/tests/impls"
```

### Task 3.10c: composite + ranking + truncation + cross-community edges

- [ ] **Step 1: Failing test**

Append:

```go
func TestPreflightComposite(t *testing.T) {
	nodes := []store.Node{
		{ID: "internal/payment/p.go::Charge", Kind: "method", Path: "internal/payment/p.go",
			Name: "Charge", Container: "PaymentProcessor", Language: "go", IsExported: true},
		{ID: "api/checkout.go::handleCheckout", Kind: "function", Path: "api/checkout.go",
			Name: "handleCheckout", Language: "go", IsExported: true},
		{ID: "internal/payment/p.go::Helper", Kind: "function", Path: "internal/payment/p.go",
			Name: "Helper", Language: "go", IsExported: false},
	}
	edges := []store.Edge{
		{Src: "api/checkout.go::handleCheckout", Dst: "internal/payment/p.go::Charge", Kind: "calls"},
	}
	r := newStore(t, nodes, edges)

	prevGitRun := gitRun
	t.Cleanup(func() { gitRun = prevGitRun })
	gitRun = func(repo string, args ...string) ([]byte, error) {
		switch args[0] {
		case "diff":
			return []byte("M\tinternal/payment/p.go\n"), nil
		case "show":
			if contains(args, "BASE:internal/payment/p.go") {
				return []byte("old"), nil
			}
			return []byte("new"), nil
		}
		return nil, nil
	}

	res, err := Preflight(r, PreflightInput{
		RepoRoot: "/fake", Base: "BASE", Head: "HEAD",
		HubThreshold: 10, CallerSample: 10, SymbolBudget: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode == "" {
		t.Error("mode must be set")
	}
	if len(res.ChangedSymbols) != 2 {
		t.Fatalf("want 2 changed symbols, got %d (%+v)", len(res.ChangedSymbols), res.ChangedSymbols)
	}
	// First should be the exported one (Charge), higher risk.
	if res.ChangedSymbols[0].ID != "internal/payment/p.go::Charge" {
		t.Errorf("ranking wrong: %+v", res.ChangedSymbols)
	}
	// One cross-community edge: api → internal/payment.
	if len(res.CrossCommunity) != 1 || res.CrossCommunity[0].From != "api" ||
		res.CrossCommunity[0].To != "internal/payment" {
		t.Errorf("cross_community=%+v", res.CrossCommunity)
	}
}
```

- [ ] **Step 2: Run, expect FAIL.**

- [ ] **Step 3: Implement `Preflight`**

Append to `preflight.go`:

```go
// Preflight composes DetectChanges + enrichment + cross-community detection
// into the §6 payload. It does not write anything.
func Preflight(r Reader, in PreflightInput) (PreflightResult, error) {
	if in.HubThreshold <= 0 {
		in.HubThreshold = 10
	}
	if in.CallerSample <= 0 {
		in.CallerSample = 10
	}
	if in.SymbolBudget <= 0 {
		in.SymbolBudget = 50
	}

	changes, err := DetectChanges(r, in.RepoRoot, in.Base, in.Head)
	if err != nil {
		return PreflightResult{}, err
	}

	hubs, err := Hubs(r, in.HubThreshold)
	if err != nil {
		return PreflightResult{}, err
	}
	set := hubSet{}
	for _, h := range hubs {
		set[h.ID] = true
	}

	details := make([]ChangedSymbolDetail, 0, len(changes))
	for _, ch := range changes {
		if ch.Kind == "file" {
			details = append(details, ChangedSymbolDetail{
				ID: ch.ID, Kind: "file", ChangeType: ch.ChangeType,
				IsNew:     ch.IsNew,
				Community: communityFromPath(ch.ID),
			})
			continue
		}
		d, err := enrichChangedSymbol(r, ch, set, in.CallerSample)
		if err != nil {
			return PreflightResult{}, err
		}
		details = append(details, d)
	}

	// Rank by risk descending, then by id for determinism.
	sort.SliceStable(details, func(i, j int) bool {
		if details[i].Risk != details[j].Risk {
			return details[i].Risk > details[j].Risk
		}
		return details[i].ID < details[j].ID
	})

	var truncated []string
	if len(details) > in.SymbolBudget {
		for _, d := range details[in.SymbolBudget:] {
			truncated = append(truncated, d.ID)
		}
		details = details[:in.SymbolBudget]
	}

	cross := crossCommunityEdges(r, details)
	summary := buildRiskSummary(details, cross)

	return PreflightResult{
		Mode:             "built",
		Graph:            GraphMeta{Freshness: Freshness{CoversBaseSHA: true}, Languages: detectLanguages(r), SkippedFiles: nil},
		ChangedSymbols:   details,
		CrossCommunity:   cross,
		RiskSummary:      summary,
		TruncatedSymbols: truncated,
	}, nil
}

func crossCommunityEdges(r Reader, details []ChangedSymbolDetail) []CrossCommunityEdge {
	type key struct{ from, to string }
	agg := map[key]*CrossCommunityEdge{}
	for _, d := range details {
		dstNode, err := r.GetNode(d.ID)
		if err != nil {
			continue
		}
		toCom := communityFromPath(dstNode.Path)
		for _, callerID := range d.Callers.Sample {
			n, err := r.GetNode(callerID)
			if err != nil {
				continue
			}
			fromCom := communityFromPath(n.Path)
			if fromCom == "" || fromCom == toCom {
				continue
			}
			k := key{fromCom, toCom}
			e, ok := agg[k]
			if !ok {
				e = &CrossCommunityEdge{From: fromCom, To: toCom}
				agg[k] = e
			}
			e.CountAdded++
			if len(e.Samples) < 5 {
				e.Samples = append(e.Samples, callerID+" → "+d.ID)
			}
		}
	}
	out := make([]CrossCommunityEdge, 0, len(agg))
	for _, v := range agg {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}

func buildRiskSummary(details []ChangedSymbolDetail, cross []CrossCommunityEdge) RiskSummary {
	var s RiskSummary
	for _, d := range details {
		if d.Callers.InHub {
			s.HubNodesModified++
		}
		if d.IsExported && !d.Tests.HasTests {
			s.UntestedPublicChanges++
		}
		if containsString(d.RiskFactors, "interface_change") {
			s.InterfaceChanges++
		}
	}
	for _, c := range cross {
		s.NewCrossCommunityEdges += c.CountAdded
	}
	return s
}

func detectLanguages(r Reader) []string {
	all, err := r.AllNodes()
	if err != nil {
		return nil
	}
	set := map[string]struct{}{}
	for _, n := range all {
		if n.Language != "" {
			set[n.Language] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for l := range set {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}
```

- [ ] **Step 4: Run, expect PASS.**

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(graph/query): Preflight composite with ranking, truncation, cross-community"
```

---

## Task 3.11: Round-trip smoke test against a real fixture

**Files:**
- Create: `internal/graph/query/preflight_roundtrip_test.go`

This is the Phase 3 acceptance test specified in the master plan: build a small graph (via `parser` + `store.InsertNodes`/`InsertEdges`), call `Preflight` with stubbed git, and assert the produced JSON shape matches §6.

- [ ] **Step 1: Write the test**

```go
package query

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestPreflightShapeMatchesSpec(t *testing.T) {
	// Parse a single Go file from the existing parser testdata.
	p := parser.NewGoParser()
	src := []byte(`package x
func Foo() {}
func Bar() { Foo() }
`)
	r, err := p.Parse("x/x.go", src)
	if err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(t.TempDir() + "/graph.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.InsertNodes(r.Nodes); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertEdges(r.Edges); err != nil {
		t.Fatal(err)
	}

	prevGitRun := gitRun
	t.Cleanup(func() { gitRun = prevGitRun })
	gitRun = func(repo string, args ...string) ([]byte, error) {
		switch args[0] {
		case "diff":
			return []byte("M\tx/x.go\n"), nil
		case "show":
			if strings.HasSuffix(args[1], "BASE:x/x.go") {
				return []byte("old"), nil
			}
			return []byte("new"), nil
		}
		return nil, nil
	}

	res, err := Preflight(st, PreflightInput{RepoRoot: "/fake", Base: "BASE", Head: "HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"mode"`, `"changed_symbols"`, `"risk_summary"`,
		`"cross_community_edges"`, `"truncated_symbols"`,
		`"callers"`, `"tests"`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing field %s in marshalled payload:\n%s", want, string(b))
		}
	}
}
```

- [ ] **Step 2: Run, expect PASS** (all dependencies are already in place).

```bash
go test ./internal/graph/query/ -run TestPreflightShapeMatchesSpec -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/graph/query/preflight_roundtrip_test.go
git commit -m "test(graph/query): preflight round-trip shape matches design §6"
```

---

## Phase 3 acceptance

- [ ] `go test ./internal/graph/query/...` is green.
- [ ] `go test ./internal/graph/store/...` is green (with the two added accessor sets).
- [ ] `make lint` clean.
- [ ] Every query in §6 of the design doc has at least one passing test in `internal/graph/query/`:
  - `CallersOf` → `callers_test.go`
  - `CalleesOf` → `callees_test.go`
  - `TestsFor` → `tests_test.go`
  - `ImpactRadius` → `impact_test.go`
  - `Hubs` → `hubs_test.go`
  - `ImplementorsOf` → `implementors_test.go`
  - `Context` → `context_test.go`
  - `DetectChanges` → `detect_changes_test.go`
  - `Preflight` → `preflight_test.go` + `preflight_roundtrip_test.go`
  - `RiskScore` → `risk_test.go`
- [ ] `Preflight` output JSON contains every field in §6 of the design doc (verified by the round-trip test).
- [ ] No query writes to the store (verified by code inspection; the `Reader` interface exposes only read methods).

---

## Cross-phase notes

**Phase 4 handoff.** Each `query.<Verb>` function returns a typed Go struct whose JSON tags match the §6 schema. Phase 4 CLI commands thread their flags into the corresponding query function and wrap the result with `envelope.OK(result)`. No reshaping should be needed at the CLI layer.

**LSP cross-check (Phase 5).** All BFS-style queries (`CallersOf`, `CalleesOf`, `TestsFor`, `ImplementorsOf`) are the targets of LSP cross-check. Keep their public signatures stable; the cross-check harness binds against the names defined in this phase.

**Skill consumption (Phase 6).** The skill only consumes the `Preflight` JSON. Any future tweak to `PreflightResult` field shape must bump `version` in the envelope; Phase 6 contains a contract test that locks the v1 shape.

---

## Self-review

- **Spec coverage:** §6 schema fields enumerated above each map to a `ChangedSymbolDetail`/`RiskSummary`/`CrossCommunityEdge` field. The four risk factors (`is_exported`, `in_hub`, `interface_change`, `untested`) are encoded in `RiskScore` exactly per §6 weights.
- **Placeholder scan:** no TBDs, no "implement later", no naked "add tests" — every step has a code block or an exact command.
- **Type consistency:** `Caller`, `Callee`, `Hub`, `Snippet`, `ChangedSymbol`, `ChangedSymbolDetail`, `CallerSummary`, `TestSummary`, `CrossCommunityEdge`, `RiskSummary`, `RiskInputs`, `Preflight*` — every name introduced in one task is reused verbatim in later tasks.
- **No cycles:** `query` imports `store`; nothing in `store` imports `query`. `Preflight` calls `DetectChanges`, `Hubs`, `TestsFor`, `ImplementorsOf` — all same package.
- **Test isolation:** `detect_changes_test.go` and `preflight_test.go` are the only tests that touch `gitRun`; both restore the previous value via `t.Cleanup`.
