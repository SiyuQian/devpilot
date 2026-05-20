package query

import "fmt"

// Caller is a node that (possibly transitively) calls the queried target.
// Hop is the BFS distance from the target (1 = direct caller).
type Caller struct {
	ID  string
	Hop int
}

// CallersOf returns all transitive callers of id up to maxDepth hops via
// `calls` edges, in BFS order. The target itself is never returned.
func CallersOf(r Reader, id string, maxDepth int) ([]Caller, error) {
	if maxDepth < 1 {
		return nil, nil
	}
	seen := map[string]bool{id: true}
	frontier := []string{id}
	var out []Caller
	for hop := 1; hop <= maxDepth && len(frontier) > 0; hop++ {
		var next []string
		for _, cur := range frontier {
			edges, err := r.EdgesByDst(cur, "calls")
			if err != nil {
				return nil, fmt.Errorf("CallersOf at hop %d: %w", hop, err)
			}
			for _, e := range edges {
				if seen[e.Src] {
					continue
				}
				seen[e.Src] = true
				out = append(out, Caller{ID: e.Src, Hop: hop})
				next = append(next, e.Src)
			}
		}
		frontier = next
	}
	return out, nil
}
