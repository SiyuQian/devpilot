package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) []parser.ParseResult
		check func(t *testing.T, resolved []parser.ParseResult)
	}{
		{
			name: "intra_module_calls",
			setup: func(t *testing.T) []parser.ParseResult {
				p := parser.NewGoParser()
				dir := filepath.Join("..", "parser", "testdata", "go", "multifile")
				aSrc, err := os.ReadFile(filepath.Join(dir, "a.go"))
				if err != nil {
					t.Fatal(err)
				}
				bSrc, err := os.ReadFile(filepath.Join(dir, "b.go"))
				if err != nil {
					t.Fatal(err)
				}
				rA, err := p.Parse("multifile/a.go", aSrc)
				if err != nil {
					t.Fatal(err)
				}
				rB, err := p.Parse("multifile/b.go", bSrc)
				if err != nil {
					t.Fatal(err)
				}
				_ = store.Node{} // keep store import used
				return []parser.ParseResult{rA, rB}
			},
			check: func(t *testing.T, resolved []parser.ParseResult) {
				var foundCall bool
				for _, r := range resolved {
					for _, e := range r.Edges {
						if e.Kind == "calls" && e.Src == "multifile/a.go::A" && e.Dst == "multifile/b.go::B" {
							foundCall = true
						}
					}
				}
				if !foundCall {
					var seen []string
					for _, r := range resolved {
						for _, e := range r.Edges {
							if e.Kind == "calls" && e.Src == "multifile/a.go::A" {
								seen = append(seen, e.Dst)
							}
						}
					}
					t.Fatalf("expected calls edge multifile/a.go::A -> multifile/b.go::B, saw dsts: %v", seen)
				}

				var foundFmt bool
				for _, r := range resolved {
					for _, e := range r.Edges {
						if e.Kind == "calls" && e.Src == "multifile/a.go::A" && e.Dst == "external::fmt.Println" {
							foundFmt = true
						}
					}
				}
				if !foundFmt {
					t.Errorf("external::fmt.Println edge was wrongly rewritten")
				}

				if len(resolved) != 2 {
					t.Errorf("expected 2 ParseResults out, got %d", len(resolved))
				}
			},
		},
		{
			name: "implements_edges",
			setup: func(t *testing.T) []parser.ParseResult {
				p := parser.NewGoParser()
				dir := filepath.Join("..", "parser", "testdata", "go", "iface")
				src, err := os.ReadFile(filepath.Join(dir, "iface.go"))
				if err != nil {
					t.Fatal(err)
				}
				r, err := p.Parse("iface/iface.go", src)
				if err != nil {
					t.Fatal(err)
				}
				return []parser.ParseResult{r}
			},
			check: func(t *testing.T, resolved []parser.ParseResult) {
				wantSrc := "iface/iface.go::Console"
				wantDst := "iface/iface.go::Greeter"
				var have, haveMute bool
				for _, rr := range resolved {
					for _, e := range rr.Edges {
						if e.Kind == "implements" && e.Src == wantSrc && e.Dst == wantDst {
							have = true
						}
						if e.Kind == "implements" && e.Src == "iface/iface.go::Mute" && e.Dst == wantDst {
							haveMute = true
						}
					}
				}
				if !have {
					t.Errorf("missing implements edge Console -> Greeter")
				}
				if haveMute {
					t.Errorf("Mute should NOT implement Greeter (no methods)")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inputs := tc.setup(t)
			resolved := Resolve(inputs)
			tc.check(t, resolved)
		})
	}
}
