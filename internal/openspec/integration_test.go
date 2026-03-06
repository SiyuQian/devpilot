package openspec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFullSyncFlow(t *testing.T) {
	// Setup: create a mock project with openspec/changes/
	dir := t.TempDir()

	// Create two changes
	for _, name := range []string{"add-auth", "fix-bug"} {
		changeDir := filepath.Join(dir, "openspec", "changes", name)
		os.MkdirAll(changeDir, 0755)
		os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("# "+name+"\nDescription"), 0644)
		os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [ ] Task 1"), 0644)
	}

	// Scan
	changes, err := ScanChanges(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	// Sync to mock target — first time creates
	target := &mockTarget{cards: map[string]string{}}
	results, err := Sync(changes, target)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Action != "created" {
			t.Errorf("expected created, got %s for %s", r.Action, r.Name)
		}
	}

	// Sync again with existing cards — should update
	target2 := &mockTarget{cards: map[string]string{"add-auth": "c1", "fix-bug": "c2"}}
	results2, err := Sync(changes, target2)
	if err != nil {
		t.Fatalf("sync2: %v", err)
	}
	for _, r := range results2 {
		if r.Action != "updated" {
			t.Errorf("expected updated, got %s for %s", r.Action, r.Name)
		}
	}
}
