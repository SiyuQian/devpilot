package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{Board: "My Board"}
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Board != "My Board" {
		t.Errorf("Board = %q, want %q", loaded.Board, "My Board")
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error for missing file: %v", err)
	}
	if cfg.Board != "" {
		t.Errorf("Board = %q, want empty string", cfg.Board)
	}
}

func TestExistsTrue(t *testing.T) {
	dir := t.TempDir()

	if err := Save(dir, &Config{Board: "test"}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !Exists(dir) {
		t.Error("Exists returned false, want true")
	}
}

func TestExistsFalse(t *testing.T) {
	dir := t.TempDir()

	if Exists(dir) {
		t.Error("Exists returned true for missing file, want false")
	}
}

func TestExistsMissingDir(t *testing.T) {
	if Exists("/nonexistent/path/that/does/not/exist") {
		t.Error("Exists returned true for missing dir, want false")
	}
}

func TestSaveCreatesIntermediateDirectories(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")

	if err := Save(nested, &Config{Board: "nested"}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	cfg, err := Load(nested)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Board != "nested" {
		t.Errorf("Board = %q, want %q", cfg.Board, "nested")
	}
}

func TestSaveFilePermissions(t *testing.T) {
	dir := t.TempDir()

	if err := Save(dir, &Config{Board: "test"}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, ".devpilot.json"))
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0644 {
		t.Errorf("permissions = %o, want 0644", perm)
	}
}

func TestSaveJSONFormat(t *testing.T) {
	dir := t.TempDir()

	if err := Save(dir, &Config{Board: "My Board"}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".devpilot.json"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	expected := "{\n  \"board\": \"My Board\"\n}\n"
	if string(data) != expected {
		t.Errorf("file content = %q, want %q", string(data), expected)
	}
}
