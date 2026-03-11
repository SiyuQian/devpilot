package skillmgr

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallSkill writes skill files into .claude/skills/<skillName>/ under destDir.
// Existing files are silently overwritten.
func InstallSkill(destDir, skillName string, files []SkillFile) error {
	skillDir := filepath.Join(destDir, ".claude", "skills", skillName)

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
