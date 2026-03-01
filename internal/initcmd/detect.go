package initcmd

import (
	"os"
	"path/filepath"

	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/siyuqian/devpilot/internal/project"
)

// Status holds the detection results for a project directory.
type Status struct {
	HasClaudeMD    bool
	HasTrelloCreds bool
	HasBoardConfig bool
HasSkills      bool
	IsGitRepo      bool
	WorkDir        string
}

// Detect inspects the given directory and returns a Status with what's configured.
func Detect(dir string) *Status {
	s := &Status{WorkDir: dir}

	// CLAUDE.md
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
		s.HasClaudeMD = true
	}

	// Trello credentials
	if _, err := auth.Load("trello"); err == nil {
		s.HasTrelloCreds = true
	}

	// Board config in .devpilot.json
	cfg, err := project.Load(dir)
	if err == nil && cfg.Board != "" {
		s.HasBoardConfig = true
	}

	// Skills: check for subdirectories containing SKILL.md
	skillsDir := filepath.Join(dir, ".claude", "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				if _, err := os.Stat(filepath.Join(skillsDir, e.Name(), "SKILL.md")); err == nil {
					s.HasSkills = true
					break
				}
			}
		}
	}

	// Git repo
	if info, err := os.Stat(filepath.Join(dir, ".git")); err == nil && info.IsDir() {
		s.IsGitRepo = true
	}

	return s
}
