package initcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/siyuqian/devpilot/internal/project"
)

func TestDetectHasBoardConfig(t *testing.T) {
	dir := t.TempDir()

	// Without .devpilot.yaml
	s := Detect(dir)
	if s.HasBoardConfig {
		t.Error("HasBoardConfig = true, want false")
	}

	// With .devpilot.yaml but no board
	if err := project.Save(dir, &project.Config{}); err != nil {
		t.Fatal(err)
	}
	s = Detect(dir)
	if s.HasBoardConfig {
		t.Error("HasBoardConfig = true for empty board, want false")
	}

	// With .devpilot.yaml and board set
	if err := project.Save(dir, &project.Config{Board: "My Board"}); err != nil {
		t.Fatal(err)
	}
	s = Detect(dir)
	if !s.HasBoardConfig {
		t.Error("HasBoardConfig = false, want true")
	}
}

func TestDetectHasSkills(t *testing.T) {
	dir := t.TempDir()

	// Without skills dir
	s := Detect(dir)
	if s.HasSkills {
		t.Error("HasSkills = true, want false")
	}

	// With empty skills dir
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "skills"), 0755); err != nil {
		t.Fatal(err)
	}
	s = Detect(dir)
	if s.HasSkills {
		t.Error("HasSkills = true for empty skills dir, want false")
	}

	// With a subdirectory but no SKILL.md
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "skills", "my-skill"), 0755); err != nil {
		t.Fatal(err)
	}
	s = Detect(dir)
	if s.HasSkills {
		t.Error("HasSkills = true for skill dir without SKILL.md, want false")
	}

	// With a subdirectory containing SKILL.md
	if err := os.WriteFile(filepath.Join(dir, ".claude", "skills", "my-skill", "SKILL.md"), []byte("---\nname: test\n---"), 0644); err != nil {
		t.Fatal(err)
	}
	s = Detect(dir)
	if !s.HasSkills {
		t.Error("HasSkills = false, want true")
	}
}

func TestDetectIsGitRepo(t *testing.T) {
	dir := t.TempDir()

	// Without .git
	s := Detect(dir)
	if s.IsGitRepo {
		t.Error("IsGitRepo = true, want false")
	}

	// With .git directory
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	s = Detect(dir)
	if !s.IsGitRepo {
		t.Error("IsGitRepo = false, want true")
	}
}

func TestDetectWorkDir(t *testing.T) {
	dir := t.TempDir()
	s := Detect(dir)
	if s.WorkDir != dir {
		t.Errorf("WorkDir = %q, want %q", s.WorkDir, dir)
	}
}

func TestDetectSource(t *testing.T) {
	dir := t.TempDir()

	// No config file: Source should be ""
	s := Detect(dir)
	if s.Source != "" {
		t.Errorf("Source = %q, want empty string when no config", s.Source)
	}

	// Config with source=github
	if err := project.Save(dir, &project.Config{Source: "github"}); err != nil {
		t.Fatal(err)
	}
	s = Detect(dir)
	if s.Source != "github" {
		t.Errorf("Source = %q, want %q", s.Source, "github")
	}

	// Config with source=trello
	if err := project.Save(dir, &project.Config{Source: "trello", Board: "My Board"}); err != nil {
		t.Fatal(err)
	}
	s = Detect(dir)
	if s.Source != "trello" {
		t.Errorf("Source = %q, want %q", s.Source, "trello")
	}
}
