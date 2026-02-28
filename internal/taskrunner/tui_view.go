package taskrunner

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("15")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	activeCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")).
			Padding(0, 1)

	doneIcon   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✅")
	failedIcon = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("❌")

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func formatTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	k := float64(n) / 1000.0
	if k < 10 {
		return fmt.Sprintf("%.1fk", k)
	}
	return fmt.Sprintf("%.0fk", k)
}

func formatDuration(ms int) string {
	if ms < 0 {
		return "..."
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000.0)
}

func (m TUIModel) renderView() string {
	if m.width < 80 || m.height < 20 {
		return fmt.Sprintf("  Terminal too small (need 80x20, have %dx%d). Resize or use --no-tui.", m.width, m.height)
	}
	var sections []string
	sections = append(sections, renderHeader(m))
	sections = append(sections, renderStatusAndActive(m))
	sections = append(sections, renderToolsAndFiles(m))
	sections = append(sections, renderTextPane(m))
	sections = append(sections, renderFooter(m))
	return strings.Join(sections, "\n")
}

func renderHeader(m TUIModel) string {
	left := titleStyle.Render("devkit run")
	middle := fmt.Sprintf(" Board: %s", m.boardName)

	phaseText := m.phase
	switch m.phase {
	case "polling":
		phaseText = "polling..."
	case "running":
		phaseText = "▶ running"
	case "idle":
		phaseText = "waiting"
	case "stopped":
		phaseText = "■ stopped"
	}

	statsText := ""
	if m.stats.inputTokens > 0 || m.stats.outputTokens > 0 {
		statsText = fmt.Sprintf(" ↑%s ↓%s", formatTokens(m.stats.inputTokens), formatTokens(m.stats.outputTokens))
	}
	if m.stats.turns > 0 {
		statsText += fmt.Sprintf(" T:%d", m.stats.turns)
	}

	right := fmt.Sprintf("[%s]%s [q: quit]", phaseText, statsText)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(middle) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return headerStyle.Width(m.width).Render(left + middle + strings.Repeat(" ", gap) + right)
}

func renderStatusPanel(m TUIModel) string {
	if len(m.lists) == 0 {
		return "(no lists)"
	}
	var lines []string
	for _, l := range m.lists {
		lines = append(lines, fmt.Sprintf("  %-14s", l.name))
	}
	return statusStyle.Render(strings.Join(lines, "\n"))
}

func renderActiveTask(m TUIModel) string {
	if m.activeCard == nil {
		switch m.phase {
		case "idle":
			return activeCardStyle.Render("  (waiting for tasks...)")
		case "stopped":
			return activeCardStyle.Render("  (runner stopped)")
		default:
			return activeCardStyle.Render("  (polling...)")
		}
	}

	elapsed := time.Since(m.activeCard.started).Round(time.Second)
	lines := []string{
		fmt.Sprintf("  ▶ %q", m.activeCard.name),
		fmt.Sprintf("    Branch: %s", m.activeCard.branch),
		fmt.Sprintf("    Duration: %s", elapsed),
	}
	if m.activeCall != nil {
		lines = append(lines, fmt.Sprintf("    ⚡ %s %s", m.activeCall.toolName, m.activeCall.summary))
	}
	return activeCardStyle.Render(strings.Join(lines, "\n"))
}

func renderStatusAndActive(m TUIModel) string {
	status := renderStatusPanel(m)
	active := renderActiveTask(m)

	statusWidth := 22
	activeWidth := m.width - statusWidth - 1
	if activeWidth < 10 {
		activeWidth = 10
	}

	statusRendered := lipgloss.NewStyle().Width(statusWidth).Render(status)
	activeRendered := lipgloss.NewStyle().Width(activeWidth).Render(active)

	return lipgloss.JoinHorizontal(lipgloss.Top, statusRendered, " ", activeRendered)
}

func renderToolsAndFiles(m TUIModel) string {
	toolsWidth := m.width - 30 - 1 // reserve 30 for files panel, 1 for spacer
	if toolsWidth < 30 {
		toolsWidth = 30
	}
	filesWidth := m.width - toolsWidth - 1

	toolsPanel := renderToolCallsPanel(m, toolsWidth)
	filesPanel := renderFilesPanel(m, filesWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, toolsPanel, " ", filesPanel)
}

func renderToolCallsPanel(m TUIModel, width int) string {
	focusColor := lipgloss.Color("240")
	if m.focusedPane == "tools" {
		focusColor = lipgloss.Color("12")
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(focusColor).
		Width(width - 2).
		Padding(0, 1)

	if len(m.toolCalls) == 0 && m.activeCall == nil {
		return style.Render("Tool Calls\n  (none)")
	}

	return style.Render("Tool Calls\n" + m.toolViewport.View())
}

func renderToolCallsList(m TUIModel) string {
	var lines []string
	for _, tc := range m.toolCalls {
		line := fmt.Sprintf("  ✓ %-6s %-40s %s", tc.toolName, truncate(tc.summary, 40), formatDuration(tc.durationMs))
		lines = append(lines, line)
	}
	if m.activeCall != nil {
		line := fmt.Sprintf("  ⚡ %-6s %-40s %s", m.activeCall.toolName, truncate(m.activeCall.summary, 40), formatDuration(m.activeCall.durationMs))
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func renderFilesPanel(m TUIModel, width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(width - 2).
		Padding(0, 1)

	if len(m.filesRead) == 0 && len(m.filesEdited) == 0 {
		return style.Render("Files\n  (none)")
	}

	var lines []string
	lines = append(lines, "Files")
	for _, f := range m.filesEdited {
		lines = append(lines, "  E "+shortenPath(f))
	}
	for _, f := range m.filesRead {
		lines = append(lines, "  R "+shortenPath(f))
	}
	return style.Render(strings.Join(lines, "\n"))
}

func renderTextPane(m TUIModel) string {
	focusColor := lipgloss.Color("240")
	if m.focusedPane == "text" {
		focusColor = lipgloss.Color("12")
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(focusColor).
		Width(m.width - 2).
		Padding(0, 1)

	if len(m.textLines) == 0 {
		return style.Render("Claude Output\n  (no output yet)")
	}

	return style.Render("Claude Output\n" + m.textViewport.View())
}

func renderFooter(m TUIModel) string {
	var parts []string

	if m.lastErr != "" {
		parts = append(parts, errorStyle.Render("Error: "+m.lastErr))
	}

	historyStart := 0
	if len(m.history) > 5 {
		historyStart = len(m.history) - 5
	}
	var historyParts []string
	for _, h := range m.history[historyStart:] {
		icon := doneIcon
		if h.status == "failed" {
			icon = failedIcon
		}
		historyParts = append(historyParts, fmt.Sprintf("%s %q (%s)", icon, h.name, h.duration))
	}

	if len(historyParts) > 0 {
		parts = append(parts, "History: "+strings.Join(historyParts, " | "))
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}
