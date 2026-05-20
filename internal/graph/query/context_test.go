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
