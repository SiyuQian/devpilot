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
