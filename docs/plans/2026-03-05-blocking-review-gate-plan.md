# Blocking Review Gate Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make code review a blocking gate with a self-heal loop — review finds issues, Claude fixes them, re-reviews, up to 5 retries.

**Architecture:** Modify the existing review block in `processCard` to loop: review → check approval → fix → push → re-review. Add `IsApproved()` and `FixPrompt()` to reviewer, new events for TUI.

**Tech Stack:** Go, Cobra, Bubble Tea

---

### Task 1: Create `config.go` with constants

**Files:**
- Create: `internal/taskrunner/config.go`
- Test: `internal/taskrunner/config_test.go`

**Step 1: Write the test**

```go
// internal/taskrunner/config_test.go
package taskrunner

import "testing"

func TestMaxReviewRetries(t *testing.T) {
	if MaxReviewRetries < 1 {
		t.Error("MaxReviewRetries must be at least 1")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/taskrunner/ -run TestMaxReviewRetries -v`
Expected: FAIL — `MaxReviewRetries` not defined

**Step 3: Write implementation**

```go
// internal/taskrunner/config.go
package taskrunner

const (
	// MaxReviewRetries is the maximum number of fix attempts after a code
	// review requests changes.
	MaxReviewRetries = 5
)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/taskrunner/ -run TestMaxReviewRetries -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/taskrunner/config.go internal/taskrunner/config_test.go
git commit -m "feat: add config.go with MaxReviewRetries constant"
```

---

### Task 2: Add new events (`FixStartedEvent`, `FixDoneEvent`)

**Files:**
- Modify: `internal/taskrunner/events.go` (append after line 68, after `ReviewDoneEvent`)
- Modify: `internal/taskrunner/events_test.go` (add to test table)

**Step 1: Write the test**

Add two entries to the `tests` table in `internal/taskrunner/events_test.go`, after the `ReviewDone` entry (line 21):

```go
{"FixStarted", FixStartedEvent{PRURL: "http://pr", Attempt: 1}, "fix_started"},
{"FixDone", FixDoneEvent{PRURL: "http://pr", Attempt: 1, ExitCode: 0}, "fix_done"},
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/taskrunner/ -run TestEventTypes -v`
Expected: FAIL — `FixStartedEvent` and `FixDoneEvent` not defined

**Step 3: Write implementation**

Append to `internal/taskrunner/events.go` after line 68 (after `ReviewDoneEvent`):

```go
type FixStartedEvent struct {
	PRURL   string
	Attempt int
}

func (e FixStartedEvent) eventType() string { return "fix_started" }

type FixDoneEvent struct {
	PRURL    string
	Attempt  int
	ExitCode int
}

func (e FixDoneEvent) eventType() string { return "fix_done" }
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/taskrunner/ -run TestEventTypes -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/taskrunner/events.go internal/taskrunner/events_test.go
git commit -m "feat: add FixStartedEvent and FixDoneEvent"
```

---

### Task 3: Update reviewer with new prompts and `IsApproved()`

**Files:**
- Modify: `internal/taskrunner/reviewer.go`
- Modify: `internal/taskrunner/reviewer_test.go`

**Step 1: Write the tests**

Append to `internal/taskrunner/reviewer_test.go`:

```go
func TestReviewPrompt_UsesCodeReviewSkill(t *testing.T) {
	prompt := ReviewPrompt("https://github.com/user/repo/pull/42")

	if !strings.Contains(prompt, "Code review:") {
		t.Error("prompt should use code-review skill trigger")
	}
	if !strings.Contains(prompt, "https://github.com/user/repo/pull/42") {
		t.Error("prompt should contain PR URL")
	}
}

func TestFixPrompt(t *testing.T) {
	prompt := FixPrompt("https://github.com/user/repo/pull/42")

	if !strings.Contains(prompt, "https://github.com/user/repo/pull/42") {
		t.Error("prompt should contain PR URL")
	}
	if !strings.Contains(prompt, "Fix") {
		t.Error("prompt should instruct to fix")
	}
}

func TestIsApproved(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		want   bool
	}{
		{"approved", "No issues found. Checked for bugs and CLAUDE.md compliance.", true},
		{"approved partial", "blah blah\nNo issues found\nmore text", true},
		{"rejected", "Found 3 issues:\n1. Bug in foo.go", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsApproved(tt.stdout)
			if got != tt.want {
				t.Errorf("IsApproved(%q) = %v, want %v", tt.stdout, got, tt.want)
			}
		})
	}
}

func TestReviewer_Fix(t *testing.T) {
	reviewer := NewReviewer(WithCommand("echo", "fix done"))
	ctx := context.Background()

	result, err := reviewer.Fix(ctx, "https://github.com/user/repo/pull/42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", result.ExitCode)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/taskrunner/ -run "TestReviewPrompt_UsesCodeReviewSkill|TestFixPrompt|TestIsApproved|TestReviewer_Fix" -v`
Expected: FAIL — `FixPrompt`, `IsApproved`, `Fix` not defined

**Step 3: Write implementation**

Replace `internal/taskrunner/reviewer.go` entirely:

```go
package taskrunner

import (
	"context"
	"fmt"
	"strings"
)

type Reviewer struct {
	executor *Executor
}

func NewReviewer(opts ...ExecutorOption) *Reviewer {
	return &Reviewer{
		executor: NewExecutor(opts...),
	}
}

func (rv *Reviewer) Review(ctx context.Context, prURL string) (*ExecuteResult, error) {
	prompt := ReviewPrompt(prURL)
	return rv.executor.Run(ctx, prompt)
}

func (rv *Reviewer) Fix(ctx context.Context, prURL string) (*ExecuteResult, error) {
	prompt := FixPrompt(prURL)
	return rv.executor.Run(ctx, prompt)
}

func ReviewPrompt(prURL string) string {
	return fmt.Sprintf("Code review: %s", prURL)
}

func FixPrompt(prURL string) string {
	return fmt.Sprintf(`Fix the code review comments on %s. Read the review with gh pr view and address all requested changes. Commit and push your fixes.`, prURL)
}

func IsApproved(stdout string) bool {
	return strings.Contains(stdout, "No issues found")
}
```

**Step 4: Run all reviewer tests**

Run: `go test ./internal/taskrunner/ -run "TestReview|TestFix|TestIsApproved" -v`
Expected: PASS. Note: `TestReviewPrompt` (the old test checking for `gh pr diff`) will now fail because we changed the prompt. Delete it — it's been replaced by `TestReviewPrompt_UsesCodeReviewSkill`.

**Step 4b: Remove old `TestReviewPrompt`**

Delete the `TestReviewPrompt` function (lines 10-27) from `reviewer_test.go`.

**Step 4c: Run all reviewer tests again**

Run: `go test ./internal/taskrunner/ -run "TestReview|TestFix|TestIsApproved" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/taskrunner/reviewer.go internal/taskrunner/reviewer_test.go
git commit -m "feat: update reviewer with code-review skill prompt, add Fix() and IsApproved()"
```

---

### Task 4: Implement the blocking review loop in `runner.go`

**Files:**
- Modify: `internal/taskrunner/runner.go` (lines 237-258, the review+merge block in `processCard`)

**Step 1: Write the test**

Create a focused integration-style test. Append to a new file `internal/taskrunner/runner_review_test.go`:

```go
package taskrunner

import (
	"testing"
)

func TestIsApproved_InReviewLoop(t *testing.T) {
	// This tests the approval detection logic that the review loop depends on.
	// The full loop is tested via the reviewer tests + integration.
	tests := []struct {
		name     string
		stdout   string
		approved bool
	}{
		{"clean review", "No issues found. Checked for bugs.", true},
		{"issues found", "Found 2 issues:\n1. Bug", false},
		{"empty output", "", false},
		{"partial match", "No issues found buried in text", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsApproved(tt.stdout); got != tt.approved {
				t.Errorf("IsApproved() = %v, want %v", got, tt.approved)
			}
		})
	}
}
```

**Step 2: Run test to verify it passes** (this test validates the helper, not the loop itself)

Run: `go test ./internal/taskrunner/ -run TestIsApproved_InReviewLoop -v`
Expected: PASS (since `IsApproved` was implemented in Task 3)

**Step 3: Replace the review+merge block in `runner.go`**

In `internal/taskrunner/runner.go`, replace lines 237-258 (the `// Code review (non-blocking)` block through the `r.git.MergePR()` call) with:

```go
	// Code review gate (blocking with self-heal loop)
	if r.reviewer != nil {
		approved := false
		for attempt := 0; attempt <= MaxReviewRetries; attempt++ {
			r.logger.Printf("Running code review for PR: %s (attempt %d)", prURL, attempt+1)
			r.emit(ReviewStartedEvent{PRURL: prURL})
			reviewCtx, reviewCancel := context.WithTimeout(ctx, r.config.ReviewTimeout)
			reviewResult, reviewErr := r.reviewer.Review(reviewCtx, prURL)
			reviewCancel()

			if reviewErr != nil {
				r.logger.Printf("Code review error: %v", reviewErr)
				r.emit(ReviewDoneEvent{PRURL: prURL, ExitCode: -1})
				break
			}

			r.emit(ReviewDoneEvent{PRURL: prURL, ExitCode: reviewResult.ExitCode})

			if IsApproved(reviewResult.Stdout) {
				r.logger.Printf("Code review approved for PR: %s", prURL)
				approved = true
				break
			}

			// Review found issues — attempt fix if retries remain
			if attempt < MaxReviewRetries {
				r.logger.Printf("Review found issues, attempting fix (attempt %d/%d)", attempt+1, MaxReviewRetries)
				r.emit(FixStartedEvent{PRURL: prURL, Attempt: attempt + 1})
				fixCtx, fixCancel := context.WithTimeout(ctx, r.config.ReviewTimeout)
				fixResult, fixErr := r.reviewer.Fix(fixCtx, prURL)
				fixCancel()

				fixExitCode := -1
				if fixErr == nil {
					fixExitCode = fixResult.ExitCode
				}
				r.emit(FixDoneEvent{PRURL: prURL, Attempt: attempt + 1, ExitCode: fixExitCode})

				if fixErr != nil {
					r.logger.Printf("Fix attempt failed: %v", fixErr)
					continue
				}

				// Push the fix
				if err := r.git.Push(branch); err != nil {
					r.logger.Printf("Failed to push fix: %v", err)
					r.failCard(task, start, fmt.Sprintf("push fix: %v", err))
					r.git.CheckoutMain()
					return
				}
			}
		}

		if !approved {
			r.failCard(task, start, fmt.Sprintf("code review failed after %d attempts", MaxReviewRetries+1))
			r.git.CheckoutMain()
			return
		}
	}

	if err := r.git.MergePR(); err != nil {
		r.logger.Printf("Auto-merge failed (may need approval): %v", err)
	}
```

**Step 4: Run full test suite**

Run: `go test ./internal/taskrunner/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/taskrunner/runner.go internal/taskrunner/runner_review_test.go
git commit -m "feat: make code review a blocking gate with self-heal loop"
```

---

### Task 5: Handle new events in TUI and plain-text handler

**Files:**
- Modify: `internal/taskrunner/tui.go` (add cases in `Update` method, after `ReviewDoneEvent` case ~line 343)
- Modify: `internal/taskrunner/commands.go` (add cases in plain-text handler, after `ReviewDoneEvent` case ~line 171)

**Step 1: Write the test**

Append to `internal/taskrunner/tui_test.go` (check existing patterns in the file first):

```go
func TestTUIModel_FixEvents(t *testing.T) {
	eventCh := make(chan Event, 10)
	cancel := func() {}
	m := NewTUIModel("test", eventCh, cancel)

	// Simulate FixStartedEvent
	updated, _ := m.Update(FixStartedEvent{PRURL: "http://pr", Attempt: 1})
	model := updated.(TUIModel)
	if model.phase != "starting" {
		t.Errorf("phase should not change on fix event, got %q", model.phase)
	}

	// Simulate FixDoneEvent
	updated, _ = model.Update(FixDoneEvent{PRURL: "http://pr", Attempt: 1, ExitCode: 0})
	model = updated.(TUIModel)
	if model.phase != "starting" {
		t.Errorf("phase should not change on fix done event, got %q", model.phase)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/taskrunner/ -run TestTUIModel_FixEvents -v`
Expected: FAIL — unhandled event type causes no `waitForEvent` return, test may hang or fail

**Step 3: Write TUI handler**

Add to `internal/taskrunner/tui.go`, after the `ReviewDoneEvent` case (after line 343):

```go
	case FixStartedEvent:
		return m, waitForEvent(m.eventCh)

	case FixDoneEvent:
		return m, waitForEvent(m.eventCh)
```

**Step 4: Write plain-text handler**

Add to `internal/taskrunner/commands.go`, in the plain-text event handler switch, after the `ReviewDoneEvent` case (after line 171):

```go
		case FixStartedEvent:
			logger.Printf("[fix] Attempting fix for %s (attempt %d)", ev.PRURL, ev.Attempt)
		case FixDoneEvent:
			logger.Printf("[fix] Fix done (attempt %d, exit %d)", ev.Attempt, ev.ExitCode)
```

**Step 5: Run all tests**

Run: `go test ./internal/taskrunner/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/taskrunner/tui.go internal/taskrunner/tui_test.go internal/taskrunner/commands.go
git commit -m "feat: handle FixStarted/FixDone events in TUI and plain-text output"
```

---

### Task 6: Final verification

**Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: ALL PASS

**Step 2: Build**

Run: `make build`
Expected: Success, binary at `bin/devpilot`

**Step 3: Smoke test help output**

Run: `bin/devpilot run --help`
Expected: Shows existing flags, no new flags. Confirm `--review-timeout` is present.

**Step 4: Commit any remaining changes** (if any)

**Step 5: Final commit message**

```bash
git log --oneline -5
```

Verify 5 commits from this plan:
1. `feat: add config.go with MaxReviewRetries constant`
2. `feat: add FixStartedEvent and FixDoneEvent`
3. `feat: update reviewer with code-review skill prompt, add Fix() and IsApproved()`
4. `feat: make code review a blocking gate with self-heal loop`
5. `feat: handle FixStarted/FixDone events in TUI and plain-text output`
