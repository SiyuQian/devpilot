package query

import "fmt"

// Hub is a high-fanin node — a frequent call target.
type Hub struct {
	ID          string
	CallerCount int
}

// hubReader extends Reader with the hub aggregation that callers/callees do
// not need. It is satisfied by *store.Store.
type hubReader interface {
	Reader
	HubsByCalls(minCallers int) ([]struct {
		ID    string
		Count int
	}, error)
}

// Hubs returns all nodes whose inbound `calls` edge count is >= threshold.
func Hubs(r Reader, threshold int) ([]Hub, error) {
	hr, ok := r.(hubReader)
	if !ok {
		return nil, fmt.Errorf("Hubs: reader does not implement HubsByCalls")
	}
	rows, err := hr.HubsByCalls(threshold)
	if err != nil {
		return nil, fmt.Errorf("Hubs: %w", err)
	}
	out := make([]Hub, 0, len(rows))
	for _, row := range rows {
		out = append(out, Hub{ID: row.ID, CallerCount: row.Count})
	}
	return out, nil
}
