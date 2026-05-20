package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSweepPreflight(t *testing.T) {
	dir := t.TempDir()
	preflightDir := filepath.Join(dir, "preflight")
	if err := os.MkdirAll(preflightDir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(preflightDir, "old.json")
	fresh := filepath.Join(preflightDir, "fresh.json")
	for _, f := range []string{old, fresh} {
		if err := os.WriteFile(f, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	stale := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(old, stale, stale); err != nil {
		t.Fatal(err)
	}

	if err := SweepPreflight(dir, 7*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("old file still exists: %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("fresh file removed: %v", err)
	}
}
