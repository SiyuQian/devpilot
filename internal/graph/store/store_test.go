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

func TestStoreSchema(t *testing.T) {
	tests := []struct {
		name  string
		table string
	}{
		{"nodes_table_present", "nodes"},
		{"edges_table_present", "edges"},
		{"schema_version_table_present", "schema_version"},
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "graph.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var exists int
			err := s.db.QueryRow(
				`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, tc.table,
			).Scan(&exists)
			if err != nil || exists != 1 {
				t.Errorf("table %s missing", tc.table)
			}
		})
	}
}

func TestStoreNodeRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		node      Node
		wantName  string
		wantExp   bool
		wantCont  string
	}{
		{
			name: "method_with_container",
			node: Node{
				ID: "internal/foo.go::Foo.Bar", Kind: "method", Path: "internal/foo.go",
				Name: "Bar", Container: "Foo", Language: "go", IsExported: true,
			},
			wantName: "Bar",
			wantExp:  true,
			wantCont: "Foo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestStore(t)
			if err := s.InsertNodes([]Node{tc.node}); err != nil {
				t.Fatalf("InsertNodes: %v", err)
			}
			got, err := s.GetNode(tc.node.ID)
			if err != nil {
				t.Fatalf("GetNode: %v", err)
			}
			if got.Name != tc.wantName {
				t.Errorf("Name=%q, want %q", got.Name, tc.wantName)
			}
			if got.IsExported != tc.wantExp {
				t.Errorf("IsExported=%v, want %v", got.IsExported, tc.wantExp)
			}
			if got.Container != tc.wantCont {
				t.Errorf("Container=%q, want %q", got.Container, tc.wantCont)
			}
		})
	}
}

func TestStoreCallersOf(t *testing.T) {
	tests := []struct {
		name        string
		nodes       []Node
		edges       []Edge
		targetID    string
		wantCallers int
	}{
		{
			name: "two_callers",
			nodes: []Node{
				{ID: "a.go::A", Kind: "function", Path: "a.go", Name: "A", Language: "go"},
				{ID: "b.go::B", Kind: "function", Path: "b.go", Name: "B", Language: "go"},
				{ID: "c.go::C", Kind: "function", Path: "c.go", Name: "C", Language: "go"},
			},
			edges: []Edge{
				{Src: "a.go::A", Dst: "c.go::C", Kind: "calls"},
				{Src: "b.go::B", Dst: "c.go::C", Kind: "calls"},
			},
			targetID:    "c.go::C",
			wantCallers: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestStore(t)
			must := func(err error) { t.Helper(); if err != nil { t.Fatal(err) } }
			must(s.InsertNodes(tc.nodes))
			must(s.InsertEdges(tc.edges))
			callers, err := s.CallersOf(tc.targetID)
			must(err)
			if len(callers) != tc.wantCallers {
				t.Fatalf("expected %d callers, got %d: %v", tc.wantCallers, len(callers), callers)
			}
		})
	}
}
