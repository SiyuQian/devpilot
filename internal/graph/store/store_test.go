package store

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir() + "/graph.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

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
	if !got.IsExported {
		t.Errorf("IsExported lost in round-trip")
	}
	if got.Container != "Foo" {
		t.Errorf("Container lost: %q", got.Container)
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
