package github

import (
	"context"
	"fmt"
	"os/exec"
)

// Runner executes gh commands.
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

type ghRunner struct{}

func (r ghRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh %v: %s", args, exitErr.Stderr)
		}
		return nil, fmt.Errorf("gh %v: %w", args, err)
	}
	return out, nil
}

func checkAuth(ctx context.Context, r Runner) error {
	if _, err := r.Run(ctx, "auth", "status"); err != nil {
		return fmt.Errorf("checking GitHub CLI auth: %w", err)
	}
	return nil
}
