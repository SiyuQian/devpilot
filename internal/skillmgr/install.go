package skillmgr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InstallSkill writes skill files into <baseDir>/<skillName>/.
// baseDir is the absolute path to the skills directory (e.g., ".claude/skills" or "~/.claude/skills").
// Existing files are silently overwritten.
//
// Both skillName and each SkillFile.Path can originate from the remote
// skills/index.json, so the caller cannot assume they are trusted. Every
// input is validated to prevent a malicious catalog from writing outside
// skillDir via absolute paths or "../" segments.
func InstallSkill(baseDir, skillName string, files []SkillFile) error {
	if err := validateSkillName(skillName); err != nil {
		return err
	}

	skillDir := filepath.Join(baseDir, skillName)
	cleanSkillDir := filepath.Clean(skillDir)

	for _, f := range files {
		target, err := resolveSkillFilePath(cleanSkillDir, f.Path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", f.Path, err)
		}
		if err := os.WriteFile(target, f.Content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", f.Path, err)
		}
	}
	return nil
}

func validateSkillName(skillName string) error {
	if skillName == "" {
		return fmt.Errorf("invalid skillName %q: must not be empty", skillName)
	}
	if skillName == "." || skillName == ".." {
		return fmt.Errorf("invalid skillName %q: must not be %q", skillName, skillName)
	}
	if strings.ContainsRune(skillName, '/') || strings.ContainsRune(skillName, '\\') ||
		strings.ContainsRune(skillName, filepath.Separator) {
		return fmt.Errorf("invalid skillName %q: must not contain path separators", skillName)
	}
	if filepath.IsAbs(skillName) {
		return fmt.Errorf("invalid skillName %q: must not be an absolute path", skillName)
	}
	return nil
}

func resolveSkillFilePath(cleanSkillDir, filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("invalid file path %q: must not be empty", filePath)
	}
	// filepath.Join silently strips a leading separator, so "/etc/passwd"
	// would land under skillDir as "etc/passwd". Reject absolutes first.
	converted := filepath.FromSlash(filePath)
	if filepath.IsAbs(filePath) || filepath.IsAbs(converted) || strings.HasPrefix(filePath, "/") {
		return "", fmt.Errorf("invalid file path %q: must not be absolute", filePath)
	}
	target := filepath.Join(cleanSkillDir, converted)
	cleaned := filepath.Clean(target)
	// The "+ Separator" prefix check prevents "/a/skilldir-evil" from
	// matching "/a/skilldir" as a descendant.
	if cleaned != cleanSkillDir && !strings.HasPrefix(cleaned, cleanSkillDir+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid file path %q: escapes skill directory", filePath)
	}
	if cleaned == cleanSkillDir {
		return "", fmt.Errorf("invalid file path %q: resolves to skill directory itself", filePath)
	}
	return target, nil
}
