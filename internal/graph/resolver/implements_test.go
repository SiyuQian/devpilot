package resolver

import (
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

// hasImplementsEdge reports whether any result in the batch contains an
// implements edge from src to dst.
func hasImplementsEdge(results []parser.ParseResult, src, dst string) bool {
	for _, r := range results {
		for _, e := range r.Edges {
			if e.Kind == "implements" && e.Src == src && e.Dst == dst {
				return true
			}
		}
	}
	return false
}

// countImplementsEdges counts every implements edge across the batch.
func countImplementsEdges(results []parser.ParseResult) int {
	n := 0
	for _, r := range results {
		for _, e := range r.Edges {
			if e.Kind == "implements" {
				n++
			}
		}
	}
	return n
}

func TestAddImplementsEdges(t *testing.T) {
	tests := []struct {
		name  string
		input []parser.ParseResult
		check func(t *testing.T, out []parser.ParseResult)
	}{
		{
			name: "exact_method_set_emits_edge",
			input: []parser.ParseResult{{
				Nodes: []store.Node{
					{ID: "a.go::Greeter", Name: "Greeter", Kind: "interface", Path: "a.go"},
					{ID: "a.go::Console", Name: "Console", Kind: "struct", Path: "a.go"},
					{ID: "a.go::Console.Greet", Name: "Greet", Kind: "method", Path: "a.go", Container: "Console"},
				},
				InterfaceMethods: map[string][]string{
					"a.go::Greeter": {"Greet"},
				},
			}},
			check: func(t *testing.T, out []parser.ParseResult) {
				if !hasImplementsEdge(out, "a.go::Console", "a.go::Greeter") {
					t.Fatalf("expected Console -> Greeter implements edge")
				}
				if countImplementsEdges(out) != 1 {
					t.Errorf("expected exactly 1 implements edge, got %d", countImplementsEdges(out))
				}
			},
		},
		{
			name: "missing_method_no_edge",
			input: []parser.ParseResult{{
				Nodes: []store.Node{
					{ID: "a.go::ReadWriter", Name: "ReadWriter", Kind: "interface", Path: "a.go"},
					{ID: "a.go::OnlyReader", Name: "OnlyReader", Kind: "struct", Path: "a.go"},
					{ID: "a.go::OnlyReader.Read", Name: "Read", Kind: "method", Path: "a.go", Container: "OnlyReader"},
				},
				InterfaceMethods: map[string][]string{
					"a.go::ReadWriter": {"Read", "Write"},
				},
			}},
			check: func(t *testing.T, out []parser.ParseResult) {
				if hasImplementsEdge(out, "a.go::OnlyReader", "a.go::ReadWriter") {
					t.Fatalf("OnlyReader is missing Write; must not implement ReadWriter")
				}
				if countImplementsEdges(out) != 0 {
					t.Errorf("expected no implements edges, got %d", countImplementsEdges(out))
				}
			},
		},
		{
			name: "superset_emits_edge",
			input: []parser.ParseResult{{
				Nodes: []store.Node{
					{ID: "a.go::Reader", Name: "Reader", Kind: "interface", Path: "a.go"},
					{ID: "a.go::Fancy", Name: "Fancy", Kind: "struct", Path: "a.go"},
					{ID: "a.go::Fancy.Read", Name: "Read", Kind: "method", Path: "a.go", Container: "Fancy"},
					{ID: "a.go::Fancy.Close", Name: "Close", Kind: "method", Path: "a.go", Container: "Fancy"},
					{ID: "a.go::Fancy.Reset", Name: "Reset", Kind: "method", Path: "a.go", Container: "Fancy"},
				},
				InterfaceMethods: map[string][]string{
					"a.go::Reader": {"Read"},
				},
			}},
			check: func(t *testing.T, out []parser.ParseResult) {
				if !hasImplementsEdge(out, "a.go::Fancy", "a.go::Reader") {
					t.Fatalf("Fancy has superset of Reader's methods; expected implements edge")
				}
			},
		},
		{
			name: "empty_interface_matches_nothing",
			input: []parser.ParseResult{{
				Nodes: []store.Node{
					{ID: "a.go::Any", Name: "Any", Kind: "interface", Path: "a.go"},
					{ID: "a.go::S1", Name: "S1", Kind: "struct", Path: "a.go"},
					{ID: "a.go::S2", Name: "S2", Kind: "struct", Path: "a.go"},
					{ID: "a.go::S1.Foo", Name: "Foo", Kind: "method", Path: "a.go", Container: "S1"},
				},
				InterfaceMethods: map[string][]string{
					"a.go::Any": {}, // interface{}
				},
			}},
			check: func(t *testing.T, out []parser.ParseResult) {
				if countImplementsEdges(out) != 0 {
					t.Fatalf("interface{} must not match-everything; expected 0 implements edges, got %d", countImplementsEdges(out))
				}
			},
		},
		{
			name: "unrelated_struct_with_same_method_name_only_matches_interface_subset",
			input: []parser.ParseResult{{
				// Two structs with method "Run". Only structs whose method
				// set is a superset of the interface's declared method set
				// should get the edge — same-name across unrelated structs
				// is the intended match (per superset semantics on names),
				// but the edge must point to the right struct ID, not
				// silently broaden to others.
				Nodes: []store.Node{
					{ID: "a.go::Runner", Name: "Runner", Kind: "interface", Path: "a.go"},
					{ID: "a.go::Job", Name: "Job", Kind: "struct", Path: "a.go"},
					{ID: "a.go::Job.Run", Name: "Run", Kind: "method", Path: "a.go", Container: "Job"},
					{ID: "a.go::Task", Name: "Task", Kind: "struct", Path: "a.go"},
					// Task has no methods.
				},
				InterfaceMethods: map[string][]string{
					"a.go::Runner": {"Run"},
				},
			}},
			check: func(t *testing.T, out []parser.ParseResult) {
				if !hasImplementsEdge(out, "a.go::Job", "a.go::Runner") {
					t.Errorf("Job should implement Runner")
				}
				if hasImplementsEdge(out, "a.go::Task", "a.go::Runner") {
					t.Errorf("Task has no methods; must not implement Runner")
				}
			},
		},
		{
			name: "cross_file_methods_within_package_not_attributed_v1_limitation",
			// v1 limitation locked in: methods declared in a separate file
			// from their struct are not attributed because attribution
			// matches on `<path>::<container>` exactly. Document this so a
			// future fix to relax the rule will fail this test loudly.
			input: []parser.ParseResult{
				{
					Nodes: []store.Node{
						{ID: "a.go::Greeter", Name: "Greeter", Kind: "interface", Path: "a.go"},
						{ID: "a.go::Console", Name: "Console", Kind: "struct", Path: "a.go"},
					},
					InterfaceMethods: map[string][]string{
						"a.go::Greeter": {"Greet"},
					},
				},
				{
					Nodes: []store.Node{
						// Method declared in b.go, but the struct lives in a.go.
						// Current resolver requires `<path>::<container>` to match,
						// so this method is NOT attached to a.go::Console.
						{ID: "b.go::Console.Greet", Name: "Greet", Kind: "method", Path: "b.go", Container: "Console"},
					},
				},
			},
			check: func(t *testing.T, out []parser.ParseResult) {
				if hasImplementsEdge(out, "a.go::Console", "a.go::Greeter") {
					t.Fatalf("v1 limitation regressed: cross-file method should NOT yet be attributed to a.go::Console")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := addImplementsEdges(tc.input)
			tc.check(t, out)
		})
	}
}

func TestIsSuperset(t *testing.T) {
	mk := func(keys ...string) map[string]struct{} {
		m := map[string]struct{}{}
		for _, k := range keys {
			m[k] = struct{}{}
		}
		return m
	}

	tests := []struct {
		name  string
		super map[string]struct{}
		sub   map[string]struct{}
		want  bool
	}{
		{name: "equal_sets", super: mk("a", "b"), sub: mk("a", "b"), want: true},
		{name: "strict_superset", super: mk("a", "b", "c"), sub: mk("a", "b"), want: true},
		{name: "missing_key", super: mk("a"), sub: mk("a", "b"), want: false},
		{name: "empty_sub_trivially_true", super: mk("a"), sub: mk(), want: true},
		{name: "both_empty", super: mk(), sub: mk(), want: true},
		{name: "disjoint", super: mk("a"), sub: mk("b"), want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isSuperset(tc.super, tc.sub)
			if got != tc.want {
				t.Errorf("isSuperset(%v, %v) = %v, want %v", tc.super, tc.sub, got, tc.want)
			}
		})
	}
}
