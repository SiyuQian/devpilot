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
