package taskrunner

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIInit(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil command, expected non-nil")
	}
}

func TestTUIUpdateWindowSize(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model := updated.(TUIModel)

	if model.width != 120 {
		t.Errorf("width = %d, want 120", model.width)
	}
	if model.height != 40 {
		t.Errorf("height = %d, want 40", model.height)
	}
	if !model.ready {
		t.Error("ready = false, want true")
	}
}

func TestTUIUpdateCardStarted(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	event := CardStartedEvent{CardID: "c1", CardName: "Fix bug", Branch: "task/c1-fix"}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.activeCard == nil {
		t.Fatal("activeCard is nil, expected non-nil")
	}
	if model.activeCard.name != "Fix bug" {
		t.Errorf("activeCard.name = %q, want %q", model.activeCard.name, "Fix bug")
	}
	if model.activeCard.branch != "task/c1-fix" {
		t.Errorf("activeCard.branch = %q, want %q", model.activeCard.branch, "task/c1-fix")
	}
	if model.phase != "running" {
		t.Errorf("phase = %q, want %q", model.phase, "running")
	}
}

func TestTUIUpdateCardDone(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	// First set an active card
	m.activeCard = &cardState{id: "c1", name: "Fix bug", status: "running"}

	event := CardDoneEvent{CardID: "c1", CardName: "Fix bug", PRURL: "http://pr/1", Duration: 3 * time.Minute}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.activeCard != nil {
		t.Error("activeCard should be nil after done")
	}
	if len(model.history) != 1 {
		t.Fatalf("history len = %d, want 1", len(model.history))
	}
	if model.history[0].status != "done" {
		t.Errorf("history[0].status = %q, want %q", model.history[0].status, "done")
	}
	if model.history[0].prURL != "http://pr/1" {
		t.Errorf("history[0].prURL = %q, want %q", model.history[0].prURL, "http://pr/1")
	}
	if model.phase != "polling" {
		t.Errorf("phase = %q, want %q", model.phase, "polling")
	}
}

func TestTUIUpdateCardFailed(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	m.activeCard = &cardState{id: "c1", name: "Fix bug", status: "running"}

	event := CardFailedEvent{CardID: "c1", CardName: "Fix bug", ErrMsg: "oops", Duration: time.Minute}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.activeCard != nil {
		t.Error("activeCard should be nil after failure")
	}
	if len(model.history) != 1 {
		t.Fatalf("history len = %d, want 1", len(model.history))
	}
	if model.history[0].status != "failed" {
		t.Errorf("history[0].status = %q, want %q", model.history[0].status, "failed")
	}
	if model.history[0].errMsg != "oops" {
		t.Errorf("history[0].errMsg = %q, want %q", model.history[0].errMsg, "oops")
	}
}

func TestTUIUpdateCardOutput(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	event := CardOutputEvent{Line: OutputLine{Stream: "stdout", Text: "hello world"}}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if len(model.logLines) != 1 {
		t.Fatalf("logLines len = %d, want 1", len(model.logLines))
	}
	if model.logLines[0] != "[stdout] hello world" {
		t.Errorf("logLines[0] = %q, want %q", model.logLines[0], "[stdout] hello world")
	}
}

func TestTUIUpdateNoTasks(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	event := NoTasksEvent{NextPoll: 5 * time.Second}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.phase != "idle" {
		t.Errorf("phase = %q, want %q", model.phase, "idle")
	}
}

func TestTUIUpdateRunnerStopped(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	event := RunnerStoppedEvent{}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.phase != "stopped" {
		t.Errorf("phase = %q, want %q", model.phase, "stopped")
	}
}

func TestTUIKeyQuit(t *testing.T) {
	ch := make(chan Event, 1)
	cancelCalled := false
	cancel := func() { cancelCalled = true }

	m := NewTUIModel("Test Board", ch, cancel)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !cancelCalled {
		t.Error("cancel was not called on ctrl+c")
	}
	// The command should be tea.Quit
	if cmd == nil {
		t.Fatal("cmd is nil, expected tea.Quit")
	}
}

func TestTUIUpdateRunnerDone(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	updated, _ := m.Update(runnerDoneMsg{})
	model := updated.(TUIModel)

	if model.phase != "stopped" {
		t.Errorf("phase = %q, want %q", model.phase, "stopped")
	}
}

func TestTickUpdatesView(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Error("tickMsg should return a non-nil cmd to continue ticking")
	}
}

func TestKeyUpDown(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	// Set up viewport with some content
	m.ready = true
	m.width = 100
	m.height = 30

	// Add enough content so we can scroll
	for i := 0; i < 50; i++ {
		m.logLines = append(m.logLines, "line")
	}
	m.viewport.SetContent(joinLines(m.logLines))

	// Test that key j delegates to viewport (should not error)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model := updated.(TUIModel)
	_ = model // just verify it doesn't panic

	// Test key k
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = updated.(TUIModel)
	_ = model
}

func TestKeyQByRune(t *testing.T) {
	ch := make(chan Event, 1)
	cancelCalled := false
	cancel := func() { cancelCalled = true }

	m := NewTUIModel("Test Board", ch, cancel)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !cancelCalled {
		t.Error("cancel was not called on 'q' key")
	}
	if cmd == nil {
		t.Fatal("cmd is nil, expected tea.Quit")
	}
}

func TestTUIUpdateRunnerStarted(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	event := RunnerStartedEvent{
		BoardName: "Sprint Board",
		BoardID:   "board123",
		Lists: map[string]string{
			"Ready":       "list1",
			"In Progress": "list2",
			"Done":        "list3",
			"Failed":      "list4",
		},
	}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.boardID != "board123" {
		t.Errorf("boardID = %q, want %q", model.boardID, "board123")
	}
	if len(model.lists) != 4 {
		t.Errorf("lists len = %d, want 4", len(model.lists))
	}
	if model.phase != "polling" {
		t.Errorf("phase = %q, want %q", model.phase, "polling")
	}
}
