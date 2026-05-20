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
