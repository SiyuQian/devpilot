package resolver

import (
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
				if len(resolved) != 1 {
					t.Errorf("expected 1 result, got %d", len(resolved))
				}
				if len(resolved[0].Nodes) != 2 {
					t.Errorf("expected 2 nodes, got %d", len(resolved[0].Nodes))
				}
				if len(resolved[0].Edges) != 1 {
					t.Errorf("expected 1 edge, got %d", len(resolved[0].Edges))
				}
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
		{
			name: "preserves_external_pkg_dotted",
			setup: func(t *testing.T) []parser.ParseResult {
				// `external::pkg.Sym` (with a dot) is a real external symbol —
				// the resolver must NOT try to rewrite it as an intra-module name.
				return []parser.ParseResult{
					{
						Nodes: []store.Node{
							{ID: "a.go::A", Name: "A", Kind: "function", Path: "a.go"},
						},
						Edges: []store.Edge{
							{Src: "a.go::A", Dst: "external::fmt.Println", Kind: "calls"},
						},
					},
				}
			},
			check: func(t *testing.T, resolved []parser.ParseResult) {
				for _, r := range resolved {
					for _, e := range r.Edges {
						if e.Kind == "calls" && e.Src == "a.go::A" && e.Dst != "external::fmt.Println" {
							t.Errorf("external::fmt.Println edge was wrongly rewritten to %q", e.Dst)
						}
					}
				}
			},
		},
		{
			name: "implements_edges_from_interface_methods",
			setup: func(t *testing.T) []parser.ParseResult {
				// Two type nodes plus a method node and an InterfaceMethods
				// declaration. Console implements Greeter (has Greet); Mute does not.
				return []parser.ParseResult{
					{
						Nodes: []store.Node{
							{ID: "iface.go::Greeter", Name: "Greeter", Kind: "interface", Path: "iface.go"},
							{ID: "iface.go::Console", Name: "Console", Kind: "struct", Path: "iface.go"},
							{ID: "iface.go::Mute", Name: "Mute", Kind: "struct", Path: "iface.go"},
							{ID: "iface.go::Console.Greet", Name: "Greet", Kind: "method", Path: "iface.go", Container: "Console"},
						},
						InterfaceMethods: map[string][]string{
							"iface.go::Greeter": {"Greet"},
						},
					},
				}
			},
			check: func(t *testing.T, resolved []parser.ParseResult) {
				wantSrc := "iface.go::Console"
				wantDst := "iface.go::Greeter"
				var have, haveMute bool
				for _, rr := range resolved {
					for _, e := range rr.Edges {
						if e.Kind == "implements" && e.Src == wantSrc && e.Dst == wantDst {
							have = true
						}
						if e.Kind == "implements" && e.Src == "iface.go::Mute" && e.Dst == wantDst {
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
