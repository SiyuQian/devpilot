package openspec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanChanges_empty(t *testing.T) {
	dir := t.TempDir()
	changesDir := filepath.Join(dir, "openspec", "changes")
	if err := os.MkdirAll(changesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	changes, err := ScanChanges(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("expected 0 changes, got %d", len(changes))
	}
}

func TestScanChanges_findsChanges(t *testing.T) {
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "openspec", "changes", "add-auth")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	proposal := "# Add Auth\nImplement authentication."
	tasks := "- [ ] Create auth module\n- [ ] Add tests"

	if err := os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte(proposal), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte(tasks), 0o644); err != nil {
		t.Fatal(err)
	}

	changes, err := ScanChanges(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	c := changes[0]
	if c.Name != "add-auth" {
		t.Errorf("expected name 'add-auth', got %q", c.Name)
	}

	expectedDesc := proposal + "\n\n---\n\n" + tasks
	if c.Description != expectedDesc {
		t.Errorf("expected description %q, got %q", expectedDesc, c.Description)
	}
}

func TestScanChanges_noOpenSpecDir(t *testing.T) {
	dir := t.TempDir()

	_, err := ScanChanges(dir)
	if err == nil {
		t.Fatal("expected error for missing openspec/changes dir, got nil")
	}
}

func TestCheckInstalled_notInstalled(t *testing.T) {
	err := CheckInstalled("nonexistent-binary-xyz-12345")
	if err == nil {
		t.Fatal("expected error for nonexistent binary, got nil")
	}
}
