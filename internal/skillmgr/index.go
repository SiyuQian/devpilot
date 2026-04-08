package skillmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const rawBaseURL = "https://raw.githubusercontent.com"

// IndexEntry represents a single skill in the catalog index.
type IndexEntry struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
}

type indexFile struct {
	Skills []IndexEntry `json:"skills"`
}

// ParseIndex parses a skills/index.json payload into a slice of IndexEntry.
func ParseIndex(data []byte) ([]IndexEntry, error) {
	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing index.json: %w", err)
	}
	var valid []IndexEntry
	for _, e := range idx.Skills {
		if e.Name != "" {
			valid = append(valid, e)
		}
	}
	return valid, nil
}

// FetchIndex downloads and parses skills/index.json from raw.githubusercontent.com.
func FetchIndex(ctx context.Context, owner, repo, ref string) ([]IndexEntry, error) {
	return fetchIndexFromBase(ctx, rawBaseURL, owner, repo, ref)
}

// maxIndexSize is the maximum allowed size for index.json (1 MB).
const maxIndexSize = 1 << 20

func fetchIndexFromBase(ctx context.Context, base, owner, repo, ref string) ([]IndexEntry, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/skills/index.json", base, owner, repo, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for index.json: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching index.json: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching index.json: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxIndexSize))
	if err != nil {
		return nil, fmt.Errorf("reading index.json body: %w", err)
	}
	return ParseIndex(data)
}
