package query

import "fmt"

// ImplementorsOf returns the IDs of types that implement the given interface
// (i.e., nodes connected to ifaceID by an `implements` edge).
func ImplementorsOf(r Reader, ifaceID string) ([]string, error) {
	edges, err := r.EdgesByDst(ifaceID, "implements")
	if err != nil {
		return nil, fmt.Errorf("ImplementorsOf: %w", err)
	}
	out := make([]string, 0, len(edges))
	for _, e := range edges {
		out = append(out, e.Src)
	}
	return out, nil
}
