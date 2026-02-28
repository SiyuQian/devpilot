package taskrunner

import (
	"context"
	"fmt"
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

func ReviewPrompt(prURL string) string {
	return fmt.Sprintf(`You are a code reviewer. Review the pull request at: %s

Steps:
1. Run "gh pr diff" to see the full diff of the PR
2. Analyze the changes for:
   - Bugs or logic errors
   - Security vulnerabilities
   - Performance issues
   - Code style and readability
   - Missing error handling
   - Test coverage gaps
3. Post your review using "gh pr review" with appropriate comments

If the changes look good, approve the PR:
  gh pr review --approve --body "your summary"

If there are issues, request changes:
  gh pr review --request-changes --body "your summary"

Be concise and actionable in your feedback. Focus on substantive issues, not style nitpicks.`, prURL)
}
