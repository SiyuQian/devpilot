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
		{
			name: "no_op_when_no_externals",
			setup: func(t *testing.T) []parser.ParseResult {
				// Construct a simple batch with no external:: edges and no InterfaceMethods.
				// The resolver should return this input unchanged (fast path).
				return []parser.ParseResult{
					{
						Nodes: []store.Node{
							{ID: "test.go::Foo", Name: "Foo", Kind: "function", Path: "test.go"},
							{ID: "test.go::Bar", Name: "Bar", Kind: "function", Path: "test.go"},
						},
						Edges: []store.Edge{
							{Src: "test.go::Foo", Dst: "test.go::Bar", Kind: "calls"},
						},
						Errors:           nil,
						InterfaceMethods: map[string][]string{},
					},
				}
			},
			check: func(t *testing.T, resolved []parser.ParseResult) {
				// Verify the result is byte-identical and of expected length.
				if len(resolved) != 1 {
					t.Errorf("expected 1 result, got %d", len(resolved))
				}
				if len(resolved[0].Nodes) != 2 {
					t.Errorf("expected 2 nodes, got %d", len(resolved[0].Nodes))
				}
				if len(resolved[0].Edges) != 1 {
					t.Errorf("expected 1 edge, got %d", len(resolved[0].Edges))
				}
				// Check that the edge is unchanged.
				if resolved[0].Edges[0].Src != "test.go::Foo" || resolved[0].Edges[0].Dst != "test.go::Bar" {
					t.Errorf("edge was modified: got %v -> %v", resolved[0].Edges[0].Src, resolved[0].Edges[0].Dst)
				}
			},
		},
		{
			name: "rewrite_external_edge_when_present",
			setup: func(t *testing.T) []parser.ParseResult {
				// Construct a batch with an external:: edge that can be resolved.
				// First result contains a call to external::Bar.
				// Second result defines Bar function.
				// Resolver should rewrite the external::Bar edge to point to the real Bar ID.
				return []parser.ParseResult{
					{
						Nodes: []store.Node{
							{ID: "a.go::Foo", Name: "Foo", Kind: "function", Path: "a.go"},
						},
						Edges: []store.Edge{
							{Src: "a.go::Foo", Dst: "external::Bar", Kind: "calls"},
						},
						Errors:           nil,
						InterfaceMethods: map[string][]string{},
					},
					{
						Nodes: []store.Node{
							{ID: "b.go::Bar", Name: "Bar", Kind: "function", Path: "b.go"},
						},
						Edges:            []store.Edge{},
						Errors:           nil,
						InterfaceMethods: map[string][]string{},
					},
				}
			},
			check: func(t *testing.T, resolved []parser.ParseResult) {
				// Verify the external::Bar edge was rewritten to b.go::Bar.
				if len(resolved) != 2 {
					t.Errorf("expected 2 results, got %d", len(resolved))
				}
				if len(resolved[0].Edges) != 1 {
					t.Errorf("expected 1 edge in first result, got %d", len(resolved[0].Edges))
				}
				edge := resolved[0].Edges[0]
				if edge.Dst != "b.go::Bar" {
					t.Errorf("expected external::Bar rewritten to b.go::Bar, got %s", edge.Dst)
				}
				if edge.Src != "a.go::Foo" || edge.Kind != "calls" {
					t.Errorf("edge src/kind were modified: src=%s kind=%s", edge.Src, edge.Kind)
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
