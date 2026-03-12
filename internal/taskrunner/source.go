package taskrunner

// Task is a provider-agnostic unit of work.
type Task struct {
	ID          string
	Name        string
	Description string
	URL         string
	Priority    int   // 0=P0, 1=P1, 2=P2 (default)
	CreatedAt   int64 // Unix timestamp; used as tiebreaker within the same priority (FIFO)
}

// SourceInfo is returned by TaskSource.Init and used to populate RunnerStartedEvent.
type SourceInfo struct {
	DisplayName string
	BoardID     string            // optional; empty for GitHub
	Lists       map[string]string // optional; nil for GitHub
}

// TaskSource is the interface for task management backends.
type TaskSource interface {
	Init() (SourceInfo, error)
	FetchReady() ([]Task, error)
	MarkInProgress(id string) error
	MarkDone(id, comment string) error
	MarkFailed(id, comment string) error
}
