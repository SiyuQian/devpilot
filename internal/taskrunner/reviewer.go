package taskrunner

import (
	"context"

	"github.com/siyuqian/devpilot/internal/executor"
	"github.com/siyuqian/devpilot/internal/review"
)

// Reviewer runs automated code reviews and fix attempts via the review package.
type Reviewer struct {
	pipelineOpts []review.PipelineOption
	fixExecOpts  []executor.ExecutorOption
}

// NewReviewer creates a Reviewer with the given executor options.
func NewReviewer(execOpts ...executor.ExecutorOption) *Reviewer {
	return &Reviewer{
		fixExecOpts: execOpts,
	}
}

func (rv *Reviewer) Review(ctx context.Context, prURL string) (*review.PipelineResult, error) {
	return review.Review(ctx, prURL, rv.pipelineOpts...)
}

func (rv *Reviewer) Fix(ctx context.Context, prURL string) (*executor.ExecuteResult, error) {
	return review.Fix(ctx, prURL, rv.fixExecOpts...)
}

// IsApproved reports whether the review output indicates approval.
func IsApproved(result *review.PipelineResult) bool {
	return review.IsApprovedResult(result)
}
