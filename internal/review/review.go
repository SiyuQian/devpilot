package review

import (
	"context"

	"github.com/siyuqian/devpilot/internal/executor"
)

// DefaultReviewModel is the default Claude model used for code reviews.
const DefaultReviewModel = "claude-opus-4-6-20250415"

// Option configures a review invocation.
type Option func(*options)

type options struct {
	model        string
	execOpts     []executor.ExecutorOption
	eventHandler executor.ClaudeEventHandler
}

// WithModel overrides the Claude model for the review.
func WithModel(model string) Option {
	return func(o *options) {
		o.model = model
	}
}

// WithEventHandler sets a callback for streaming Claude events during review.
func WithEventHandler(handler executor.ClaudeEventHandler) Option {
	return func(o *options) {
		o.eventHandler = handler
	}
}

// WithExecutorOptions passes additional executor options (e.g., for streaming).
func WithExecutorOptions(opts ...executor.ExecutorOption) Option {
	return func(o *options) {
		o.execOpts = append(o.execOpts, opts...)
	}
}

// Review performs an AI-powered code review on the given PR URL.
func Review(ctx context.Context, prURL string, opts ...Option) (*executor.ExecuteResult, error) {
	pr, err := ParsePRURL(prURL)
	if err != nil {
		return nil, err
	}

	o := resolveOptions(opts)
	prompt := BuildPrompt(pr)
	exec := newReviewExecutor(o)
	return exec.Run(ctx, prompt)
}

// Fix reads review comments on the given PR and addresses them.
func Fix(ctx context.Context, prURL string, opts ...Option) (*executor.ExecuteResult, error) {
	_, err := ParsePRURL(prURL)
	if err != nil {
		return nil, err
	}

	o := resolveOptions(opts)
	prompt := BuildFixPrompt(prURL)
	exec := newFixExecutor(o)
	return exec.Run(ctx, prompt)
}

// BuildFixPrompt returns the prompt for fixing review comments.
func BuildFixPrompt(prURL string) string {
	return "Fix the code review comments on " + prURL + ". Read the review comments with `gh pr view` and the diff with `gh pr diff`. Address all requested changes. Commit and push your fixes."
}

func resolveOptions(opts []Option) *options {
	o := &options{
		model: DefaultReviewModel,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func newReviewExecutor(o *options) *executor.Executor {
	args := []string{"-p", "--thinking", "--model", o.model, "--verbose", "--output-format", "stream-json", "--allowedTools=*"}
	allOpts := []executor.ExecutorOption{executor.WithCommand("claude", args...)}
	if o.eventHandler != nil {
		allOpts = append(allOpts, executor.WithClaudeEventHandler(o.eventHandler))
	}
	allOpts = append(allOpts, o.execOpts...)
	return executor.NewExecutor(allOpts...)
}

func newFixExecutor(o *options) *executor.Executor {
	args := []string{"-p", "--verbose", "--output-format", "stream-json", "--allowedTools=*"}
	allOpts := []executor.ExecutorOption{executor.WithCommand("claude", args...)}
	allOpts = append(allOpts, o.execOpts...)
	return executor.NewExecutor(allOpts...)
}
