package taskrunner

import (
	"testing"
)

func TestParseLineSystemMsg(t *testing.T) {
	input := []byte(`{"type":"system","session_id":"abc123","model":"claude-sonnet-4-20250514","tools":["Bash","Read","Write","Glob","Grep"]}`)

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := event.(ClaudeSystemMsg)
	if !ok {
		t.Fatalf("expected ClaudeSystemMsg, got %T", event)
	}

	if msg.SessionID != "abc123" {
		t.Errorf("SessionID = %q, want %q", msg.SessionID, "abc123")
	}
	if msg.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want %q", msg.Model, "claude-sonnet-4-20250514")
	}
	if len(msg.Tools) != 5 {
		t.Errorf("len(Tools) = %d, want 5", len(msg.Tools))
	}
	if msg.Tools[0] != "Bash" {
		t.Errorf("Tools[0] = %q, want %q", msg.Tools[0], "Bash")
	}
}

func TestParseLineAssistantTextBlock(t *testing.T) {
	input := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Hello, world!"}],"usage":{"input_tokens":100,"output_tokens":50}}}`)

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := event.(ClaudeAssistantMsg)
	if !ok {
		t.Fatalf("expected ClaudeAssistantMsg, got %T", event)
	}

	if len(msg.Content) != 1 {
		t.Fatalf("len(Content) = %d, want 1", len(msg.Content))
	}

	tb, ok := msg.Content[0].(TextBlock)
	if !ok {
		t.Fatalf("expected TextBlock, got %T", msg.Content[0])
	}
	if tb.Text != "Hello, world!" {
		t.Errorf("Text = %q, want %q", tb.Text, "Hello, world!")
	}

	if msg.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", msg.InputTokens)
	}
	if msg.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", msg.OutputTokens)
	}
}

func TestParseLineAssistantToolUseBlock(t *testing.T) {
	input := []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_123","name":"Bash","input":{"command":"ls -la"}}],"usage":{"input_tokens":200,"output_tokens":30}}}`)

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := event.(ClaudeAssistantMsg)
	if !ok {
		t.Fatalf("expected ClaudeAssistantMsg, got %T", event)
	}

	if len(msg.Content) != 1 {
		t.Fatalf("len(Content) = %d, want 1", len(msg.Content))
	}

	tub, ok := msg.Content[0].(ToolUseBlock)
	if !ok {
		t.Fatalf("expected ToolUseBlock, got %T", msg.Content[0])
	}
	if tub.ID != "tu_123" {
		t.Errorf("ID = %q, want %q", tub.ID, "tu_123")
	}
	if tub.Name != "Bash" {
		t.Errorf("Name = %q, want %q", tub.Name, "Bash")
	}
	if cmd, ok := tub.Input["command"]; !ok || cmd != "ls -la" {
		t.Errorf("Input[command] = %v, want %q", tub.Input["command"], "ls -la")
	}
}

func TestParseLineAssistantMixedBlocks(t *testing.T) {
	input := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Let me check."},{"type":"tool_use","id":"tu_456","name":"Read","input":{"path":"/tmp/file.go"}}],"usage":{"input_tokens":150,"output_tokens":75}}}`)

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := event.(ClaudeAssistantMsg)
	if !ok {
		t.Fatalf("expected ClaudeAssistantMsg, got %T", event)
	}

	if len(msg.Content) != 2 {
		t.Fatalf("len(Content) = %d, want 2", len(msg.Content))
	}

	if _, ok := msg.Content[0].(TextBlock); !ok {
		t.Errorf("Content[0] expected TextBlock, got %T", msg.Content[0])
	}
	if _, ok := msg.Content[1].(ToolUseBlock); !ok {
		t.Errorf("Content[1] expected ToolUseBlock, got %T", msg.Content[1])
	}

	tb := msg.Content[0].(TextBlock)
	if tb.Text != "Let me check." {
		t.Errorf("TextBlock.Text = %q, want %q", tb.Text, "Let me check.")
	}

	tub := msg.Content[1].(ToolUseBlock)
	if tub.Name != "Read" {
		t.Errorf("ToolUseBlock.Name = %q, want %q", tub.Name, "Read")
	}
}

func TestParseLineUserToolResult(t *testing.T) {
	input := []byte(`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tu_123","content":"file contents here"}]},"tool_use_result":{"durationMs":250,"truncated":false}}`)

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := event.(ClaudeUserMsg)
	if !ok {
		t.Fatalf("expected ClaudeUserMsg, got %T", event)
	}

	if len(msg.ToolResults) != 1 {
		t.Fatalf("len(ToolResults) = %d, want 1", len(msg.ToolResults))
	}

	tr := msg.ToolResults[0]
	if tr.ToolUseID != "tu_123" {
		t.Errorf("ToolUseID = %q, want %q", tr.ToolUseID, "tu_123")
	}
	if tr.Content != "file contents here" {
		t.Errorf("Content = %q, want %q", tr.Content, "file contents here")
	}
	if tr.DurationMs != 250 {
		t.Errorf("DurationMs = %d, want 250", tr.DurationMs)
	}
	if tr.Truncated != false {
		t.Errorf("Truncated = %v, want false", tr.Truncated)
	}
}

func TestParseLineResultMsg(t *testing.T) {
	input := []byte(`{"type":"result","subtype":"success","num_turns":5,"duration_ms":12345,"usage":{"input_tokens":5000,"output_tokens":2000}}`)

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := event.(ClaudeResultMsg)
	if !ok {
		t.Fatalf("expected ClaudeResultMsg, got %T", event)
	}

	if msg.Subtype != "success" {
		t.Errorf("Subtype = %q, want %q", msg.Subtype, "success")
	}
	if msg.Turns != 5 {
		t.Errorf("Turns = %d, want 5", msg.Turns)
	}
	if msg.DurationMs != 12345 {
		t.Errorf("DurationMs = %d, want 12345", msg.DurationMs)
	}
	if msg.InputTokens != 5000 {
		t.Errorf("InputTokens = %d, want 5000", msg.InputTokens)
	}
	if msg.OutputTokens != 2000 {
		t.Errorf("OutputTokens = %d, want 2000", msg.OutputTokens)
	}
}

func TestParseLineNonJSON(t *testing.T) {
	input := []byte("this is not json at all")

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := event.(RawOutputMsg)
	if !ok {
		t.Fatalf("expected RawOutputMsg, got %T", event)
	}

	if msg.Text != "this is not json at all" {
		t.Errorf("Text = %q, want %q", msg.Text, "this is not json at all")
	}
}

func TestParseLineUnknownType(t *testing.T) {
	input := []byte(`{"type":"stream_event","data":"something"}`)

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event != nil {
		t.Errorf("expected nil event for unknown type, got %T", event)
	}
}

func TestParseLineEmptyInput(t *testing.T) {
	event, err := ParseLine([]byte(""))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := event.(RawOutputMsg)
	if !ok {
		t.Fatalf("expected RawOutputMsg, got %T", event)
	}
	if msg.Text != "" {
		t.Errorf("Text = %q, want empty string", msg.Text)
	}
}

func TestParseLineAssistantEmptyContent(t *testing.T) {
	input := []byte(`{"type":"assistant","message":{"content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`)

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := event.(ClaudeAssistantMsg)
	if !ok {
		t.Fatalf("expected ClaudeAssistantMsg, got %T", event)
	}

	if len(msg.Content) != 0 {
		t.Errorf("len(Content) = %d, want 0", len(msg.Content))
	}
}

func TestParseLineUserMultipleToolResults(t *testing.T) {
	input := []byte(`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tu_1","content":"result1"},{"type":"tool_result","tool_use_id":"tu_2","content":"result2"}]},"tool_use_result":{"durationMs":500,"truncated":true}}`)

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := event.(ClaudeUserMsg)
	if !ok {
		t.Fatalf("expected ClaudeUserMsg, got %T", event)
	}

	if len(msg.ToolResults) != 2 {
		t.Fatalf("len(ToolResults) = %d, want 2", len(msg.ToolResults))
	}

	if msg.ToolResults[0].ToolUseID != "tu_1" {
		t.Errorf("ToolResults[0].ToolUseID = %q, want %q", msg.ToolResults[0].ToolUseID, "tu_1")
	}
	if msg.ToolResults[1].ToolUseID != "tu_2" {
		t.Errorf("ToolResults[1].ToolUseID = %q, want %q", msg.ToolResults[1].ToolUseID, "tu_2")
	}
}

func TestParseLineContentBlockInterface(t *testing.T) {
	// Verify that TextBlock and ToolUseBlock both satisfy ContentBlock.
	var _ ContentBlock = TextBlock{}
	var _ ContentBlock = ToolUseBlock{}
}

func TestParseLineClaudeEventInterface(t *testing.T) {
	// Verify that all message types satisfy ClaudeEvent.
	var _ ClaudeEvent = ClaudeSystemMsg{}
	var _ ClaudeEvent = ClaudeAssistantMsg{}
	var _ ClaudeEvent = ClaudeUserMsg{}
	var _ ClaudeEvent = ClaudeResultMsg{}
	var _ ClaudeEvent = RawOutputMsg{}
}

func TestParseLineSystemEmptyTools(t *testing.T) {
	input := []byte(`{"type":"system","session_id":"sess_0","model":"claude-opus-4-20250514","tools":[]}`)

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := event.(ClaudeSystemMsg)
	if !ok {
		t.Fatalf("expected ClaudeSystemMsg, got %T", event)
	}
	if len(msg.Tools) != 0 {
		t.Errorf("len(Tools) = %d, want 0", len(msg.Tools))
	}
}

func TestParseLineToolUseInputNestedObject(t *testing.T) {
	input := []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_789","name":"Write","input":{"path":"/tmp/out.json","content":"{\"key\":\"value\"}"}}],"usage":{"input_tokens":300,"output_tokens":100}}}`)

	event, err := ParseLine(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := event.(ClaudeAssistantMsg)
	tub := msg.Content[0].(ToolUseBlock)

	if tub.Input["path"] != "/tmp/out.json" {
		t.Errorf("Input[path] = %v, want %q", tub.Input["path"], "/tmp/out.json")
	}
}
