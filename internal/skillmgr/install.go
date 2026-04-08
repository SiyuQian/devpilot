package skillmgr

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallSkill writes skill files into <baseDir>/<skillName>/.
// baseDir is the absolute path to the skills directory (e.g., ".claude/skills" or "~/.claude/skills").
// Existing files are silently overwritten.
func InstallSkill(baseDir, skillName string, files []SkillFile) error {
	skillDir := filepath.Join(baseDir, skillName)

	for _, f := range files {
		target := filepath.Join(skillDir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", f.Path, err)
		}
		if err := os.WriteFile(target, f.Content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", f.Path, err)
		}
	}
	return nil
}
