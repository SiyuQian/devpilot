package openspec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Change represents an OpenSpec change proposal found in openspec/changes/.
type Change struct {
	Name        string // directory name
	Description string // combined proposal.md + tasks.md content
}

// ScanChanges reads the openspec/changes/ directory under projectDir and returns
// a Change for each subdirectory that contains proposal.md or tasks.md.
func ScanChanges(projectDir string) ([]Change, error) {
	changesDir := filepath.Join(projectDir, "openspec", "changes")

	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return nil, fmt.Errorf("reading openspec/changes: %w", err)
	}

	var changes []Change
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		changeDir := filepath.Join(changesDir, entry.Name())
		desc := buildDescription(changeDir)
		if desc == "" {
			continue
		}

		changes = append(changes, Change{
			Name:        entry.Name(),
			Description: desc,
		})
	}

	return changes, nil
}

// buildDescription concatenates proposal.md and tasks.md from changeDir,
// separated by "\n\n---\n\n". Returns empty string if neither file exists.
func buildDescription(changeDir string) string {
	var parts []string

	for _, name := range []string{"proposal.md", "tasks.md"} {
		data, err := os.ReadFile(filepath.Join(changeDir, name))
		if err != nil {
			continue
		}
		if content := strings.TrimSpace(string(data)); content != "" {
			parts = append(parts, content)
		} else {
			continue
		}
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// CheckInstalled verifies that the given binary is available on PATH.
// Returns a helpful error message if not found.
func CheckInstalled(binary string) error {
	_, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf("%s is not installed or not in PATH: %w", binary, err)
	}
	return nil
}
