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

func (m TUIModel) renderView() string {
	if m.width < 60 || m.height < 15 {
		return fmt.Sprintf("  Terminal too small (need 60x15, have %dx%d). Resize or use --no-tui.", m.width, m.height)
	}

	var sections []string

	sections = append(sections, renderHeader(m))
	sections = append(sections, renderStatusAndActive(m))
	sections = append(sections, renderLogPane(m))
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

	right := fmt.Sprintf("[%s] [q: quit]", phaseText)

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
		fmt.Sprintf("    Phase: %s", m.activeCard.status),
	}
	return activeCardStyle.Render(strings.Join(lines, "\n"))
}

func renderStatusAndActive(m TUIModel) string {
	status := renderStatusPanel(m)
	active := renderActiveTask(m)

	statusWidth := 22
	activeWidth := m.width - statusWidth - 3
	if activeWidth < 10 {
		activeWidth = 10
	}

	statusRendered := lipgloss.NewStyle().Width(statusWidth).Render(status)
	activeRendered := lipgloss.NewStyle().Width(activeWidth).Render(active)

	return lipgloss.JoinHorizontal(lipgloss.Top, statusRendered, " ", activeRendered)
}

func renderLogPane(m TUIModel) string {
	if len(m.logLines) == 0 {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Width(m.width - 4).
			Render("  (no output yet)")
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(m.width - 4).
		Render(m.viewport.View())
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
