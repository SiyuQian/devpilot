package graph

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/siyuqian/devpilot/internal/graph/cache"
	"github.com/siyuqian/devpilot/internal/graph/envelope"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

// resolveRepo normalises a user-provided repo path to an absolute existing dir.
// An empty input falls back to the current working directory.
func resolveRepo(repo string) (string, error) {
	if repo == "" {
		var err error
		repo, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getwd: %w", err)
		}
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return "", fmt.Errorf("abs %s: %w", repo, err)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("repo %s is not a directory", abs)
	}
	return abs, nil
}

// openStore opens the cached graph for the resolved repo, returning an error
// when the cache is absent so callers can map it to a structured envelope code.
func openStore(repoAbs string) (*store.Store, string, error) {
	key := cache.RepoKey(repoAbs)
	dbPath := cache.GraphDB(cache.Home(), key)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, key, fmt.Errorf("graph cache missing for %s — run `devpilot graph build`", repoAbs)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, key, fmt.Errorf("open %s: %w", dbPath, err)
	}
	return st, key, nil
}

// emit writes the envelope as canonical JSON to stdout, validates it against
// schemaID, and returns a process-exit code (0 ok, 1 otherwise).
func emit(e *envelope.Envelope, schemaID string) int {
	b, err := e.Marshal()
	if err != nil {
		fmt.Fprintln(os.Stderr, "envelope marshal:", err)
		return 1
	}
	if err := envelope.Validate(b, schemaID); err != nil {
		fmt.Fprintln(os.Stderr, "envelope schema violation:", err)
		fmt.Fprintln(os.Stderr, string(b))
		return 1
	}
	fmt.Println(string(b))
	if !e.OKFlag {
		return 1
	}
	return 0
}
