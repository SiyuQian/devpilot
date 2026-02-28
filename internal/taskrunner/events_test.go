package taskrunner

import (
	"testing"
	"time"
)

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		wantType string
	}{
		{"RunnerStarted", RunnerStartedEvent{BoardName: "B", BoardID: "1", Lists: map[string]string{"Ready": "r1"}}, "runner_started"},
		{"Polling", PollingEvent{}, "polling"},
		{"NoTasks", NoTasksEvent{NextPoll: 5 * time.Second}, "no_tasks"},
		{"CardStarted", CardStartedEvent{CardID: "c1", CardName: "Fix bug", Branch: "task/c1-fix"}, "card_started"},
		{"CardDone", CardDoneEvent{CardID: "c1", CardName: "Fix bug", PRURL: "http://pr", Duration: time.Minute}, "card_done"},
		{"CardFailed", CardFailedEvent{CardID: "c1", CardName: "Fix bug", ErrMsg: "oops", Duration: time.Minute}, "card_failed"},
		{"ReviewStarted", ReviewStartedEvent{PRURL: "http://pr"}, "review_started"},
		{"ReviewDone", ReviewDoneEvent{PRURL: "http://pr", ExitCode: 0}, "review_done"},
		{"RunnerStopped", RunnerStoppedEvent{}, "runner_stopped"},
		{"RunnerError", RunnerErrorEvent{Err: nil}, "runner_error"},
		{"ToolStart", ToolStartEvent{ToolName: "Read", Input: map[string]any{"file_path": "/tmp/f.go"}}, "tool_start"},
		{"ToolResult", ToolResultEvent{ToolName: "Read", DurationMs: 12, Truncated: false}, "tool_result"},
		{"TextOutput", TextOutputEvent{Text: "Let me check..."}, "text_output"},
		{"StatsUpdate", StatsUpdateEvent{InputTokens: 100, OutputTokens: 50, Turns: 3}, "stats_update"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.event.eventType()
			if got != tt.wantType {
				t.Errorf("eventType() = %q, want %q", got, tt.wantType)
			}
		})
	}
}
