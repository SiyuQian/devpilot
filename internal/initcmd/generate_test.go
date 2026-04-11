package initcmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/project"
	"github.com/siyuqian/devpilot/internal/skillmgr"
)

func TestConfigureBoard_NonInteractiveSkips(t *testing.T) {
	dir := t.TempDir()

	opts := GenerateOpts{Dir: dir, Interactive: false}
	if err := ConfigureBoard(opts, nil); err != nil {
		t.Fatalf("ConfigureBoard failed: %v", err)
	}

	// Should not have created .devpilot.yaml
	if _, err := os.Stat(filepath.Join(dir, ".devpilot.yaml")); !os.IsNotExist(err) {
		t.Error(".devpilot.yaml should not exist in non-interactive mode")
	}
}

func TestConfigureBoard_InteractiveWithListBoards(t *testing.T) {
	dir := t.TempDir()

	input := strings.NewReader("1\n")
	opts := GenerateOpts{
		Dir:         dir,
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	listBoards := func() ([]Board, error) {
		return []Board{{Name: "Dev Board"}, {Name: "Other Board"}}, nil
	}

	if err := ConfigureBoard(opts, listBoards); err != nil {
		t.Fatalf("ConfigureBoard failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".devpilot.yaml"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), "Dev Board") {
		t.Errorf(".devpilot.yaml does not contain board name, got: %s", string(data))
	}
}

func TestConfigureBoard_InteractiveFreeText(t *testing.T) {
	dir := t.TempDir()

	input := strings.NewReader("My Custom Board\n")
	opts := GenerateOpts{
		Dir:         dir,
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	if err := ConfigureBoard(opts, nil); err != nil {
		t.Fatalf("ConfigureBoard failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".devpilot.yaml"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), "My Custom Board") {
		t.Errorf(".devpilot.yaml does not contain board name, got: %s", string(data))
	}
}

func TestConfigureBoard_PreservesExistingConfig(t *testing.T) {
	dir := t.TempDir()

	// Write a config with existing skills entry.
	initial := []byte("skills:\n- name: pm\n  source: github.com/siyuqian/devpilot\n  installedAt: 2026-01-01T00:00:00Z\n")
	if err := os.WriteFile(filepath.Join(dir, ".devpilot.yaml"), initial, 0644); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader("My Board\n")
	opts := GenerateOpts{
		Dir:         dir,
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	if err := ConfigureBoard(opts, nil); err != nil {
		t.Fatalf("ConfigureBoard failed: %v", err)
	}

	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Board != "My Board" {
		t.Errorf("Board = %q, want %q", cfg.Board, "My Board")
	}
	if len(cfg.Skills) != 1 || cfg.Skills[0].Name != "pm" {
		t.Errorf("existing skill entry was overwritten, skills = %v", cfg.Skills)
	}
}

func TestInstallSkills_NonInteractiveSkips(t *testing.T) {
	dir := t.TempDir()
	opts := GenerateOpts{Dir: dir, Interactive: false}

	called := false
	installOpts := SkillInstallOpts{
		SelectFn: func(catalog []skillmgr.CatalogEntry) ([]string, error) {
			called = true
			return []string{"pm"}, nil
		},
	}

	if err := InstallSkills(opts, installOpts); err != nil {
		t.Fatalf("InstallSkills: %v", err)
	}
	if called {
		t.Error("selectFn should not be called in non-interactive mode")
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills")); !os.IsNotExist(err) {
		t.Error(".claude/skills should not exist when skipped")
	}
}

func stubCatalogFn() ([]skillmgr.CatalogEntry, error) {
	return []skillmgr.CatalogEntry{
		{Name: "pm", Description: "Product manager skill"},
		{Name: "trello", Description: "Trello integration"},
	}, nil
}

func TestInstallSkills_InteractiveInstalls(t *testing.T) {
	dir := t.TempDir()
	opts := GenerateOpts{Dir: dir, Interactive: true}

	installOpts := SkillInstallOpts{
		SelectFn: func(catalog []skillmgr.CatalogEntry) ([]string, error) {
			return []string{"pm"}, nil
		},
		FetchCatalogFn: stubCatalogFn,
		FetchSkillFn: func(name, tag string) ([]skillmgr.SkillFile, error) {
			return []skillmgr.SkillFile{
				{Path: "SKILL.md", Content: []byte("---\nname: " + name + "\n---")},
			}, nil
		},
	}

	if err := InstallSkills(opts, installOpts); err != nil {
		t.Fatalf("InstallSkills: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills", "pm", "SKILL.md")); err != nil {
		t.Errorf("SKILL.md not created: %v", err)
	}
}

func TestInstallSkills_NoSelection(t *testing.T) {
	dir := t.TempDir()
	opts := GenerateOpts{Dir: dir, Interactive: true}

	installOpts := SkillInstallOpts{
		SelectFn: func(catalog []skillmgr.CatalogEntry) ([]string, error) {
			return nil, nil // user selected nothing
		},
		FetchCatalogFn: stubCatalogFn,
	}

	if err := InstallSkills(opts, installOpts); err != nil {
		t.Fatalf("InstallSkills: %v", err)
	}
}
