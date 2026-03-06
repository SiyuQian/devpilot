package generate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
)

var commitTmpl = template.Must(template.ParseFS(promptsFS, "prompts/commit.tmpl"))

type commitData struct {
	NameStatus  string
	DiffStat    string
	DiffContent string
	Context     string
}

const (
	maxLinesPerFile = 200
	maxDiffChars    = 15000
)

func buildCommitPrompt(nameStatus, diffStat, diffContent, userContext string) (string, error) {
	truncated := truncateDiff(diffContent)
	var buf bytes.Buffer
	err := commitTmpl.Execute(&buf, commitData{
		NameStatus:  nameStatus,
		DiffStat:    diffStat,
		DiffContent: truncated,
		Context:     userContext,
	})
	return buf.String(), err
}

// truncateDiff truncates diff content per-file (200 lines) and total (15K chars).
func truncateDiff(diff string) string {
	if diff == "" {
		return diff
	}

	var result strings.Builder
	sections := splitDiffSections(diff)

	for _, section := range sections {
		if isBinaryDiff(section) {
			path := extractDiffPath(section)
			result.WriteString(fmt.Sprintf("Binary file: %s\n", path))
			continue
		}

		lines := strings.Split(section, "\n")
		if len(lines) > maxLinesPerFile {
			remaining := len(lines) - maxLinesPerFile
			lines = lines[:maxLinesPerFile]
			lines = append(lines, fmt.Sprintf("[truncated — %d more lines]", remaining))
		}
		truncated := strings.Join(lines, "\n")

		if result.Len()+len(truncated) > maxDiffChars {
			result.WriteString(fmt.Sprintf("\n[truncated — diff too large, %d chars remaining]\n", len(truncated)))
			break
		}
		result.WriteString(truncated)
		result.WriteString("\n")
	}

	return result.String()
}

// splitDiffSections splits a unified diff into per-file sections.
func splitDiffSections(diff string) []string {
	var sections []string
	var current strings.Builder
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") && current.Len() > 0 {
			sections = append(sections, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		sections = append(sections, current.String())
	}
	return sections
}

// isBinaryDiff checks if a diff section is for a binary file.
func isBinaryDiff(section string) bool {
	return strings.Contains(section, "Binary files") ||
		strings.Contains(section, "GIT binary patch")
}

// extractDiffPath extracts the file path from a diff header.
func extractDiffPath(section string) string {
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				return strings.TrimPrefix(parts[3], "b/")
			}
		}
	}
	return "unknown"
}

// gitOutput runs a git command and returns trimmed stdout.
func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RunCommit launches the Bubble Tea commit workflow.
func RunCommit(ctx context.Context, model, userContext string, dryRun bool) error {
	m := NewCommitModel(ctx, model, userContext, dryRun)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	fm := finalModel.(CommitModel)
	if fm.err != nil {
		if errors.Is(fm.err, errAborted) || errors.Is(fm.err, errNoChanges) {
			return nil
		}
		return fm.err
	}
	return nil
}
