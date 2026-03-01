# Task Retry & Error Recovery Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** When a task fails due to a transient error (API timeout, rate limit, service outage), automatically retry with exponential backoff instead of immediately moving the card to Failed. Persistent failures still fail the card after max retries.

**Architecture:** Add a `RetryPolicy` to the runner that classifies errors as transient vs. permanent. Transient errors trigger retry with backoff. The card stays in "In Progress" during retries. Retry count and errors are tracked in card comments for visibility.

**Tech Stack:** Go

---

### Task 1: Implement error classification

**Files:**
- Create: `internal/taskrunner/retrypolicy.go`
- Create: `internal/taskrunner/retrypolicy_test.go`

**Step 1: Write failing tests for error classification**

Test cases:
- `TestClassify_TransientTimeout`: Execution timed out → transient
- `TestClassify_TransientExitCode`: Exit code 1 with stderr containing "rate limit" or "timeout" or "503" or "overloaded" → transient
- `TestClassify_PermanentNoCommits`: Completed successfully but no commits → permanent
- `TestClassify_PermanentEmptyDesc`: Empty description → permanent
- `TestClassify_PermanentGitError`: Git checkout/branch failure → permanent
- `TestClassify_PermanentTestFailure`: Test gate failure → permanent

**Step 2: Implement RetryPolicy**

```go
type ErrorKind int

const (
    PermanentError ErrorKind = iota
    TransientError
)

type RetryPolicy struct {
    MaxRetries int           // default 2 (total 3 attempts)
    BaseDelay  time.Duration // default 30s
    MaxDelay   time.Duration // default 5min
}

func (rp *RetryPolicy) Classify(result *ExecutionResult, err error) ErrorKind
func (rp *RetryPolicy) Delay(attempt int) time.Duration // exponential backoff with jitter
```

Classification rules:
- `result.TimedOut` → Transient
- Exit code non-zero AND stderr matches transient patterns → Transient
- Everything else → Permanent

Backoff: `min(BaseDelay * 2^attempt + jitter, MaxDelay)`

**Step 3: Verify all tests pass**

---

### Task 2: Add retry loop to runner

**Files:**
- Edit: `internal/taskrunner/runner.go`
- Edit: `internal/taskrunner/events.go`

**Step 1: Add retry events**

Add new events:
- `RetryScheduledEvent` — emitted when a retry is planned, includes attempt number, delay, and error reason
- `RetryAttemptEvent` — emitted when a retry attempt starts

**Step 2: Wrap execution in retry loop**

In `processCard()`, wrap the Claude execution + verification section in a retry loop:

```go
for attempt := 0; attempt <= r.retryPolicy.MaxRetries; attempt++ {
    if attempt > 0 {
        // Reset git state: checkout branch, reset to main
        r.emit(RetryAttemptEvent{Attempt: attempt, CardID: card.ID})
    }

    result := r.executor.Run(ctx, prompt)

    if result succeeded with commits {
        break // continue to test gate and PR
    }

    kind := r.retryPolicy.Classify(result, err)
    if kind == PermanentError || attempt == r.retryPolicy.MaxRetries {
        failCard(card, err)
        return
    }

    delay := r.retryPolicy.Delay(attempt)
    r.emit(RetryScheduledEvent{Attempt: attempt, Delay: delay, Reason: err.Error()})

    // Add comment to card about retry
    r.trello.AddComment(card.ID, fmt.Sprintf("Attempt %d failed: %s. Retrying in %s...", attempt+1, err, delay))

    select {
    case <-time.After(delay):
    case <-ctx.Done():
        failCard(card, ctx.Err())
        return
    }
}
```

**Step 3: Add git reset between retries**

Before each retry attempt, reset the branch to start fresh:
- `git reset --hard main` on the task branch
- This ensures Claude gets a clean slate

**Step 4: Write integration test**

Test that:
- Transient error on attempt 1 → retries → succeeds on attempt 2
- Permanent error → fails immediately (no retry)
- Max retries exceeded → fails after all attempts with combined error history

---

### Task 3: Add retry configuration

**Files:**
- Edit: `internal/taskrunner/runner.go`

**Step 1: Add retry config to Runner**

Add `RetryPolicy` field to `Runner` struct. Add functional option:

```go
func WithRetryPolicy(policy RetryPolicy) RunnerOption
```

Default: `RetryPolicy{MaxRetries: 2, BaseDelay: 30*time.Second, MaxDelay: 5*time.Minute}`

**Step 2: Add CLI flags**

Add to `devpilot run`:
- `--max-retries` (default 2): Maximum retry attempts per task
- `--no-retry`: Disable retry entirely (sets MaxRetries=0)

**Step 3: Verify flags work**

---

### Task 4: Update TUI for retry status

**Files:**
- Edit: `internal/taskrunner/tui.go` (or relevant TUI file)

**Step 1: Handle retry events in TUI**

Display:
- "Retry scheduled: attempt 2/3 in 30s (rate limit)" when RetryScheduledEvent
- "Retry attempt 2/3..." when RetryAttemptEvent
- Show retry count in the active task panel

**Step 2: Verify TUI compiles and retry events render**
