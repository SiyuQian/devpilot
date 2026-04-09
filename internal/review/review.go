package review

import (
	"context"
	"strings"

	"github.com/siyuqian/devpilot/internal/executor"
)

// Review performs an AI-powered code review using the multi-round pipeline.
func Review(ctx context.Context, prURL string, opts ...PipelineOption) (*PipelineResult, error) {
	return RunPipeline(ctx, prURL, opts...)
}

// Fix reads review comments on the given PR and addresses them.
func Fix(ctx context.Context, prURL string, execOpts ...executor.ExecutorOption) (*executor.ExecuteResult, error) {
	_, err := ParsePRURL(prURL)
	if err != nil {
		return nil, err
	}

	prompt := BuildFixPrompt(prURL)
	exec := newFixExecutor(execOpts)
	return exec.Run(ctx, prompt)
}

// BuildFixPrompt returns the prompt for fixing review comments.
func BuildFixPrompt(prURL string) string {
	return "Fix the code review comments on " + prURL + ". Read the review comments with `gh pr view` and the diff with `gh pr diff`. Address all requested changes. Commit and push your fixes."
}

func newFixExecutor(extraOpts []executor.ExecutorOption) *executor.Executor {
	args := []string{"-p", "--verbose", "--output-format", "stream-json", "--allowedTools=*"}
	opts := []executor.ExecutorOption{executor.WithCommand("claude", args...)}
	opts = append(opts, extraOpts...)
	return executor.NewExecutor(opts...)
}

// IsApproved parses the structured review output and returns true if the verdict is APPROVE.
// Kept for backward compatibility with code that passes raw stdout.
func IsApproved(stdout string) bool {
	output, err := ParseReviewOutput(stdout)
	if err != nil {
		// Fallback: check for APPROVE in text
		return strings.Contains(stdout, "APPROVE") && !strings.Contains(stdout, "REQUEST_CHANGES")
	}
	for _, f := range output.Findings {
		if f.Severity == "CRITICAL" {
			return false
		}
	}
	return true
}
