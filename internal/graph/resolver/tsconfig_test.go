package resolver

import (
	"path/filepath"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestTSConfigResolverRewritesAliasImports(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "parser", "testdata", "ts", "alias"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewTSConfigResolver(root)
	if err != nil {
		t.Fatal(err)
	}
	edges := []store.Edge{
		{Src: "src/a.ts", Dst: "external::@lib/b", Kind: "imports"},
	}
	got := r.Rewrite(edges)
	if len(got) != 1 {
		t.Fatalf("want 1 edge, got %d", len(got))
	}
	if got[0].Dst != "src/lib/b.ts" {
		t.Errorf("dst=%q want src/lib/b.ts", got[0].Dst)
	}
}
