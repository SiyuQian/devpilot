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
