package taskrunner

import "time"

// EventHandler receives runner lifecycle events.
type EventHandler func(Event)

// Event is the interface all runner events implement.
type Event interface {
	eventType() string
}

type RunnerStartedEvent struct {
	BoardName string
	BoardID   string
	Lists     map[string]string // list name -> ID
}

func (e RunnerStartedEvent) eventType() string { return "runner_started" }

type PollingEvent struct{}

func (e PollingEvent) eventType() string { return "polling" }

type NoTasksEvent struct {
	NextPoll time.Duration
}

func (e NoTasksEvent) eventType() string { return "no_tasks" }

type CardStartedEvent struct {
	CardID   string
	CardName string
	Branch   string
}

func (e CardStartedEvent) eventType() string { return "card_started" }

type CardOutputEvent struct {
	Line OutputLine
}

func (e CardOutputEvent) eventType() string { return "card_output" }

type CardDoneEvent struct {
	CardID   string
	CardName string
	PRURL    string
	Duration time.Duration
}

func (e CardDoneEvent) eventType() string { return "card_done" }

type CardFailedEvent struct {
	CardID   string
	CardName string
	ErrMsg   string
	Duration time.Duration
}

func (e CardFailedEvent) eventType() string { return "card_failed" }

type ReviewStartedEvent struct {
	PRURL string
}

func (e ReviewStartedEvent) eventType() string { return "review_started" }

type ReviewDoneEvent struct {
	PRURL    string
	ExitCode int
}

func (e ReviewDoneEvent) eventType() string { return "review_done" }

type RunnerStoppedEvent struct{}

func (e RunnerStoppedEvent) eventType() string { return "runner_stopped" }

type RunnerErrorEvent struct {
	Err error
}

func (e RunnerErrorEvent) eventType() string { return "runner_error" }
