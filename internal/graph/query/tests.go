package query

import "fmt"

// TestsFor returns the IDs of test symbols that exercise id (i.e., nodes
// connected to id by a `tests` edge). Order is insertion order from SQLite.
func TestsFor(r Reader, id string) ([]string, error) {
	edges, err := r.EdgesByDst(id, "tests")
	if err != nil {
		return nil, fmt.Errorf("TestsFor: %w", err)
	}
	out := make([]string, 0, len(edges))
	for _, e := range edges {
		out = append(out, e.Src)
	}
	return out, nil
}
