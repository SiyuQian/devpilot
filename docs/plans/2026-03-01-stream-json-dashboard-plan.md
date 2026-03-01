# Stream-JSON Dashboard Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Parse Claude Code's `--output-format stream-json` NDJSON output to show real-time tool calls, token usage, and file changes in the `devpilot run` TUI dashboard.

**Architecture:** Add a `StreamParser` that reads NDJSON lines into typed Go structs. Replace the raw `CardOutputEvent` with richer events (`ToolStartEvent`, `ToolResultEvent`, `TextOutputEvent`, `StatsUpdateEvent`). Redesign the Bubble Tea TUI with multi-panel layout: tool call history, Claude text output, file list, and token stats.

**Tech Stack:** Go, Bubble Tea, Lipgloss, `encoding/json`

**Design doc:** `docs/plans/2026-03-01-stream-json-dashboard-design.md`

---

### Task 1: Stream Parser Types

**Files:**
- Create: `internal/taskrunner/streamparser.go`
- Test: `internal/taskrunner/streamparser_test.go`

**Step 1: Write failing tests for ParseLine**

```go
// streamparser_test.go
package taskrunner

import (
	"testing"
)

func TestParseLineSystem(t *testing.T) {
	line := `{"type":"system","subtype":"init","session_id":"sess1","model":"claude-sonnet-4-5-20250929","tools":["Bash","Read","Edit"]}`
	event, err := ParseLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sys, ok := event.(ClaudeSystemMsg)
	if !ok {
		t.Fatalf("expected ClaudeSystemMsg, got %T", event)
	}
	if sys.SessionID != "sess1" {
		t.Errorf("SessionID = %q, want %q", sys.SessionID, "sess1")
	}
	if sys.Model != "claude-sonnet-4-5-20250929" {
		t.Errorf("Model = %q, want %q", sys.Model, "claude-sonnet-4-5-20250929")
	}
	if len(sys.Tools) != 3 {
		t.Errorf("Tools len = %d, want 3", len(sys.Tools))
	}
}

func TestParseLineAssistantText(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"text","text":"Let me check that."}],"usage":{"input_tokens":100,"output_tokens":20}}}`
	event, err := ParseLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg, ok := event.(ClaudeAssistantMsg)
	if !ok {
		t.Fatalf("expected ClaudeAssistantMsg, got %T", event)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("Content len = %d, want 1", len(msg.Content))
	}
	tb, ok := msg.Content[0].(TextBlock)
	if !ok {
		t.Fatalf("expected TextBlock, got %T", msg.Content[0])
	}
	if tb.Text != "Let me check that." {
		t.Errorf("Text = %q, want %q", tb.Text, "Let me check that.")
	}
	if msg.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", msg.InputTokens)
	}
	if msg.OutputTokens != 20 {
		t.Errorf("OutputTokens = %d, want 20", msg.OutputTokens)
	}
}

func TestParseLineAssistantToolUse(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_1","name":"Read","input":{"file_path":"/tmp/main.go"}}],"usage":{"input_tokens":50,"output_tokens":10}}}`
	event, err := ParseLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg, ok := event.(ClaudeAssistantMsg)
	if !ok {
		t.Fatalf("expected ClaudeAssistantMsg, got %T", event)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("Content len = %d, want 1", len(msg.Content))
	}
	tu, ok := msg.Content[0].(ToolUseBlock)
	if !ok {
		t.Fatalf("expected ToolUseBlock, got %T", msg.Content[0])
	}
	if tu.Name != "Read" {
		t.Errorf("Name = %q, want %q", tu.Name, "Read")
	}
	if tu.ID != "toolu_1" {
		t.Errorf("ID = %q, want %q", tu.ID, "toolu_1")
	}
	fp, ok := tu.Input["file_path"]
	if !ok || fp != "/tmp/main.go" {
		t.Errorf("Input[file_path] = %v, want /tmp/main.go", fp)
	}
}

func TestParseLineUserToolResult(t *testing.T) {
	line := `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"file contents here"}]},"tool_use_result":{"durationMs":12,"truncated":false}}`
	event, err := ParseLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg, ok := event.(ClaudeUserMsg)
	if !ok {
		t.Fatalf("expected ClaudeUserMsg, got %T", event)
	}
	if len(msg.ToolResults) != 1 {
		t.Fatalf("ToolResults len = %d, want 1", len(msg.ToolResults))
	}
	tr := msg.ToolResults[0]
	if tr.ToolUseID != "toolu_1" {
		t.Errorf("ToolUseID = %q, want %q", tr.ToolUseID, "toolu_1")
	}
	if tr.DurationMs != 12 {
		t.Errorf("DurationMs = %d, want 12", tr.DurationMs)
	}
}

func TestParseLineResult(t *testing.T) {
	line := `{"type":"result","subtype":"success","duration_ms":5840,"num_turns":3,"usage":{"input_tokens":1000,"output_tokens":500}}`
	event, err := ParseLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg, ok := event.(ClaudeResultMsg)
	if !ok {
		t.Fatalf("expected ClaudeResultMsg, got %T", event)
	}
	if msg.Turns != 3 {
		t.Errorf("Turns = %d, want 3", msg.Turns)
	}
	if msg.DurationMs != 5840 {
		t.Errorf("DurationMs = %d, want 5840", msg.DurationMs)
	}
	if msg.InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want 1000", msg.InputTokens)
	}
	if msg.OutputTokens != 500 {
		t.Errorf("OutputTokens = %d, want 500", msg.OutputTokens)
	}
}

func TestParseLineInvalidJSON(t *testing.T) {
	line := `not json at all`
	event, err := ParseLine([]byte(line))
	if err != nil {
		t.Fatalf("non-JSON should not error, got: %v", err)
	}
	msg, ok := event.(RawOutputMsg)
	if !ok {
		t.Fatalf("expected RawOutputMsg, got %T", event)
	}
	if msg.Text != "not json at all" {
		t.Errorf("Text = %q, want %q", msg.Text, "not json at all")
	}
}

func TestParseLineUnknownType(t *testing.T) {
	line := `{"type":"stream_event","event":{"type":"content_block_delta"}}`
	event, err := ParseLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Unknown types should be silently ignored
	if event != nil {
		t.Errorf("expected nil for unknown type, got %T", event)
	}
}

func TestParseLineAssistantMixed(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"text","text":"Reading file..."},{"type":"tool_use","id":"toolu_2","name":"Bash","input":{"command":"go test ./..."}}],"usage":{"input_tokens":200,"output_tokens":50}}}`
	event, err := ParseLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg, ok := event.(ClaudeAssistantMsg)
	if !ok {
		t.Fatalf("expected ClaudeAssistantMsg, got %T", event)
	}
	if len(msg.Content) != 2 {
		t.Fatalf("Content len = %d, want 2", len(msg.Content))
	}
	if _, ok := msg.Content[0].(TextBlock); !ok {
		t.Errorf("Content[0] should be TextBlock, got %T", msg.Content[0])
	}
	if tu, ok := msg.Content[1].(ToolUseBlock); !ok {
		t.Errorf("Content[1] should be ToolUseBlock, got %T", msg.Content[1])
	} else if tu.Name != "Bash" {
		t.Errorf("ToolUseBlock.Name = %q, want %q", tu.Name, "Bash")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/taskrunner/ -run TestParseLine -v`
Expected: FAIL — `ParseLine` undefined

**Step 3: Implement StreamParser types and ParseLine**

```go
// streamparser.go
package taskrunner

import "encoding/json"

// ClaudeEvent is the interface for all parsed stream-json events.
type ClaudeEvent interface {
	claudeEventType() string
}

// ContentBlock is the interface for assistant message content blocks.
type ContentBlock interface {
	blockType() string
}

type ClaudeSystemMsg struct {
	SessionID string   `json:"session_id"`
	Model     string   `json:"model"`
	Tools     []string `json:"tools"`
}

func (e ClaudeSystemMsg) claudeEventType() string { return "system" }

type ClaudeAssistantMsg struct {
	Content      []ContentBlock
	InputTokens  int
	OutputTokens int
}

func (e ClaudeAssistantMsg) claudeEventType() string { return "assistant" }

type ClaudeUserMsg struct {
	ToolResults []ToolResult
}

func (e ClaudeUserMsg) claudeEventType() string { return "user" }

type ClaudeResultMsg struct {
	Subtype      string
	Turns        int
	DurationMs   int
	InputTokens  int
	OutputTokens int
}

func (e ClaudeResultMsg) claudeEventType() string { return "result" }

type RawOutputMsg struct {
	Text string
}

func (e RawOutputMsg) claudeEventType() string { return "raw" }

type TextBlock struct {
	Text string
}

func (e TextBlock) blockType() string { return "text" }

type ToolUseBlock struct {
	ID    string
	Name  string
	Input map[string]any
}

func (e ToolUseBlock) blockType() string { return "tool_use" }

type ToolResult struct {
	ToolUseID  string
	Content    string
	DurationMs int
	Truncated  bool
}

// ParseLine parses a single NDJSON line from stream-json output.
// Returns nil for unknown/ignored event types (e.g. stream_event).
// Returns RawOutputMsg for non-JSON input (graceful degradation).
func ParseLine(data []byte) (ClaudeEvent, error) {
	// Quick check: is this JSON?
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return RawOutputMsg{Text: string(data)}, nil
	}

	switch envelope.Type {
	case "system":
		return parseSystem(data)
	case "assistant":
		return parseAssistant(data)
	case "user":
		return parseUser(data)
	case "result":
		return parseResult(data)
	default:
		return nil, nil
	}
}

func parseSystem(data []byte) (ClaudeEvent, error) {
	var raw struct {
		SessionID string   `json:"session_id"`
		Model     string   `json:"model"`
		Tools     []string `json:"tools"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return ClaudeSystemMsg{
		SessionID: raw.SessionID,
		Model:     raw.Model,
		Tools:     raw.Tools,
	}, nil
}

func parseAssistant(data []byte) (ClaudeEvent, error) {
	var raw struct {
		Message struct {
			Content []json.RawMessage `json:"content"`
			Usage   struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var blocks []ContentBlock
	for _, rawBlock := range raw.Message.Content {
		var blockType struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(rawBlock, &blockType); err != nil {
			continue
		}
		switch blockType.Type {
		case "text":
			var tb struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(rawBlock, &tb); err != nil {
				continue
			}
			blocks = append(blocks, TextBlock{Text: tb.Text})
		case "tool_use":
			var tu struct {
				ID    string         `json:"id"`
				Name  string         `json:"name"`
				Input map[string]any `json:"input"`
			}
			if err := json.Unmarshal(rawBlock, &tu); err != nil {
				continue
			}
			blocks = append(blocks, ToolUseBlock{ID: tu.ID, Name: tu.Name, Input: tu.Input})
		}
	}

	return ClaudeAssistantMsg{
		Content:      blocks,
		InputTokens:  raw.Message.Usage.InputTokens,
		OutputTokens: raw.Message.Usage.OutputTokens,
	}, nil
}

func parseUser(data []byte) (ClaudeEvent, error) {
	var raw struct {
		Message struct {
			Content []struct {
				Type      string `json:"type"`
				ToolUseID string `json:"tool_use_id"`
				Content   any    `json:"content"`
			} `json:"content"`
		} `json:"message"`
		ToolUseResult struct {
			DurationMs int  `json:"durationMs"`
			Truncated  bool `json:"truncated"`
		} `json:"tool_use_result"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var results []ToolResult
	for _, c := range raw.Message.Content {
		if c.Type != "tool_result" {
			continue
		}
		contentStr := ""
		if s, ok := c.Content.(string); ok {
			contentStr = s
		}
		results = append(results, ToolResult{
			ToolUseID:  c.ToolUseID,
			Content:    contentStr,
			DurationMs: raw.ToolUseResult.DurationMs,
			Truncated:  raw.ToolUseResult.Truncated,
		})
	}

	return ClaudeUserMsg{ToolResults: results}, nil
}

func parseResult(data []byte) (ClaudeEvent, error) {
	var raw struct {
		Subtype    string `json:"subtype"`
		NumTurns   int    `json:"num_turns"`
		DurationMs int    `json:"duration_ms"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return ClaudeResultMsg{
		Subtype:      raw.Subtype,
		Turns:        raw.NumTurns,
		DurationMs:   raw.DurationMs,
		InputTokens:  raw.Usage.InputTokens,
		OutputTokens: raw.Usage.OutputTokens,
	}, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/taskrunner/ -run TestParseLine -v`
Expected: PASS (all 8 tests)

**Step 5: Commit**

```bash
git add internal/taskrunner/streamparser.go internal/taskrunner/streamparser_test.go
git commit -m "feat: add stream-json NDJSON parser for Claude Code output"
```

---

### Task 2: New Runner Event Types

**Files:**
- Modify: `internal/taskrunner/events.go`
- Modify: `internal/taskrunner/events_test.go`

**Step 1: Write failing test for new event types**

Add to `events_test.go`:

```go
// Add these test cases to the existing TestEventTypes table:
{"ToolStart", ToolStartEvent{ToolName: "Read", Input: map[string]any{"file_path": "/tmp/f.go"}}, "tool_start"},
{"ToolResult", ToolResultEvent{ToolName: "Read", DurationMs: 12, Truncated: false}, "tool_result"},
{"TextOutput", TextOutputEvent{Text: "Let me check..."}, "text_output"},
{"StatsUpdate", StatsUpdateEvent{InputTokens: 100, OutputTokens: 50, Turns: 3}, "stats_update"},
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/taskrunner/ -run TestEventTypes -v`
Expected: FAIL — undefined types

**Step 3: Add new event types to events.go**

Append to `events.go`:

```go
type ToolStartEvent struct {
	ToolName string
	Input    map[string]any
}

func (e ToolStartEvent) eventType() string { return "tool_start" }

type ToolResultEvent struct {
	ToolName   string
	DurationMs int
	Truncated  bool
}

func (e ToolResultEvent) eventType() string { return "tool_result" }

type TextOutputEvent struct {
	Text string
}

func (e TextOutputEvent) eventType() string { return "text_output" }

type StatsUpdateEvent struct {
	InputTokens     int
	OutputTokens    int
	CacheReadTokens int
	Turns           int
}

func (e StatsUpdateEvent) eventType() string { return "stats_update" }
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/taskrunner/ -run TestEventTypes -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/taskrunner/events.go internal/taskrunner/events_test.go
git commit -m "feat: add ToolStart, ToolResult, TextOutput, StatsUpdate event types"
```

---

### Task 3: Executor Stream-JSON Integration

**Files:**
- Modify: `internal/taskrunner/executor.go`
- Modify: `internal/taskrunner/executor_test.go`

This task changes the executor to:
1. Use `--output-format stream-json` as default args
2. Add a new `ClaudeEventHandler` callback type
3. Parse NDJSON lines via `ParseLine` and call the event handler
4. Keep the old `OutputHandler` working for backward compatibility (reviewer)

**Step 1: Write failing tests for the new event handler**

Add to `executor_test.go`:

```go
func TestExecute_ClaudeEventHandler_ParsesJSON(t *testing.T) {
	var mu sync.Mutex
	var events []ClaudeEvent

	handler := func(event ClaudeEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	}

	// Emit two NDJSON lines that look like stream-json output
	jsonLine1 := `{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}],"usage":{"input_tokens":10,"output_tokens":5}}}`
	jsonLine2 := `{"type":"result","subtype":"success","num_turns":1,"duration_ms":100,"usage":{"input_tokens":10,"output_tokens":5}}`

	exec := NewExecutor(
		WithCommand("printf", jsonLine1+"\n"+jsonLine2+"\n"),
		WithClaudeEventHandler(handler),
	)
	_, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if _, ok := events[0].(ClaudeAssistantMsg); !ok {
		t.Errorf("events[0] should be ClaudeAssistantMsg, got %T", events[0])
	}
	if _, ok := events[1].(ClaudeResultMsg); !ok {
		t.Errorf("events[1] should be ClaudeResultMsg, got %T", events[1])
	}
}

func TestExecute_ClaudeEventHandler_SkipsNilEvents(t *testing.T) {
	var mu sync.Mutex
	var events []ClaudeEvent

	handler := func(event ClaudeEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	}

	// stream_event type returns nil from ParseLine — should be skipped
	jsonLine := `{"type":"stream_event","event":{"type":"content_block_delta"}}`

	exec := NewExecutor(
		WithCommand("printf", jsonLine+"\n"),
		WithClaudeEventHandler(handler),
	)
	_, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 0 {
		t.Errorf("expected 0 events (nil skipped), got %d", len(events))
	}
}

func TestExecute_ClaudeEventHandler_NonJSONFallback(t *testing.T) {
	var mu sync.Mutex
	var events []ClaudeEvent

	handler := func(event ClaudeEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	}

	exec := NewExecutor(
		WithCommand("printf", "plain text output\n"),
		WithClaudeEventHandler(handler),
	)
	_, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	raw, ok := events[0].(RawOutputMsg)
	if !ok {
		t.Fatalf("expected RawOutputMsg, got %T", events[0])
	}
	if raw.Text != "plain text output" {
		t.Errorf("Text = %q, want %q", raw.Text, "plain text output")
	}
}

func TestExecute_DefaultArgsIncludeStreamJSON(t *testing.T) {
	exec := NewExecutor()
	// Check that the default args include --output-format stream-json
	found := false
	for _, arg := range exec.args {
		if arg == "stream-json" {
			found = true
		}
	}
	if !found {
		t.Errorf("default args should include stream-json, got %v", exec.args)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/taskrunner/ -run "TestExecute_ClaudeEvent|TestExecute_DefaultArgs" -v`
Expected: FAIL — `WithClaudeEventHandler` undefined

**Step 3: Implement executor changes**

Modify `executor.go`:

1. Add `ClaudeEventHandler` type and `claudeEventHandler` field to `Executor`
2. Add `WithClaudeEventHandler` option
3. Change default args to include `--output-format stream-json`
4. In `scanStream`, when `claudeEventHandler` is set, parse each line via `ParseLine` and call handler
5. When `claudeEventHandler` is set but `outputHandler` is not, set up an internal `outputHandler` that bridges to stream parsing

```go
// Add to executor.go:

// ClaudeEventHandler is called for each parsed stream-json event.
type ClaudeEventHandler func(event ClaudeEvent)

// Add field to Executor struct:
// claudeEventHandler ClaudeEventHandler

// Add option:
func WithClaudeEventHandler(handler ClaudeEventHandler) ExecutorOption {
	return func(e *Executor) {
		e.claudeEventHandler = handler
	}
}

// Change NewExecutor defaults:
// e.args = []string{"-p", "--output-format", "stream-json", "--allowedTools=*"}

// Modify scanStream: when claudeEventHandler is set, parse each line and call handler
// (still call outputHandler if set, and still accumulate to buffer)
```

The key change in `scanStream`:

```go
func (e *Executor) scanStream(pipe io.Reader, stream string, buf *bytes.Buffer) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteByte('\n')
		if e.outputHandler != nil {
			e.outputHandler(OutputLine{Stream: stream, Text: line})
		}
		if e.claudeEventHandler != nil && stream == "stdout" {
			event, err := ParseLine([]byte(line))
			if err == nil && event != nil {
				e.claudeEventHandler(event)
			}
		}
	}
}
```

When `claudeEventHandler` is set but `outputHandler` is not, the executor should still use `runStreaming` (not `runBuffered`). Update the `Run` method:

```go
func (e *Executor) Run(ctx context.Context, prompt string) (*ExecuteResult, error) {
	// ...
	if e.outputHandler == nil && e.claudeEventHandler == nil {
		return e.runBuffered(ctx, cmd)
	}
	return e.runStreaming(ctx, cmd)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/taskrunner/ -run "TestExecute" -v`
Expected: PASS (all old tests + new tests)

**Step 5: Run full test suite to check nothing broke**

Run: `go test ./internal/taskrunner/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/taskrunner/executor.go internal/taskrunner/executor_test.go
git commit -m "feat: add ClaudeEventHandler to executor for stream-json parsing"
```

---

### Task 4: Runner Event Bridge

**Files:**
- Modify: `internal/taskrunner/runner.go`

This task wires the executor's `ClaudeEventHandler` to emit the new runner events (`ToolStartEvent`, `ToolResultEvent`, `TextOutputEvent`, `StatsUpdateEvent`). It also tracks the in-flight tool use IDs to map tool results back to tool names.

**Step 1: Write failing test**

Create `internal/taskrunner/eventbridge_test.go`:

```go
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

	// Should emit StatsUpdateEvent + ToolStartEvent
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

	// First, register a tool start
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
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/taskrunner/ -run TestEventBridge -v`
Expected: FAIL — `newEventBridge` undefined

**Step 3: Implement eventBridge**

Create `internal/taskrunner/eventbridge.go`:

```go
package taskrunner

// eventBridge converts ClaudeEvents from the stream parser into runner Events.
// It tracks in-flight tool use IDs to map results back to tool names.
type eventBridge struct {
	emit       EventHandler
	inflightTools map[string]string // tool_use_id -> tool name
}

func newEventBridge(emit EventHandler) *eventBridge {
	return &eventBridge{
		emit:          emit,
		inflightTools: make(map[string]string),
	}
}

func (b *eventBridge) Handle(ce ClaudeEvent) {
	switch msg := ce.(type) {
	case ClaudeAssistantMsg:
		if msg.InputTokens > 0 || msg.OutputTokens > 0 {
			b.emit(StatsUpdateEvent{
				InputTokens:  msg.InputTokens,
				OutputTokens: msg.OutputTokens,
			})
		}
		for _, block := range msg.Content {
			switch bl := block.(type) {
			case TextBlock:
				if bl.Text != "" {
					b.emit(TextOutputEvent{Text: bl.Text})
				}
			case ToolUseBlock:
				b.inflightTools[bl.ID] = bl.Name
				b.emit(ToolStartEvent{ToolName: bl.Name, Input: bl.Input})
			}
		}
	case ClaudeUserMsg:
		for _, tr := range msg.ToolResults {
			toolName := b.inflightTools[tr.ToolUseID]
			delete(b.inflightTools, tr.ToolUseID)
			b.emit(ToolResultEvent{
				ToolName:   toolName,
				DurationMs: tr.DurationMs,
				Truncated:  tr.Truncated,
			})
		}
	case ClaudeResultMsg:
		b.emit(StatsUpdateEvent{
			InputTokens:  msg.InputTokens,
			OutputTokens: msg.OutputTokens,
			Turns:        msg.Turns,
		})
	case RawOutputMsg:
		if msg.Text != "" {
			b.emit(TextOutputEvent{Text: msg.Text})
		}
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/taskrunner/ -run TestEventBridge -v`
Expected: PASS

**Step 5: Wire eventBridge into runner.go**

Modify `runner.go` `New()` function. Replace the current output handler wiring:

```go
// BEFORE (in New()):
if r.eventHandler != nil {
    execOpts = append(execOpts, WithOutputHandler(func(line OutputLine) {
        r.emit(CardOutputEvent{Line: line})
    }))
}

// AFTER:
if r.eventHandler != nil {
    bridge := newEventBridge(r.eventHandler)
    execOpts = append(execOpts, WithClaudeEventHandler(bridge.Handle))
}
```

**Step 6: Run full test suite**

Run: `go test ./internal/taskrunner/ -v`
Expected: PASS (some TUI tests referencing `CardOutputEvent` will need updating in Task 5)

**Step 7: Commit**

```bash
git add internal/taskrunner/eventbridge.go internal/taskrunner/eventbridge_test.go internal/taskrunner/runner.go
git commit -m "feat: add event bridge to convert stream-json events to runner events"
```

---

### Task 5: Update TUI Model for New Events

**Files:**
- Modify: `internal/taskrunner/tui.go`
- Modify: `internal/taskrunner/tui_test.go`

**Step 1: Write failing tests for new event handling**

Replace `TestTUIUpdateCardOutput` and add new tests in `tui_test.go`:

```go
func TestTUIUpdateToolStart(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	// Initialize viewport
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	event := ToolStartEvent{ToolName: "Read", Input: map[string]any{"file_path": "/tmp/main.go"}}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.activeCall == nil {
		t.Fatal("activeCall should be set")
	}
	if model.activeCall.toolName != "Read" {
		t.Errorf("toolName = %q, want %q", model.activeCall.toolName, "Read")
	}
	if model.activeCall.summary != "/tmp/main.go" {
		t.Errorf("summary = %q, want %q", model.activeCall.summary, "/tmp/main.go")
	}
}

func TestTUIUpdateToolResult(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// First set an active call
	m.activeCall = &toolCallEntry{toolName: "Read", summary: "main.go", durationMs: -1}

	event := ToolResultEvent{ToolName: "Read", DurationMs: 12}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.activeCall != nil {
		t.Error("activeCall should be nil after result")
	}
	if len(model.toolCalls) != 1 {
		t.Fatalf("toolCalls len = %d, want 1", len(model.toolCalls))
	}
	if model.toolCalls[0].durationMs != 12 {
		t.Errorf("durationMs = %d, want 12", model.toolCalls[0].durationMs)
	}
}

func TestTUIUpdateTextOutput(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	event := TextOutputEvent{Text: "Let me check the code."}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if len(model.textLines) != 1 {
		t.Fatalf("textLines len = %d, want 1", len(model.textLines))
	}
	if model.textLines[0] != "Let me check the code." {
		t.Errorf("textLines[0] = %q, want %q", model.textLines[0], "Let me check the code.")
	}
}

func TestTUIUpdateStatsUpdate(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)

	event := StatsUpdateEvent{InputTokens: 1000, OutputTokens: 500, Turns: 3}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if model.stats.inputTokens != 1000 {
		t.Errorf("inputTokens = %d, want 1000", model.stats.inputTokens)
	}
	if model.stats.outputTokens != 500 {
		t.Errorf("outputTokens = %d, want 500", model.stats.outputTokens)
	}
	if model.stats.turns != 3 {
		t.Errorf("turns = %d, want 3", model.stats.turns)
	}
}

func TestTUIUpdateToolStartTracksFiles(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Read event
	updated, _ := m.Update(ToolStartEvent{ToolName: "Read", Input: map[string]any{"file_path": "/tmp/main.go"}})
	model := updated.(TUIModel)
	if len(model.filesRead) != 1 || model.filesRead[0] != "/tmp/main.go" {
		t.Errorf("filesRead = %v, want [/tmp/main.go]", model.filesRead)
	}

	// Edit event
	updated, _ = model.Update(ToolStartEvent{ToolName: "Edit", Input: map[string]any{"file_path": "/tmp/server.go"}})
	model = updated.(TUIModel)
	if len(model.filesEdited) != 1 || model.filesEdited[0] != "/tmp/server.go" {
		t.Errorf("filesEdited = %v, want [/tmp/server.go]", model.filesEdited)
	}

	// Duplicate Read should not add again
	updated, _ = model.Update(ToolStartEvent{ToolName: "Read", Input: map[string]any{"file_path": "/tmp/main.go"}})
	model = updated.(TUIModel)
	if len(model.filesRead) != 1 {
		t.Errorf("filesRead should deduplicate, got %v", model.filesRead)
	}
}

func TestTUITabSwitchesFocus(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	if m.focusedPane != "tools" {
		t.Errorf("default focusedPane = %q, want %q", m.focusedPane, "tools")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := updated.(TUIModel)
	if model.focusedPane != "text" {
		t.Errorf("focusedPane after tab = %q, want %q", model.focusedPane, "text")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(TUIModel)
	if model.focusedPane != "tools" {
		t.Errorf("focusedPane after second tab = %q, want %q", model.focusedPane, "tools")
	}
}

func TestTUICardStartedClearsState(t *testing.T) {
	ch := make(chan Event, 1)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewTUIModel("Test Board", ch, cancel)
	// Pre-populate state from a previous card
	m.toolCalls = []toolCallEntry{{toolName: "Read", summary: "old.go"}}
	m.textLines = []string{"old text"}
	m.filesRead = []string{"old.go"}
	m.filesEdited = []string{"old2.go"}
	m.stats = sessionStats{inputTokens: 999}

	event := CardStartedEvent{CardID: "c2", CardName: "New task", Branch: "task/c2-new"}
	updated, _ := m.Update(event)
	model := updated.(TUIModel)

	if len(model.toolCalls) != 0 {
		t.Errorf("toolCalls should be cleared, got %d", len(model.toolCalls))
	}
	if len(model.textLines) != 0 {
		t.Errorf("textLines should be cleared, got %d", len(model.textLines))
	}
	if len(model.filesRead) != 0 {
		t.Errorf("filesRead should be cleared, got %d", len(model.filesRead))
	}
	if len(model.filesEdited) != 0 {
		t.Errorf("filesEdited should be cleared, got %d", len(model.filesEdited))
	}
	if model.stats.inputTokens != 0 {
		t.Errorf("stats should be cleared, got inputTokens=%d", model.stats.inputTokens)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/taskrunner/ -run "TestTUIUpdate(ToolStart|ToolResult|TextOutput|StatsUpdate|TabSwitches|CardStartedClears)" -v`
Expected: FAIL — undefined fields

**Step 3: Update TUI model struct and event handling**

Modify `tui.go`:

1. Add new fields to `TUIModel`: `toolCalls`, `activeCall`, `textLines`, `stats`, `filesRead`, `filesEdited`, `toolViewport`, `textViewport`, `focusedPane`
2. Remove old `logLines` and `viewport` fields
3. Update `NewTUIModel` to set `focusedPane: "tools"`
4. Update `WindowSizeMsg` handler to initialize both viewports (split available height ~60/40)
5. Remove `CardOutputEvent` handler
6. Add handlers for `ToolStartEvent`, `ToolResultEvent`, `TextOutputEvent`, `StatsUpdateEvent`
7. Update `CardStartedEvent` handler to clear all new state
8. Add `Tab` key handling to switch `focusedPane`
9. Update `j`/`k`/scroll delegation to use focused viewport

Helper function for tool summaries:

```go
func toolSummary(toolName string, input map[string]any) string {
	switch toolName {
	case "Read":
		if fp, ok := input["file_path"].(string); ok {
			return fp
		}
	case "Edit", "Write":
		if fp, ok := input["file_path"].(string); ok {
			return fp
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
```

Helper to extract file path and track files:

```go
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
```

For `StatsUpdateEvent`, update `stats` with cumulative values:

```go
case StatsUpdateEvent:
    m.stats.inputTokens += msg.InputTokens
    m.stats.outputTokens += msg.OutputTokens
    m.stats.cacheReadTokens += msg.CacheReadTokens
    if msg.Turns > 0 {
        m.stats.turns = msg.Turns
    }
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/taskrunner/ -run "TestTUI" -v`
Expected: PASS

**Step 5: Fix any broken old tests**

The old `TestTUIUpdateCardOutput` test references `CardOutputEvent` and `logLines` — remove it. Update `TestKeyUpDown` to use `toolCalls`/`textLines` instead of `logLines`.

**Step 6: Run full test suite**

Run: `go test ./internal/taskrunner/ -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/taskrunner/tui.go internal/taskrunner/tui_test.go
git commit -m "feat: update TUI model for structured tool calls, text, stats, and file tracking"
```

---

### Task 6: TUI View — Multi-Panel Layout

**Files:**
- Modify: `internal/taskrunner/tui_view.go`
- Modify: `internal/taskrunner/tui_view_test.go`

**Step 1: Write failing tests for new view rendering**

Replace/update tests in `tui_view_test.go`:

```go
func TestViewMinimumSize(t *testing.T) {
	m := TUIModel{ready: true, width: 60, height: 15, phase: "idle"}
	output := m.View()
	if !strings.Contains(output, "too small") {
		t.Errorf("expected 'too small' for 60x15, got %q", output)
	}
	// 80x20 should work
	m.width = 80
	m.height = 20
	m.boardName = "Test"
	output = m.View()
	if strings.Contains(output, "too small") {
		t.Errorf("80x20 should not be too small")
	}
}

func TestViewHeaderShowsTokens(t *testing.T) {
	m := TUIModel{
		ready:     true,
		width:     120,
		height:    30,
		phase:     "running",
		boardName: "Dev Board",
		stats:     sessionStats{inputTokens: 12500, outputTokens: 3200, turns: 7},
	}
	output := renderHeader(m)
	if !strings.Contains(output, "12.5k") {
		t.Errorf("expected '12.5k' in header, got %q", output)
	}
	if !strings.Contains(output, "3.2k") {
		t.Errorf("expected '3.2k' in header, got %q", output)
	}
	if !strings.Contains(output, "T:7") {
		t.Errorf("expected 'T:7' in header, got %q", output)
	}
}

func TestViewActiveCardShowsCurrentTool(t *testing.T) {
	m := TUIModel{
		ready: true,
		width: 120,
		height: 30,
		phase: "running",
		activeCard: &cardState{
			name: "Fix bug", branch: "task/c1-fix", status: "running",
			started: time.Now().Add(-2 * time.Minute),
		},
		activeCall: &toolCallEntry{toolName: "Edit", summary: "server.go"},
	}
	output := renderActiveTask(m)
	if !strings.Contains(output, "Edit") {
		t.Errorf("expected current tool 'Edit' in active card, got %q", output)
	}
	if !strings.Contains(output, "server.go") {
		t.Errorf("expected 'server.go' in active card, got %q", output)
	}
}

func TestViewToolCallsPanel(t *testing.T) {
	m := TUIModel{
		ready: true,
		width: 120,
		height: 30,
		toolCalls: []toolCallEntry{
			{toolName: "Read", summary: "main.go", durationMs: 12},
			{toolName: "Bash", summary: "go test ./...", durationMs: 3400},
		},
		activeCall: &toolCallEntry{toolName: "Edit", summary: "server.go", durationMs: -1},
	}
	output := renderToolCallsPanel(m)
	if !strings.Contains(output, "Read") {
		t.Errorf("expected 'Read' in tool calls, got %q", output)
	}
	if !strings.Contains(output, "12ms") {
		t.Errorf("expected '12ms' in tool calls, got %q", output)
	}
	if !strings.Contains(output, "3.4s") || !strings.Contains(output, "3400ms") {
		// Accept either format
	}
	if !strings.Contains(output, "Edit") {
		t.Errorf("expected active 'Edit' in tool calls, got %q", output)
	}
}

func TestViewFilesPanel(t *testing.T) {
	m := TUIModel{
		ready:       true,
		width:       120,
		height:      30,
		filesRead:   []string{"/project/main.go", "/project/server.go"},
		filesEdited: []string{"/project/middleware.go"},
	}
	output := renderFilesPanel(m)
	if !strings.Contains(output, "main.go") {
		t.Errorf("expected 'main.go' in files panel, got %q", output)
	}
	if !strings.Contains(output, "middleware.go") {
		t.Errorf("expected 'middleware.go' in files panel, got %q", output)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/taskrunner/ -run "TestView(MinimumSize|HeaderShowsTokens|ActiveCardShowsCurrentTool|ToolCallsPanel|FilesPanel)" -v`
Expected: FAIL

**Step 3: Implement new rendering functions**

Rewrite `tui_view.go`:

- `renderView()` — new layout with 5 sections
- `renderHeader()` — add token display (`↑12.5k ↓3.2k T:7`)
- `renderStatusAndActive()` — add current tool indicator (⚡) to active card
- `renderToolsAndFiles()` — new panel: left side scrollable tool call list, right side file list
- `renderTextPane()` — new panel: scrollable Claude text output
- `renderFooter()` — unchanged

Token formatting helper:

```go
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
```

Duration formatting helper:

```go
func formatDuration(ms int) string {
	if ms < 0 {
		return "..."
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000.0)
}
```

File path shortening (show only basename or last 2 path components):

```go
func shortenPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}
	return strings.Join(parts[len(parts)-2:], "/")
}
```

Update minimum terminal size check from 60x15 to 80x20.

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/taskrunner/ -run "TestView" -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `go test ./internal/taskrunner/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/taskrunner/tui_view.go internal/taskrunner/tui_view_test.go
git commit -m "feat: multi-panel TUI layout with tool calls, files, and token stats"
```

---

### Task 7: Remove Old CardOutputEvent

**Files:**
- Modify: `internal/taskrunner/events.go`
- Modify: `internal/taskrunner/events_test.go`

**Step 1: Remove CardOutputEvent from events.go**

Delete the `CardOutputEvent` struct and its `eventType()` method.

**Step 2: Remove from events_test.go**

Remove the `CardOutputEvent` test case from `TestEventTypes`.

**Step 3: Search for any remaining references**

Run: `grep -r "CardOutputEvent" internal/`

Fix any remaining references. The runner no longer emits this event (replaced by eventBridge in Task 4). The TUI no longer handles it (replaced in Task 5).

**Step 4: Run full test suite**

Run: `go test ./internal/taskrunner/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/taskrunner/events.go internal/taskrunner/events_test.go
git commit -m "refactor: remove deprecated CardOutputEvent, replaced by structured events"
```

---

### Task 8: Plain Text Mode Update

**Files:**
- Modify: `internal/taskrunner/runner.go`
- Modify: `internal/taskrunner/commands.go`

**Step 1: Update plain text mode to use eventBridge**

In `commands.go`, `runPlainText()` currently creates a runner with no event handler. Update it to optionally use a simple text-based event handler that prints formatted lines to stdout:

```go
func runPlainText(cfg Config, trelloClient *trello.Client) {
	handler := func(e Event) {
		switch ev := e.(type) {
		case ToolStartEvent:
			log.Printf("[tool] %s %s ...", ev.ToolName, toolSummary(ev.ToolName, ev.Input))
		case ToolResultEvent:
			log.Printf("[tool] %s done (%s)", ev.ToolName, formatDuration(ev.DurationMs))
		case TextOutputEvent:
			log.Printf("[text] %s", truncate(ev.Text, 120))
		case StatsUpdateEvent:
			if ev.Turns > 0 {
				log.Printf("[stats] ↑%s ↓%s turns:%d", formatTokens(ev.InputTokens), formatTokens(ev.OutputTokens), ev.Turns)
			}
		case CardStartedEvent:
			log.Printf("[card] Started: %q on branch %s", ev.CardName, ev.Branch)
		case CardDoneEvent:
			log.Printf("[card] Done: %q (%s) PR: %s", ev.CardName, ev.Duration, ev.PRURL)
		case CardFailedEvent:
			log.Printf("[card] Failed: %q — %s", ev.CardName, ev.ErrMsg)
		}
	}

	r := New(cfg, trelloClient, WithEventHandler(handler))
	// ... rest unchanged
}
```

Note: `toolSummary` and `formatDuration` and `formatTokens` need to be accessible from `commands.go`. They are defined in `tui_view.go` — since they're in the same package this works.

**Step 2: Test manually**

Run: `go build ./cmd/devpilot && ./bin/devpilot run --board devpilot --no-tui --once --dry-run`
Expected: Compiles and runs without error. Dry run prints formatted actions.

**Step 3: Run full test suite**

Run: `go test ./... -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/taskrunner/commands.go
git commit -m "feat: add structured plain text output for --no-tui mode"
```

---

### Task 9: Integration Smoke Test

**Files:**
- No new files — manual verification

**Step 1: Build**

Run: `make build`
Expected: Compiles successfully

**Step 2: Run full test suite**

Run: `make test`
Expected: All tests pass

**Step 3: Manual TUI verification (optional)**

If a Trello board is available:

Run: `./bin/devpilot run --board devpilot --once`

Verify:
- Header shows token counts and turn count
- Tool calls appear with ✓/⚡ icons and durations
- Files panel shows R/E prefixed file paths
- Claude text output appears in the text pane
- Tab switches scroll focus between panels
- `q` quits cleanly

**Step 4: Commit any final fixes**

```bash
git add -A
git commit -m "fix: integration polish for stream-json dashboard"
```
