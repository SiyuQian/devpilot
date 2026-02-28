package taskrunner

import (
	"encoding/json"
)

// ClaudeEvent is the interface implemented by all parsed stream-json events.
type ClaudeEvent interface {
	claudeEvent()
}

// ContentBlock is the interface for content blocks inside assistant messages.
type ContentBlock interface {
	contentBlock()
}

// --- Message types ---

// ClaudeSystemMsg represents a "system" event emitted at session start.
type ClaudeSystemMsg struct {
	SessionID string   `json:"session_id"`
	Model     string   `json:"model"`
	Tools     []string `json:"tools"`
}

func (ClaudeSystemMsg) claudeEvent() {}

// ClaudeAssistantMsg represents an "assistant" event with content blocks and token usage.
type ClaudeAssistantMsg struct {
	Content      []ContentBlock
	InputTokens  int
	OutputTokens int
}

func (ClaudeAssistantMsg) claudeEvent() {}

// ClaudeUserMsg represents a "user" event containing tool results.
type ClaudeUserMsg struct {
	ToolResults []ToolResult
}

func (ClaudeUserMsg) claudeEvent() {}

// ClaudeResultMsg represents a "result" event with final execution stats.
type ClaudeResultMsg struct {
	Subtype      string
	Turns        int
	DurationMs   int
	InputTokens  int
	OutputTokens int
}

func (ClaudeResultMsg) claudeEvent() {}

// RawOutputMsg is a fallback for non-JSON lines.
type RawOutputMsg struct {
	Text string
}

func (RawOutputMsg) claudeEvent() {}

// --- Content block types ---

// TextBlock is a text content block from an assistant message.
type TextBlock struct {
	Text string
}

func (TextBlock) contentBlock() {}

// ToolUseBlock is a tool_use content block from an assistant message.
type ToolUseBlock struct {
	ID    string
	Name  string
	Input map[string]any
}

func (ToolUseBlock) contentBlock() {}

// ToolResult represents a single tool result from a user message.
type ToolResult struct {
	ToolUseID  string
	Content    string
	DurationMs int
	Truncated  bool
}

// --- Raw JSON structures for parsing ---

type rawEnvelope struct {
	Type string `json:"type"`
}

type rawSystemMsg struct {
	SessionID string   `json:"session_id"`
	Model     string   `json:"model"`
	Tools     []string `json:"tools"`
}

type rawAssistantMsg struct {
	Message struct {
		Content []json.RawMessage `json:"content"`
		Usage   struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

type rawContentBlock struct {
	Type  string         `json:"type"`
	Text  string         `json:"text"`
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

type rawUserMsg struct {
	Message struct {
		Content []struct {
			Type      string `json:"type"`
			ToolUseID string `json:"tool_use_id"`
			Content   string `json:"content"`
		} `json:"content"`
	} `json:"message"`
	ToolUseResult struct {
		DurationMs int  `json:"durationMs"`
		Truncated  bool `json:"truncated"`
	} `json:"tool_use_result"`
}

type rawResultMsg struct {
	Subtype    string `json:"subtype"`
	NumTurns   int    `json:"num_turns"`
	DurationMs int    `json:"duration_ms"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ParseLine parses a single line of stream-json output from Claude Code.
// It returns the appropriate ClaudeEvent for recognized types, RawOutputMsg
// for non-JSON input (with no error), and (nil, nil) for unknown JSON types.
func ParseLine(data []byte) (ClaudeEvent, error) {
	// Try to determine the type field.
	var envelope rawEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		// Not valid JSON — return as raw output.
		return RawOutputMsg{Text: string(data)}, nil
	}

	switch envelope.Type {
	case "system":
		return parseSystemMsg(data)
	case "assistant":
		return parseAssistantMsg(data)
	case "user":
		return parseUserMsg(data)
	case "result":
		return parseResultMsg(data)
	default:
		// Unknown type (e.g., stream_event) — skip silently.
		return nil, nil
	}
}

func parseSystemMsg(data []byte) (ClaudeEvent, error) {
	var raw rawSystemMsg
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if raw.Tools == nil {
		raw.Tools = []string{}
	}
	return ClaudeSystemMsg{
		SessionID: raw.SessionID,
		Model:     raw.Model,
		Tools:     raw.Tools,
	}, nil
}

func parseAssistantMsg(data []byte) (ClaudeEvent, error) {
	var raw rawAssistantMsg
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var blocks []ContentBlock
	for _, rawBlock := range raw.Message.Content {
		var cb rawContentBlock
		if err := json.Unmarshal(rawBlock, &cb); err != nil {
			continue
		}
		switch cb.Type {
		case "text":
			blocks = append(blocks, TextBlock{Text: cb.Text})
		case "tool_use":
			blocks = append(blocks, ToolUseBlock{
				ID:    cb.ID,
				Name:  cb.Name,
				Input: cb.Input,
			})
		}
	}

	if blocks == nil {
		blocks = []ContentBlock{}
	}

	return ClaudeAssistantMsg{
		Content:      blocks,
		InputTokens:  raw.Message.Usage.InputTokens,
		OutputTokens: raw.Message.Usage.OutputTokens,
	}, nil
}

func parseUserMsg(data []byte) (ClaudeEvent, error) {
	var raw rawUserMsg
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var results []ToolResult
	for _, c := range raw.Message.Content {
		if c.Type == "tool_result" {
			results = append(results, ToolResult{
				ToolUseID:  c.ToolUseID,
				Content:    c.Content,
				DurationMs: raw.ToolUseResult.DurationMs,
				Truncated:  raw.ToolUseResult.Truncated,
			})
		}
	}

	if results == nil {
		results = []ToolResult{}
	}

	return ClaudeUserMsg{ToolResults: results}, nil
}

func parseResultMsg(data []byte) (ClaudeEvent, error) {
	var raw rawResultMsg
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
