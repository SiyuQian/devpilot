package query

import "fmt"

// Impact is the result of an impact-radius query: the symbols owned by the
// changed file set, plus the union of their transitive callers up to depth.
type Impact struct {
	ChangedSymbols []string
	Symbols        []Caller
}

// ImpactRadius returns the union of CallersOf for every symbol contained in
// the given files, up to maxDepth hops.
func ImpactRadius(r Reader, files []string, maxDepth int) (Impact, error) {
	out := Impact{}
	seen := map[string]int{} // id -> min hop
	for _, f := range files {
		nodes, err := r.NodesByPath(f)
		if err != nil {
			return Impact{}, fmt.Errorf("ImpactRadius: %w", err)
		}
		for _, n := range nodes {
			out.ChangedSymbols = append(out.ChangedSymbols, n.ID)
			callers, err := CallersOf(r, n.ID, maxDepth)
			if err != nil {
				return Impact{}, err
			}
			for _, c := range callers {
				if prev, ok := seen[c.ID]; !ok || c.Hop < prev {
					seen[c.ID] = c.Hop
				}
			}
		}
	}
	for id, hop := range seen {
		out.Symbols = append(out.Symbols, Caller{ID: id, Hop: hop})
	}
	return out, nil
}
