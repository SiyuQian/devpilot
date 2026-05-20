package query

import "fmt"

// Callee is a node that the queried source (possibly transitively) calls.
type Callee struct {
	ID  string
	Hop int
}

// CalleesOf returns all transitive callees of id up to maxDepth hops via
// `calls` edges. The source itself is excluded.
func CalleesOf(r Reader, id string, maxDepth int) ([]Callee, error) {
	if maxDepth < 1 {
		return nil, nil
	}
	seen := map[string]bool{id: true}
	frontier := []string{id}
	var out []Callee
	for hop := 1; hop <= maxDepth && len(frontier) > 0; hop++ {
		var next []string
		for _, cur := range frontier {
			edges, err := r.EdgesBySrc(cur, "calls")
			if err != nil {
				return nil, fmt.Errorf("CalleesOf at hop %d: %w", hop, err)
			}
			for _, e := range edges {
				if seen[e.Dst] {
					continue
				}
				seen[e.Dst] = true
				out = append(out, Callee{ID: e.Dst, Hop: hop})
				next = append(next, e.Dst)
			}
		}
		frontier = next
	}
	return out, nil
}
