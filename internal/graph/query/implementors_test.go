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
