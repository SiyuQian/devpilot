package taskrunner

import (
	"strings"
	"testing"
	"time"
)

func TestViewNotReady(t *testing.T) {
	m := TUIModel{ready: false, phase: "starting"}
	output := m.View()
	if !strings.Contains(output, "Starting") {
		t.Errorf("expected 'Starting' in output, got %q", output)
	}
}

func TestViewTooSmall(t *testing.T) {
	m := TUIModel{ready: true, width: 40, height: 10, phase: "idle", focusedPane: "tools"}
	output := m.View()
	if !strings.Contains(output, "too small") {
		t.Errorf("expected 'too small' in output, got %q", output)
	}
}

func TestViewIdle(t *testing.T) {
	m := TUIModel{
		ready:       true,
		width:       100,
		height:      30,
		phase:       "idle",
		boardName:   "Sprint Board",
		focusedPane: "tools",
		lists: []listState{
			{name: "Ready", id: "r1"},
			{name: "Done", id: "d1"},
		},
	}
	output := m.View()
	if !strings.Contains(output, "Sprint Board") {
		t.Errorf("expected board name in output, got %q", output)
	}
	if !strings.Contains(output, "waiting") {
		t.Errorf("expected 'waiting' in output, got %q", output)
	}
}

func TestViewRunning(t *testing.T) {
	m := TUIModel{
		ready:       true,
		width:       100,
		height:      30,
		phase:       "running",
		boardName:   "Sprint Board",
		focusedPane: "tools",
		activeCard: &cardState{
			name:    "Fix login bug",
			branch:  "task/c1-fix-login",
			status:  "running",
			started: time.Now().Add(-2 * time.Minute),
		},
	}
	output := m.View()
	if !strings.Contains(output, "Fix login bug") {
		t.Errorf("expected card name in output, got %q", output)
	}
	if !strings.Contains(output, "task/c1-fix-login") {
		t.Errorf("expected branch in output, got %q", output)
	}
}

func TestViewWithHistory(t *testing.T) {
	m := TUIModel{
		ready:       true,
		width:       100,
		height:      30,
		phase:       "polling",
		boardName:   "Sprint Board",
		focusedPane: "tools",
		history: []cardState{
			{name: "Fix login", status: "done", duration: 3 * time.Minute},
			{name: "Add DB", status: "failed", errMsg: "timeout", duration: time.Minute},
		},
	}
	output := m.View()
	if !strings.Contains(output, "Fix login") {
		t.Errorf("expected 'Fix login' in history, got %q", output)
	}
	if !strings.Contains(output, "Add DB") {
		t.Errorf("expected 'Add DB' in history, got %q", output)
	}
}

func TestViewWithError(t *testing.T) {
	m := TUIModel{
		ready:       true,
		width:       100,
		height:      30,
		phase:       "polling",
		boardName:   "Sprint Board",
		focusedPane: "tools",
		lastErr:     "connection refused",
	}
	output := m.View()
	if !strings.Contains(output, "connection refused") {
		t.Errorf("expected error in footer, got %q", output)
	}
}

func TestViewHeaderShowsTokens(t *testing.T) {
	m := TUIModel{
		ready:       true,
		width:       100,
		height:      30,
		phase:       "running",
		boardName:   "Sprint Board",
		focusedPane: "tools",
		stats: sessionStats{
			inputTokens:  15000,
			outputTokens: 3500,
			turns:        5,
		},
	}
	output := m.View()
	if !strings.Contains(output, "15k") {
		t.Errorf("expected '15k' for input tokens in header, got %q", output)
	}
	if !strings.Contains(output, "3.5k") {
		t.Errorf("expected '3.5k' for output tokens in header, got %q", output)
	}
	if !strings.Contains(output, "T:5") {
		t.Errorf("expected 'T:5' for turns in header, got %q", output)
	}
}

func TestViewActiveCardShowsCurrentTool(t *testing.T) {
	m := TUIModel{
		ready:       true,
		width:       100,
		height:      30,
		phase:       "running",
		boardName:   "Sprint Board",
		focusedPane: "tools",
		activeCard: &cardState{
			name:    "Fix bug",
			branch:  "task/c1-fix",
			status:  "running",
			started: time.Now(),
		},
		activeCall: &toolCallEntry{
			toolName:   "Read",
			summary:    "taskrunner/tui.go",
			durationMs: -1,
		},
	}
	output := m.View()
	if !strings.Contains(output, "⚡") {
		t.Errorf("expected lightning bolt for active tool in output, got %q", output)
	}
	if !strings.Contains(output, "Read") {
		t.Errorf("expected 'Read' tool name in output, got %q", output)
	}
	if !strings.Contains(output, "taskrunner/tui.go") {
		t.Errorf("expected 'taskrunner/tui.go' in output, got %q", output)
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{1500, "1.5k"},
		{9999, "10.0k"},
		{10000, "10k"},
		{15000, "15k"},
		{100000, "100k"},
	}

	for _, tt := range tests {
		got := formatTokens(tt.input)
		if got != tt.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{-1, "..."},
		{0, "0ms"},
		{50, "50ms"},
		{999, "999ms"},
		{1000, "1.0s"},
		{1500, "1.5s"},
		{5200, "5.2s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.input)
		if got != tt.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
