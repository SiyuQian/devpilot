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
	m := TUIModel{ready: true, width: 40, height: 10, phase: "idle"}
	output := m.View()
	if !strings.Contains(output, "too small") {
		t.Errorf("expected 'too small' in output, got %q", output)
	}
}

func TestViewIdle(t *testing.T) {
	m := TUIModel{
		ready:     true,
		width:     100,
		height:    30,
		phase:     "idle",
		boardName: "Sprint Board",
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
		ready:     true,
		width:     100,
		height:    30,
		phase:     "running",
		boardName: "Sprint Board",
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
		ready:     true,
		width:     100,
		height:    30,
		phase:     "polling",
		boardName: "Sprint Board",
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
		ready:     true,
		width:     100,
		height:    30,
		phase:     "polling",
		boardName: "Sprint Board",
		lastErr:   "connection refused",
	}
	output := m.View()
	if !strings.Contains(output, "connection refused") {
		t.Errorf("expected error in footer, got %q", output)
	}
}
