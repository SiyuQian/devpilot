// Package cache manages the on-disk graph cache under ~/.devpilot/graphs/.
package cache

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RepoKey is a deterministic 12-character hex identifier derived from the
// absolute path of a repository root. Different clones of the same repo
// intentionally produce different keys (see design doc §5).
func RepoKey(absRepoRoot string) string {
	sum := sha1.Sum([]byte(absRepoRoot))
	return hex.EncodeToString(sum[:])[:12]
}

// Home returns the devpilot cache root, defaulting to ~/.devpilot.
// Overridable via DEVPILOT_HOME.
func Home() string {
	if v := os.Getenv("DEVPILOT_HOME"); v != "" {
		return v
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".devpilot")
	}
	return ".devpilot"
}

// GraphDir returns the per-repo cache directory.
func GraphDir(home, key string) string {
	return filepath.Join(home, "graphs", key)
}

// GraphDB returns the SQLite file path.
func GraphDB(home, key string) string {
	return filepath.Join(GraphDir(home, key), "graph.db")
}

// MetaFile returns the meta.json path.
func MetaFile(home, key string) string {
	return filepath.Join(GraphDir(home, key), "meta.json")
}

// LockFile returns the build.lock path.
func LockFile(home, key string) string {
	return filepath.Join(GraphDir(home, key), "build.lock")
}

// PreflightFile returns a timestamped preflight output path.
func PreflightFile(home, key string) string {
	return filepath.Join(home, "preflight",
		fmt.Sprintf("%s-%d.json", key, time.Now().UnixNano()))
}

// EnsureDirs mkdir -p's the graphs and preflight directories.
func EnsureDirs(home, key string) error {
	for _, d := range []string{GraphDir(home, key), filepath.Join(home, "preflight")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return nil
}
