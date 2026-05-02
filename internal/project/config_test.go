package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUserConfigDir(t *testing.T) {
	dir, err := UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "devpilot")
	if dir != expected {
		t.Errorf("UserConfigDir() = %q, want %q", dir, expected)
	}
}

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

	info, err := os.Stat(filepath.Join(dir, ".devpilot.yaml"))
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0644 {
		t.Errorf("permissions = %o, want 0644", perm)
	}
}

func TestLoadConfigWithModels(t *testing.T) {
	dir := t.TempDir()
	data := "board: myboard\nmodels:\n  commit: claude-haiku-4-5\n  default: claude-sonnet-4-6\n"
	if err := os.WriteFile(filepath.Join(dir, ".devpilot.yaml"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Models["commit"] != "claude-haiku-4-5" {
		t.Errorf("got %q, want claude-haiku-4-5", cfg.Models["commit"])
	}
	if cfg.Models["default"] != "claude-sonnet-4-6" {
		t.Errorf("got %q, want claude-sonnet-4-6", cfg.Models["default"])
	}
}

func TestModelForCommand(t *testing.T) {
	cfg := &Config{Models: map[string]string{"commit": "claude-haiku-4-5", "default": "claude-sonnet-4-6"}}
	if got := cfg.ModelFor("commit"); got != "claude-haiku-4-5" {
		t.Errorf("got %q, want claude-haiku-4-5", got)
	}
	if got := cfg.ModelFor("readme"); got != "claude-sonnet-4-6" {
		t.Errorf("got %q, want claude-sonnet-4-6 (default fallback)", got)
	}
	if got := cfg.ModelFor("unknown"); got != "claude-sonnet-4-6" {
		t.Errorf("got %q, want claude-sonnet-4-6 (default fallback)", got)
	}

	empty := &Config{}
	if got := empty.ModelFor("commit"); got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestConfig_SourceField(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Board: "My Board", Source: "github"}
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Source != "github" {
		t.Errorf("expected source=github, got %q", got.Source)
	}
}

func TestConfig_OpenSpecMinVersion(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Board:              "devpilot",
		Source:             "github",
		OpenSpecMinVersion: "1.2.0",
	}
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.OpenSpecMinVersion != "1.2.0" {
		t.Errorf("expected 1.2.0, got %s", loaded.OpenSpecMinVersion)
	}
}

func TestSaveYAMLFormat(t *testing.T) {
	dir := t.TempDir()

	if err := Save(dir, &Config{Board: "My Board"}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".devpilot.yaml"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	content := string(data)
	if content != "board: My Board\n" {
		t.Errorf("file content = %q, want %q", content, "board: My Board\n")
	}
}

// TestLoad_IgnoresLegacySkillsBlock verifies that a .devpilot.yaml left over
// from before the in-repo skill manager was removed still loads cleanly. The
// 'skills:' key is silently dropped on the next Save.
func TestLoad_IgnoresLegacySkillsBlock(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `board: my-board
source: trello
skills:
  - name: devpilot-pr-review
    source: github.com/siyuqian/devpilot
    installedAt: 2026-01-15T10:00:00Z
`
	if err := os.WriteFile(filepath.Join(dir, ".devpilot.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error on legacy skills block: %v", err)
	}
	if cfg.Board != "my-board" {
		t.Errorf("Board = %q, want %q", cfg.Board, "my-board")
	}
	if cfg.Source != "trello" {
		t.Errorf("Source = %q, want %q", cfg.Source, "trello")
	}
}
