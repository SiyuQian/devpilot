package generate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles matching taskrunner conventions.
var (
	commitTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("12"))

	commitSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10"))

	commitErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("9"))

	commitMutedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	// File status colors
	statusModified = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // yellow
	statusAdded    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	statusDeleted  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
)

// View renders the current state of the commit TUI.
func (m commitModel) View() string {
	switch m.phase {
	case phaseStagingFiles:
		return m.viewStaging()
	case phaseAnalyzing:
		return m.viewAnalyzing()
	case phasePlan:
		return m.viewPlan()
	case phaseExecuting:
		return m.viewExecuting()
	case phaseDone:
		return m.viewDone()
	}
	return ""
}

func (m commitModel) viewStaging() string {
	return m.spinner.View() + " Staging changes...\n"
}

func (m commitModel) viewAnalyzing() string {
	return m.spinner.View() + " Analyzing changes...\n"
}

func (m commitModel) viewPlan() string {
	var sb strings.Builder
	statusMap := parseNameStatus(m.nameStatus)

	if m.isMultiCommit() {
		sb.WriteString(commitTitleStyle.Render("Commit Plan") + "\n\n")
		for i, c := range m.plan.Commits {
			sb.WriteString(fmt.Sprintf("  %s %s\n", commitTitleStyle.Render(fmt.Sprintf("%d.", i+1)), formatCommitMessage(c.Message)))
			for _, f := range c.Files {
				sb.WriteString(fmt.Sprintf("     %s\n", formatFileEntry(f, statusMap)))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(commitTitleStyle.Render("Commit") + "\n\n")
		c := m.plan.Commits[0]
		sb.WriteString(fmt.Sprintf("  %s\n", formatCommitMessage(c.Message)))
		for _, f := range c.Files {
			sb.WriteString(fmt.Sprintf("     %s\n", formatFileEntry(f, statusMap)))
		}
		sb.WriteString("\n")
	}

	if len(m.plan.Excluded) > 0 {
		sb.WriteString(commitMutedStyle.Render("  Excluded:") + "\n")
		for _, e := range m.plan.Excluded {
			sb.WriteString(commitMutedStyle.Render(fmt.Sprintf("     %s — %s", e.File, e.Reason)) + "\n")
		}
		sb.WriteString("\n")
	}

	for _, w := range m.warnings {
		sb.WriteString(commitErrorStyle.Render("  ⚠ "+w) + "\n")
	}

	if m.dryRun {
		sb.WriteString(commitMutedStyle.Render("  (dry-run: not committing)") + "\n")
	} else if m.isMultiCommit() {
		sb.WriteString("  [a]ccept all / [e]dit / [n]o\n")
	} else {
		sb.WriteString("  [y]es / [e]dit / [n]o\n")
	}

	return sb.String()
}

func (m commitModel) viewExecuting() string {
	var sb strings.Builder
	sb.WriteString(commitTitleStyle.Render("Committing...") + "\n\n")

	for i, c := range m.plan.Commits {
		firstLine := firstLineOf(c.Message)
		if i < len(m.completedCommits) {
			r := m.completedCommits[i]
			if r.err != nil {
				sb.WriteString(fmt.Sprintf("  %s %s\n", commitErrorStyle.Render("✗"), firstLine))
			} else {
				sb.WriteString(fmt.Sprintf("  %s %s %s\n",
					commitSuccessStyle.Render("✓"),
					commitMutedStyle.Render(r.hash),
					firstLine))
			}
		} else if i == m.currentCommit {
			sb.WriteString(fmt.Sprintf("  %s %s\n", m.spinner.View(), firstLine))
		} else {
			sb.WriteString(fmt.Sprintf("  %s %s\n", commitMutedStyle.Render("○"), commitMutedStyle.Render(firstLine)))
		}
	}

	return sb.String()
}

func (m commitModel) viewDone() string {
	var sb strings.Builder

	if m.err != nil {
		if errors.Is(m.err, errAborted) {
			sb.WriteString(commitMutedStyle.Render("Aborted.") + "\n")
			return sb.String()
		}
		if errors.Is(m.err, errNoChanges) {
			sb.WriteString("No changes to commit.\n")
			return sb.String()
		}
		sb.WriteString(commitErrorStyle.Render("Error: "+m.err.Error()) + "\n")
		if len(m.completedCommits) > 0 {
			sb.WriteString("\n" + commitSuccessStyle.Render("Completed before failure:") + "\n")
			for _, r := range m.completedCommits {
				if r.err == nil {
					sb.WriteString(fmt.Sprintf("  %s %s %s\n",
						commitSuccessStyle.Render("✓"),
						commitMutedStyle.Render(r.hash),
						firstLineOf(r.message)))
				}
			}
		}
		return sb.String()
	}

	if m.dryRun {
		return m.viewPlan()
	}

	sb.WriteString(commitSuccessStyle.Render("Done!") + "\n\n")
	for _, r := range m.completedCommits {
		sb.WriteString(fmt.Sprintf("  %s %s %s\n",
			commitSuccessStyle.Render("✓"),
			commitMutedStyle.Render(r.hash),
			firstLineOf(r.message)))
	}
	sb.WriteString("\n")

	return sb.String()
}

// Helper functions

func formatCommitMessage(msg string) string {
	first := firstLineOf(msg)
	// Highlight type(scope): prefix
	if idx := strings.Index(first, ":"); idx > 0 {
		prefix := first[:idx+1]
		rest := first[idx+1:]
		return commitTitleStyle.Render(prefix) + rest
	}
	return first
}

func firstLineOf(s string) string {
	if idx := strings.Index(s, "\n"); idx >= 0 {
		return s[:idx]
	}
	return s
}

// parseNameStatus parses `git diff --name-status` output into a file->status map.
func parseNameStatus(nameStatus string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(nameStatus, "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			m[parts[1]] = parts[0]
		}
	}
	return m
}

func formatFileEntry(file string, statusMap map[string]string) string {
	status := statusMap[file]
	var styledStatus string
	switch status {
	case "A":
		styledStatus = statusAdded.Render("A")
	case "D":
		styledStatus = statusDeleted.Render("D")
	default:
		styledStatus = statusModified.Render("M")
	}
	return fmt.Sprintf("%s %s", styledStatus, file)
}
