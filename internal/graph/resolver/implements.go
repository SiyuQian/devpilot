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
	for _, r := range results {
		for _, n := range r.Nodes {
			if n.Kind == "method" && n.Container != "" {
				// Match to a struct whose ID ends with "::<Container>".
				suffix := "::" + n.Container
				for _, si := range structs {
					if endsWith(si.id, suffix) {
						si.methods[n.Name] = struct{}{}
					}
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

// endsWith reports whether s ends with suffix.
func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
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
