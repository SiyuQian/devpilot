package taskrunner

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func joinLines(lines []string) string {
	var s string
	for i, l := range lines {
		if i > 0 {
			s += "\n"
		}
		s += l
	}
	return s
}

func toolSummary(toolName string, input map[string]any) string {
	switch toolName {
	case "Read":
		if fp, ok := input["file_path"].(string); ok {
			return shortenPath(fp)
		}
	case "Edit", "Write":
		if fp, ok := input["file_path"].(string); ok {
			return shortenPath(fp)
		}
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			if len(cmd) > 60 {
				return cmd[:60] + "..."
			}
			return cmd
		}
	case "Grep":
		if pat, ok := input["pattern"].(string); ok {
			return pat
		}
	case "Glob":
		if pat, ok := input["pattern"].(string); ok {
			return pat
		}
	}
	return ""
}

func extractFilePath(input map[string]any) string {
	if fp, ok := input["file_path"].(string); ok {
		return fp
	}
	return ""
}

func addUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func shortenPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}
	return strings.Join(parts[len(parts)-2:], "/")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

const maxTextLines = 2000

// wrapAndSetTextContent wraps each line in m.textLines to m.textContentWidth
// and sets the result as the text viewport content.
func (m *TUIModel) wrapAndSetTextContent() {
	if m.textContentWidth <= 0 {
		return
	}
	var wrapped []string
	for _, line := range m.textLines {
		wrapped = append(wrapped, ansi.Wordwrap(line, m.textContentWidth, ""))
	}
	m.textViewport.SetContent(joinLines(wrapped))
}
