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
