package initcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/siyuqian/developer-kit/internal/project"
)

func TestDetectHasClaudeMD(t *testing.T) {
	dir := t.TempDir()

	// Without CLAUDE.md
	s := Detect(dir)
	if s.HasClaudeMD {
		t.Error("HasClaudeMD = true, want false")
	}

	// With CLAUDE.md
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# test"), 0644)
	s = Detect(dir)
	if !s.HasClaudeMD {
		t.Error("HasClaudeMD = false, want true")
	}
}

func TestDetectHasBoardConfig(t *testing.T) {
	dir := t.TempDir()

	// Without .devkit.json
	s := Detect(dir)
	if s.HasBoardConfig {
		t.Error("HasBoardConfig = true, want false")
	}

	// With .devkit.json but no board
	project.Save(dir, &project.Config{})
	s = Detect(dir)
	if s.HasBoardConfig {
		t.Error("HasBoardConfig = true for empty board, want false")
	}

	// With .devkit.json and board set
	project.Save(dir, &project.Config{Board: "My Board"})
	s = Detect(dir)
	if !s.HasBoardConfig {
		t.Error("HasBoardConfig = false, want true")
	}
}

func TestDetectHasGitHooks(t *testing.T) {
	dir := t.TempDir()

	// Without hooks
	s := Detect(dir)
	if s.HasGitHooks {
		t.Error("HasGitHooks = true, want false")
	}

	// With pre-push hook
	hooksDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(hooksDir, 0755)
	os.WriteFile(filepath.Join(hooksDir, "pre-push"), []byte("#!/bin/sh"), 0755)
	s = Detect(dir)
	if !s.HasGitHooks {
		t.Error("HasGitHooks = false, want true")
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
	os.MkdirAll(filepath.Join(dir, ".claude", "skills"), 0755)
	s = Detect(dir)
	if s.HasSkills {
		t.Error("HasSkills = true for empty skills dir, want false")
	}

	// With a subdirectory but no SKILL.md
	os.MkdirAll(filepath.Join(dir, ".claude", "skills", "my-skill"), 0755)
	s = Detect(dir)
	if s.HasSkills {
		t.Error("HasSkills = true for skill dir without SKILL.md, want false")
	}

	// With a subdirectory containing SKILL.md
	os.WriteFile(filepath.Join(dir, ".claude", "skills", "my-skill", "SKILL.md"), []byte("---\nname: test\n---"), 0644)
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
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
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
