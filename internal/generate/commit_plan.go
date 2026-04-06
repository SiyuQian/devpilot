package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommitEntry represents a single commit in the plan.
type CommitEntry struct {
	Message string   `json:"message"`
	Files   []string `json:"files"`
}

// ExcludedFile represents a file excluded from committing.
type ExcludedFile struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

// CommitPlan represents the full commit plan returned by Claude.
type CommitPlan struct {
	Commits  []CommitEntry  `json:"commits"`
	Excluded []ExcludedFile `json:"excluded"`
}

// parseCommitPlan parses JSON output from Claude into a CommitPlan.
// On parse failure, falls back to a single commit with all files.
func parseCommitPlan(output string, stagedFiles []string) CommitPlan {
	output = cleanOutput(output)

	var plan CommitPlan
	if err := json.Unmarshal([]byte(output), &plan); err != nil {
		return fallbackPlan(output, stagedFiles)
	}

	if len(plan.Commits) == 0 {
		return fallbackPlan(output, stagedFiles)
	}

	return plan
}

// fallbackPlan creates a single-commit plan using raw output as message.
func fallbackPlan(rawMessage string, stagedFiles []string) CommitPlan {
	msg := strings.TrimSpace(rawMessage)
	if msg == "" {
		msg = "chore: update files"
	}
	return CommitPlan{
		Commits: []CommitEntry{
			{Message: msg, Files: stagedFiles},
		},
	}
}

// validatePlan checks that all plan files exist in staged changes and
// all staged files appear in exactly one commit or excluded.
// Returns warnings and a corrected plan.
func validatePlan(plan CommitPlan, stagedFiles []string) (CommitPlan, []string) {
	staged := make(map[string]bool, len(stagedFiles))
	for _, f := range stagedFiles {
		staged[f] = true
	}

	excluded := make(map[string]bool)
	for _, e := range plan.Excluded {
		excluded[e.File] = true
	}

	var warnings []string
	covered := make(map[string]bool)

	// Check each commit's files exist in staged
	for i := range plan.Commits {
		var validFiles []string
		for _, f := range plan.Commits[i].Files {
			if !staged[f] {
				warnings = append(warnings, fmt.Sprintf("removed unknown file %q from commit %d", f, i+1))
				continue
			}
			validFiles = append(validFiles, f)
			covered[f] = true
		}
		plan.Commits[i].Files = validFiles
	}

	// Check for staged files missing from plan
	var missing []string
	for _, f := range stagedFiles {
		if !covered[f] && !excluded[f] {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		// Add missing files to the last commit
		last := len(plan.Commits) - 1
		plan.Commits[last].Files = append(plan.Commits[last].Files, missing...)
		warnings = append(warnings, fmt.Sprintf("added %d missing file(s) to commit %d", len(missing), last+1))
	}

	// Remove empty commits
	var nonEmpty []CommitEntry
	for _, c := range plan.Commits {
		if len(c.Files) > 0 {
			nonEmpty = append(nonEmpty, c)
		}
	}
	plan.Commits = nonEmpty

	return plan, warnings
}

// serializePlanToMarkdown converts a CommitPlan to an editable markdown format.
func serializePlanToMarkdown(plan CommitPlan) string {
	var sb strings.Builder
	for i, c := range plan.Commits {
		fmt.Fprintf(&sb, "## Commit %d\n\n", i+1)
		fmt.Fprintf(&sb, "Message: %s\n\n", c.Message)
		sb.WriteString("Files:\n")
		for _, f := range c.Files {
			fmt.Fprintf(&sb, "- %s\n", f)
		}
		sb.WriteString("\n")
	}
	if len(plan.Excluded) > 0 {
		sb.WriteString("## Excluded\n\n")
		for _, e := range plan.Excluded {
			fmt.Fprintf(&sb, "- %s (%s)\n", e.File, e.Reason)
		}
	}
	return sb.String()
}

// parsePlanFromMarkdown parses the markdown format back into a CommitPlan.
func parsePlanFromMarkdown(text string) (CommitPlan, error) {
	var plan CommitPlan
	sections := strings.Split(text, "## ")

	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		if strings.HasPrefix(section, "Excluded") {
			lines := strings.Split(section, "\n")
			for _, line := range lines[1:] {
				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, "- ") {
					continue
				}
				line = strings.TrimPrefix(line, "- ")
				if idx := strings.LastIndex(line, " ("); idx > 0 {
					file := line[:idx]
					reason := strings.TrimSuffix(line[idx+2:], ")")
					plan.Excluded = append(plan.Excluded, ExcludedFile{File: file, Reason: reason})
				} else {
					plan.Excluded = append(plan.Excluded, ExcludedFile{File: line, Reason: "excluded by user"})
				}
			}
			continue
		}

		if !strings.HasPrefix(section, "Commit ") {
			continue
		}

		lines := strings.Split(section, "\n")
		var commit CommitEntry
		inFiles := false
		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Message: ") {
				commit.Message = strings.TrimPrefix(line, "Message: ")
				inFiles = false
			} else if line == "Files:" {
				inFiles = true
			} else if inFiles && strings.HasPrefix(line, "- ") {
				commit.Files = append(commit.Files, strings.TrimPrefix(line, "- "))
			}
		}
		if commit.Message != "" && len(commit.Files) > 0 {
			plan.Commits = append(plan.Commits, commit)
		}
	}

	if len(plan.Commits) == 0 {
		return plan, fmt.Errorf("no commits found in edited plan")
	}
	return plan, nil
}

// editPlanInTerminal opens $EDITOR with the plan in markdown format.
func editPlanInTerminal(plan CommitPlan) (CommitPlan, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	content := serializePlanToMarkdown(plan)

	tmpFile, err := os.CreateTemp("", "devpilot-plan-*.md")
	if err != nil {
		return plan, err
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		return plan, err
	}
	_ = tmpFile.Close()

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return plan, fmt.Errorf("editor: %w", err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return plan, err
	}

	return parsePlanFromMarkdown(string(data))
}
