package taskrunner

// eventBridge converts ClaudeEvents from the stream parser into runner Events.
// It tracks in-flight tool use IDs to map results back to tool names.
type eventBridge struct {
	emit          EventHandler
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
