package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SweepPreflight deletes files under <home>/preflight/ whose modtime is older
// than ttl. Missing directory is not an error.
func SweepPreflight(home string, ttl time.Duration) error {
	dir := filepath.Join(home, "preflight")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", dir, err)
	}
	cutoff := time.Now().Add(-ttl)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
	return nil
}
