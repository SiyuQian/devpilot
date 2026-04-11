package taskrunner

import (
	"context"

	"github.com/siyuqian/devpilot/internal/executor"
	"github.com/siyuqian/devpilot/internal/review"
)

// Reviewer runs automated code reviews and fix attempts via the review package.
type Reviewer struct {
	opts []review.Option
}

// NewReviewer creates a Reviewer with the given executor options.
func NewReviewer(execOpts ...executor.ExecutorOption) *Reviewer {
	var opts []review.Option
	if len(execOpts) > 0 {
		opts = append(opts, review.WithExecutorOptions(execOpts...))
	}
	return &Reviewer{opts: opts}
}

// Review runs an automated code review against the given PR URL.
func (rv *Reviewer) Review(ctx context.Context, prURL string) (*executor.ExecuteResult, error) {
	return review.Review(ctx, prURL, rv.opts...)
}

// Fix runs an automated fix attempt against the given PR URL.
func (rv *Reviewer) Fix(ctx context.Context, prURL string) (*executor.ExecuteResult, error) {
	return review.Fix(ctx, prURL, rv.opts...)
}

// IsApproved reports whether the review output indicates approval.
// Delegates to the review package's structured verdict parser.
func IsApproved(stdout string) bool {
	return review.IsApproved(stdout)
}
