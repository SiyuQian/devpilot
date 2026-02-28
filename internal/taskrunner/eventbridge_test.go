package taskrunner

import (
	"sync"
	"testing"
)

func TestEventBridge_ToolUseEmitsToolStart(t *testing.T) {
	var mu sync.Mutex
	var events []Event

	bridge := newEventBridge(func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	})

	bridge.Handle(ClaudeAssistantMsg{
		Content: []ContentBlock{
			ToolUseBlock{ID: "t1", Name: "Read", Input: map[string]any{"file_path": "/tmp/main.go"}},
		},
		InputTokens: 100, OutputTokens: 20,
	})

	mu.Lock()
	defer mu.Unlock()

	var toolStarts []ToolStartEvent
	var statsUpdates []StatsUpdateEvent
	for _, e := range events {
		switch ev := e.(type) {
		case ToolStartEvent:
			toolStarts = append(toolStarts, ev)
		case StatsUpdateEvent:
			statsUpdates = append(statsUpdates, ev)
		}
	}
	if len(toolStarts) != 1 {
		t.Fatalf("expected 1 ToolStartEvent, got %d", len(toolStarts))
	}
	if toolStarts[0].ToolName != "Read" {
		t.Errorf("ToolName = %q, want %q", toolStarts[0].ToolName, "Read")
	}
	if len(statsUpdates) != 1 {
		t.Fatalf("expected 1 StatsUpdateEvent, got %d", len(statsUpdates))
	}
	if statsUpdates[0].InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", statsUpdates[0].InputTokens)
	}
}

func TestEventBridge_TextEmitsTextOutput(t *testing.T) {
	var mu sync.Mutex
	var events []Event

	bridge := newEventBridge(func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	})

	bridge.Handle(ClaudeAssistantMsg{
		Content: []ContentBlock{
			TextBlock{Text: "Let me check the file."},
		},
	})

	mu.Lock()
	defer mu.Unlock()

	var textOutputs []TextOutputEvent
	for _, e := range events {
		if ev, ok := e.(TextOutputEvent); ok {
			textOutputs = append(textOutputs, ev)
		}
	}
	if len(textOutputs) != 1 {
		t.Fatalf("expected 1 TextOutputEvent, got %d", len(textOutputs))
	}
	if textOutputs[0].Text != "Let me check the file." {
		t.Errorf("Text = %q, want %q", textOutputs[0].Text, "Let me check the file.")
	}
}

func TestEventBridge_ToolResultEmitsToolResult(t *testing.T) {
	var mu sync.Mutex
	var events []Event

	bridge := newEventBridge(func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	})

	// Register a tool start first
	bridge.Handle(ClaudeAssistantMsg{
		Content: []ContentBlock{
			ToolUseBlock{ID: "t1", Name: "Bash", Input: map[string]any{"command": "go test"}},
		},
	})

	// Then send the result
	bridge.Handle(ClaudeUserMsg{
		ToolResults: []ToolResult{
			{ToolUseID: "t1", DurationMs: 3400},
		},
	})

	mu.Lock()
	defer mu.Unlock()

	var toolResults []ToolResultEvent
	for _, e := range events {
		if ev, ok := e.(ToolResultEvent); ok {
			toolResults = append(toolResults, ev)
		}
	}
	if len(toolResults) != 1 {
		t.Fatalf("expected 1 ToolResultEvent, got %d", len(toolResults))
	}
	if toolResults[0].ToolName != "Bash" {
		t.Errorf("ToolName = %q, want %q", toolResults[0].ToolName, "Bash")
	}
	if toolResults[0].DurationMs != 3400 {
		t.Errorf("DurationMs = %d, want 3400", toolResults[0].DurationMs)
	}
}

func TestEventBridge_ResultEmitsFinalStats(t *testing.T) {
	var mu sync.Mutex
	var events []Event

	bridge := newEventBridge(func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	})

	bridge.Handle(ClaudeResultMsg{
		Turns: 7, DurationMs: 5000, InputTokens: 12000, OutputTokens: 3000,
	})

	mu.Lock()
	defer mu.Unlock()

	var statsUpdates []StatsUpdateEvent
	for _, e := range events {
		if ev, ok := e.(StatsUpdateEvent); ok {
			statsUpdates = append(statsUpdates, ev)
		}
	}
	if len(statsUpdates) != 1 {
		t.Fatalf("expected 1 StatsUpdateEvent, got %d", len(statsUpdates))
	}
	if statsUpdates[0].Turns != 7 {
		t.Errorf("Turns = %d, want 7", statsUpdates[0].Turns)
	}
	if statsUpdates[0].InputTokens != 12000 {
		t.Errorf("InputTokens = %d, want 12000", statsUpdates[0].InputTokens)
	}
}

func TestEventBridge_RawOutputEmitsTextOutput(t *testing.T) {
	var mu sync.Mutex
	var events []Event

	bridge := newEventBridge(func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	})

	bridge.Handle(RawOutputMsg{Text: "some plain text"})

	mu.Lock()
	defer mu.Unlock()

	var textOutputs []TextOutputEvent
	for _, e := range events {
		if ev, ok := e.(TextOutputEvent); ok {
			textOutputs = append(textOutputs, ev)
		}
	}
	if len(textOutputs) != 1 {
		t.Fatalf("expected 1 TextOutputEvent, got %d", len(textOutputs))
	}
	if textOutputs[0].Text != "some plain text" {
		t.Errorf("Text = %q, want %q", textOutputs[0].Text, "some plain text")
	}
}

func TestEventBridge_EmptyTextIgnored(t *testing.T) {
	var mu sync.Mutex
	var events []Event

	bridge := newEventBridge(func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
	})

	bridge.Handle(ClaudeAssistantMsg{
		Content: []ContentBlock{TextBlock{Text: ""}},
	})
	bridge.Handle(RawOutputMsg{Text: ""})

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 0 {
		t.Errorf("expected 0 events for empty text, got %d", len(events))
	}
}
