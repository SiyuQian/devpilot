package taskrunner

import "time"

// EventHandler receives runner lifecycle events.
type EventHandler func(Event)

// Event is the interface all runner events implement.
type Event interface {
	eventType() string
}

// RunnerStartedEvent is emitted when the runner initializes and connects to a board.
type RunnerStartedEvent struct {
	BoardName string
	BoardID   string
	Lists     map[string]string // list name -> ID
}

func (e RunnerStartedEvent) eventType() string { return "runner_started" }

// PollingEvent is emitted when the runner begins polling for new tasks.
type PollingEvent struct{}

func (e PollingEvent) eventType() string { return "polling" }

// NoTasksEvent is emitted when a poll cycle finds no ready tasks.
type NoTasksEvent struct {
	NextPoll time.Duration
}

func (e NoTasksEvent) eventType() string { return "no_tasks" }

// CardStartedEvent is emitted when the runner begins processing a task card.
type CardStartedEvent struct {
	CardID   string
	CardName string
	Branch   string
}

func (e CardStartedEvent) eventType() string { return "card_started" }

// CardDoneEvent is emitted when a task card completes successfully with a PR.
type CardDoneEvent struct {
	CardID   string
	CardName string
	PRURL    string
	Duration time.Duration
}

func (e CardDoneEvent) eventType() string { return "card_done" }

// CardFailedEvent is emitted when a task card fails during processing.
type CardFailedEvent struct {
	CardID   string
	CardName string
	ErrMsg   string
	Duration time.Duration
}

func (e CardFailedEvent) eventType() string { return "card_failed" }

// ReviewStartedEvent is emitted when an automated code review begins for a PR.
type ReviewStartedEvent struct {
	PRURL string
}

func (e ReviewStartedEvent) eventType() string { return "review_started" }

// ReviewDoneEvent is emitted when an automated code review completes.
type ReviewDoneEvent struct {
	PRURL    string
	ExitCode int
}

func (e ReviewDoneEvent) eventType() string { return "review_done" }

// FixStartedEvent is emitted when a fix attempt begins for review comments.
type FixStartedEvent struct {
	PRURL   string
	Attempt int
}

func (e FixStartedEvent) eventType() string { return "fix_started" }

// FixDoneEvent is emitted when a fix attempt completes.
type FixDoneEvent struct {
	PRURL    string
	Attempt  int
	ExitCode int
}

func (e FixDoneEvent) eventType() string { return "fix_done" }

// RunnerStoppedEvent is emitted when the runner shuts down gracefully.
type RunnerStoppedEvent struct{}

func (e RunnerStoppedEvent) eventType() string { return "runner_stopped" }

// RunnerErrorEvent is emitted when the runner encounters a non-fatal error.
type RunnerErrorEvent struct {
	Err error
}

func (e RunnerErrorEvent) eventType() string { return "runner_error" }

// ToolStartEvent is emitted when Claude begins invoking a tool.
type ToolStartEvent struct {
	ToolName string
	Input    map[string]any
}

func (e ToolStartEvent) eventType() string { return "tool_start" }

// ToolResultEvent is emitted when a tool invocation completes with a result.
type ToolResultEvent struct {
	ToolName   string
	DurationMs int
	Truncated  bool
}

func (e ToolResultEvent) eventType() string { return "tool_result" }

// TextOutputEvent is emitted when Claude produces text output.
type TextOutputEvent struct {
	Text string
}

func (e TextOutputEvent) eventType() string { return "text_output" }

// StatsUpdateEvent is emitted with updated token usage and turn count statistics.
type StatsUpdateEvent struct {
	InputTokens     int
	OutputTokens    int
	CacheReadTokens int
	Turns           int
}

func (e StatsUpdateEvent) eventType() string { return "stats_update" }
