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
