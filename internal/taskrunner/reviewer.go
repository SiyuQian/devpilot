package taskrunner

import (
	"context"
	"fmt"
	"strings"
)

// Reviewer runs automated code reviews and fix attempts via Claude.
type Reviewer struct {
	executor *Executor
}

// NewReviewer creates a Reviewer with the given executor options.
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

// ReviewPrompt returns the Claude prompt for reviewing the given PR.
func ReviewPrompt(prURL string) string {
	return fmt.Sprintf("Code review: %s", prURL)
}

// FixPrompt returns the Claude prompt for fixing review comments on the given PR.
func FixPrompt(prURL string) string {
	return fmt.Sprintf(`Fix the code review comments on %s. Read the review with gh pr view and address all requested changes. Commit and push your fixes.`, prURL)
}

// IsApproved reports whether the review output indicates approval.
func IsApproved(stdout string) bool {
	return strings.Contains(stdout, "No issues found")
}
