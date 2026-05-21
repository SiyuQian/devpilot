// Package resolver rewrites synthetic external:: edges produced by the parser
// into real intra-module node IDs when the symbol can be located in another
// file of the same parse batch.
package resolver

import (
	"strings"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

// Resolve takes ParseResults from a single Go module and rewrites edges whose
// dst is "external::<Name>" into "<path>::<Name>" when a node with that name
// exists in the batch. Edges of the form "external::pkg.Sym" are left alone
// (those are true external references resolved via go.mod, out of scope for v1).
//
// Fast path: if there are no external:: edges to rewrite and no InterfaceMethods
// to process, returns the input slice unchanged (no allocation).
func Resolve(results []parser.ParseResult) []parser.ParseResult {
	// Fast path: if no external:: edges and no interface methods, return as-is.
	if !needsResolve(results) {
		return results
	}

	// Build name -> nodeID lookup from all top-level nodes (functions, types).
	// We index by simple name; a duplicate name across files keeps the first
	// occurrence — v1 limitation, called out here.
	nameIndex := map[string]string{}
	for _, r := range results {
		for _, n := range r.Nodes {
			if n.Kind == "function" || n.Kind == "struct" || n.Kind == "interface" || n.Kind == "type" {
				if _, exists := nameIndex[n.Name]; !exists {
					nameIndex[n.Name] = n.ID
				}
			}
		}
	}

	out := make([]parser.ParseResult, len(results))
	for i, r := range results {
		newEdges := make([]store.Edge, 0, len(r.Edges))
		for _, e := range r.Edges {
			if rewritten := rewriteExternalEdge(e, nameIndex); rewritten.Dst != "" {
				newEdges = append(newEdges, rewritten)
			} else {
				newEdges = append(newEdges, e)
			}
		}
		out[i] = parser.ParseResult{
			Nodes:            r.Nodes,
			Edges:            newEdges,
			Errors:           r.Errors,
			InterfaceMethods: r.InterfaceMethods,
		}
	}
	return addImplementsEdges(out)
}

// needsResolve returns true if the results contain any external:: edges to
// rewrite or any InterfaceMethods to process (which addImplementsEdges handles).
// When false, Resolve can skip all processing and return the input unchanged.
func needsResolve(results []parser.ParseResult) bool {
	for _, r := range results {
		// Check for external:: edges.
		for _, e := range r.Edges {
			if strings.HasPrefix(e.Dst, "external::") {
				return true
			}
		}
		// Check for interface methods that need implements edge synthesis.
		if len(r.InterfaceMethods) > 0 {
			return true
		}
	}
	return false
}

// rewriteExternalEdge inspects an edge and, if its dst is a synthetic
// external::<Name> reference for a name we know intra-batch, returns a new edge
// with the resolved dst. Otherwise returns the edge unchanged.
func rewriteExternalEdge(e store.Edge, idx map[string]string) store.Edge {
	const prefix = "external::"
	if !strings.HasPrefix(e.Dst, prefix) {
		return e
	}
	name := e.Dst[len(prefix):]
	if strings.Contains(name, ".") {
		// pkg.Sym — true external for now
		return e
	}
	if id, ok := idx[name]; ok {
		return store.Edge{Src: e.Src, Dst: id, Kind: e.Kind}
	}
	return e
}
