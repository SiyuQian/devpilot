package skillmgr

import "context"

// CatalogEntry describes a skill available from the default source.
type CatalogEntry struct {
	Name        string
	Description string
}

// FetchCatalog discovers available skills by downloading skills/index.json
// from raw.githubusercontent.com and converting entries to CatalogEntry.
func FetchCatalog(ctx context.Context, owner, repo, ref string) ([]CatalogEntry, error) {
	entries, err := FetchIndex(ctx, owner, repo, ref)
	if err != nil {
		return nil, err
	}
	catalog := make([]CatalogEntry, len(entries))
	for i, e := range entries {
		catalog[i] = CatalogEntry{Name: e.Name, Description: e.Description}
	}
	return catalog, nil
}
