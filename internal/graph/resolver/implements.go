package resolver

import (
	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

// addImplementsEdges emits "implements" edges where a struct's set of methods
// (by name) is a superset of an interface's declared method set. Uses method
// names only; signature matching is deferred to a later task.
//
// An interface with no recorded methods (e.g. interface{}) is skipped to
// avoid emitting an edge for every concrete type.
func addImplementsEdges(results []parser.ParseResult) []parser.ParseResult {
	type structInfo struct {
		id      string
		methods map[string]struct{}
		owner   int // index into results
	}

	// 1. Collect struct nodes and their method sets.
	structs := map[string]*structInfo{} // structNodeID -> info
	for i, r := range results {
		for _, n := range r.Nodes {
			if n.Kind == "struct" {
				if _, ok := structs[n.ID]; !ok {
					structs[n.ID] = &structInfo{
						id:      n.ID,
						methods: map[string]struct{}{},
						owner:   i,
					}
				}
			}
		}
	}

	// 2. Populate method sets: walk method nodes and match to their struct.
	// We require an exact match on the fully-qualified struct ID
	// (`<path>::<container>`) so methods only attach to the struct in the
	// same file. v1 limitation: Go allows methods on a type in any file of
	// the same package; cross-file methods within a package are not yet
	// attributed. This is conservative — false negatives, never false
	// positives. Cross-file attribution will be revisited when the package
	// resolver lands in a later phase.
	for _, r := range results {
		for _, n := range r.Nodes {
			if n.Kind == "method" && n.Container != "" {
				want := n.Path + "::" + n.Container
				if si, ok := structs[want]; ok {
					si.methods[n.Name] = struct{}{}
				}
			}
		}
	}

	// 3. Emit implements edges.
	for _, r := range results {
		for ifaceID, methodNames := range r.InterfaceMethods {
			if len(methodNames) == 0 {
				continue
			}
			ifaceSet := make(map[string]struct{}, len(methodNames))
			for _, m := range methodNames {
				ifaceSet[m] = struct{}{}
			}
			for _, si := range structs {
				if isSuperset(si.methods, ifaceSet) {
					results[si.owner].Edges = append(results[si.owner].Edges, store.Edge{
						Src:  si.id,
						Dst:  ifaceID,
						Kind: "implements",
					})
				}
			}
		}
	}

	return results
}

// isSuperset reports whether super contains every key in sub.
func isSuperset(super, sub map[string]struct{}) bool {
	for k := range sub {
		if _, ok := super[k]; !ok {
			return false
		}
	}
	return true
}
