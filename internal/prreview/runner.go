// Package prreview implements the deterministic halves of the pr-review
// skill: `devpilot pr-review preflight` (eligibility gate + data collection)
// and `devpilot pr-review post` (anchor validation + single combined POST).
package prreview

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Runner executes gh commands.
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
	// RunInput executes gh with the given bytes piped to stdin.
	RunInput(ctx context.Context, stdin []byte, args ...string) ([]byte, error)
}

type ghRunner struct{}

func (r ghRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	return r.RunInput(ctx, nil, args...)
}

func (r ghRunner) RunInput(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh %v: %s", args, exitErr.Stderr)
		}
		return nil, fmt.Errorf("gh %v: %w", args, err)
	}
	return out, nil
}
