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

	// Add enough tool calls so we can scroll
	for i := 0; i < 50; i++ {
		m.toolCalls = append(m.toolCalls, toolCallEntry{toolName: "Read", summary: "file.go"})
	}
	m.toolViewport.SetContent(renderToolCallsList(m))

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

func TestTUIUpdateToolStart(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	event := ToolStartEvent{
		ToolName: "Read",
		Input:    map[string]any{"file_path": "/home/user/project/main.go"},
	}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.activeCall == nil {
		t.Fatal("activeCall is nil, expected non-nil")
	}
	if model.activeCall.toolName != "Read" {
		t.Errorf("activeCall.toolName = %q, want %q", model.activeCall.toolName, "Read")
	}
	if model.activeCall.summary != "project/main.go" {
		t.Errorf("activeCall.summary = %q, want %q", model.activeCall.summary, "project/main.go")
	}
	if model.activeCall.durationMs != -1 {
		t.Errorf("activeCall.durationMs = %d, want -1", model.activeCall.durationMs)
	}
}

func TestTUIUpdateToolResult(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	m.activeCall = &toolCallEntry{
		toolName:   "Read",
		summary:    "project/main.go",
		durationMs: -1,
		timestamp:  time.Now(),
	}

	event := ToolResultEvent{ToolName: "Read", DurationMs: 150}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.activeCall != nil {
		t.Error("activeCall should be nil after ToolResultEvent")
	}
	if len(model.toolCalls) != 1 {
		t.Fatalf("toolCalls len = %d, want 1", len(model.toolCalls))
	}
	if model.toolCalls[0].durationMs != 150 {
		t.Errorf("toolCalls[0].durationMs = %d, want 150", model.toolCalls[0].durationMs)
	}
	if model.toolCalls[0].toolName != "Read" {
		t.Errorf("toolCalls[0].toolName = %q, want %q", model.toolCalls[0].toolName, "Read")
	}
}

func TestTUIUpdateTextOutput(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	event := TextOutputEvent{Text: "Analyzing the codebase..."}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if len(model.textLines) != 1 {
		t.Fatalf("textLines len = %d, want 1", len(model.textLines))
	}
	if model.textLines[0] != "Analyzing the codebase..." {
		t.Errorf("textLines[0] = %q, want %q", model.textLines[0], "Analyzing the codebase...")
	}
}

func TestTUIUpdateStatsUpdate(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	// First stats event
	event := StatsUpdateEvent{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 200, Turns: 1}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.stats.inputTokens != 1000 {
		t.Errorf("stats.inputTokens = %d, want 1000", model.stats.inputTokens)
	}
	if model.stats.outputTokens != 500 {
		t.Errorf("stats.outputTokens = %d, want 500", model.stats.outputTokens)
	}
	if model.stats.cacheReadTokens != 200 {
		t.Errorf("stats.cacheReadTokens = %d, want 200", model.stats.cacheReadTokens)
	}
	if model.stats.turns != 1 {
		t.Errorf("stats.turns = %d, want 1", model.stats.turns)
	}

	// Second stats event - should accumulate
	event2 := StatsUpdateEvent{InputTokens: 500, OutputTokens: 300, CacheReadTokens: 100, Turns: 2}
	updated2, _ := model.Update(event2)
	model2 := updated2.(TUIModel)

	if model2.stats.inputTokens != 1500 {
		t.Errorf("stats.inputTokens = %d, want 1500", model2.stats.inputTokens)
	}
	if model2.stats.outputTokens != 800 {
		t.Errorf("stats.outputTokens = %d, want 800", model2.stats.outputTokens)
	}
	if model2.stats.turns != 2 {
		t.Errorf("stats.turns = %d, want 2", model2.stats.turns)
	}
}

func TestTUIUpdateToolStartTracksFiles(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	// Read a file
	event1 := ToolStartEvent{ToolName: "Read", Input: map[string]any{"file_path": "/home/user/main.go"}}
	updated, _ := m.Update(event1)
	model := updated.(TUIModel)

	if len(model.filesRead) != 1 {
		t.Fatalf("filesRead len = %d, want 1", len(model.filesRead))
	}
	if model.filesRead[0] != "/home/user/main.go" {
		t.Errorf("filesRead[0] = %q, want %q", model.filesRead[0], "/home/user/main.go")
	}

	// Edit a file
	event2 := ToolStartEvent{ToolName: "Edit", Input: map[string]any{"file_path": "/home/user/util.go"}}
	updated2, _ := model.Update(event2)
	model2 := updated2.(TUIModel)

	if len(model2.filesEdited) != 1 {
		t.Fatalf("filesEdited len = %d, want 1", len(model2.filesEdited))
	}
	if model2.filesEdited[0] != "/home/user/util.go" {
		t.Errorf("filesEdited[0] = %q, want %q", model2.filesEdited[0], "/home/user/util.go")
	}

	// Read same file again - should not duplicate
	event3 := ToolStartEvent{ToolName: "Read", Input: map[string]any{"file_path": "/home/user/main.go"}}
	updated3, _ := model2.Update(event3)
	model3 := updated3.(TUIModel)

	if len(model3.filesRead) != 1 {
		t.Errorf("filesRead len = %d, want 1 (no duplicates)", len(model3.filesRead))
	}
}

func TestTUITabSwitchesFocus(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	if m.focusedPane != "tools" {
		t.Errorf("initial focusedPane = %q, want %q", m.focusedPane, "tools")
	}

	// Tab should switch to text
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := updated.(TUIModel)
	if model.focusedPane != "text" {
		t.Errorf("after tab focusedPane = %q, want %q", model.focusedPane, "text")
	}

	// Tab again should switch back to tools
	updated2, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model2 := updated2.(TUIModel)
	if model2.focusedPane != "tools" {
		t.Errorf("after second tab focusedPane = %q, want %q", model2.focusedPane, "tools")
	}
}

func TestTUICardStartedClearsState(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	// Set up some existing state
	m.toolCalls = []toolCallEntry{{toolName: "Read", summary: "old.go", durationMs: 100}}
	m.activeCall = &toolCallEntry{toolName: "Bash", summary: "echo hi", durationMs: -1}
	m.textLines = []string{"old output"}
	m.stats = sessionStats{inputTokens: 5000, outputTokens: 2000, turns: 3}
	m.filesRead = []string{"/old/file.go"}
	m.filesEdited = []string{"/old/edit.go"}

	event := CardStartedEvent{CardID: "c2", CardName: "New task", Branch: "task/c2-new"}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if len(model.toolCalls) != 0 {
		t.Errorf("toolCalls should be cleared, len = %d", len(model.toolCalls))
	}
	if model.activeCall != nil {
		t.Error("activeCall should be nil after CardStartedEvent")
	}
	if len(model.textLines) != 0 {
		t.Errorf("textLines should be cleared, len = %d", len(model.textLines))
	}
	if model.stats.inputTokens != 0 {
		t.Errorf("stats.inputTokens should be 0, got %d", model.stats.inputTokens)
	}
	if model.stats.outputTokens != 0 {
		t.Errorf("stats.outputTokens should be 0, got %d", model.stats.outputTokens)
	}
	if model.stats.turns != 0 {
		t.Errorf("stats.turns should be 0, got %d", model.stats.turns)
	}
	if len(model.filesRead) != 0 {
		t.Errorf("filesRead should be cleared, len = %d", len(model.filesRead))
	}
	if len(model.filesEdited) != 0 {
		t.Errorf("filesEdited should be cleared, len = %d", len(model.filesEdited))
	}
}
