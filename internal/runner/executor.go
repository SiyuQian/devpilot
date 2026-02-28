package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"
)

type ExecuteResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	TimedOut bool
}

type Executor struct {
	command string
	args    []string
}

type ExecutorOption func(*Executor)

func WithCommand(command string, args ...string) ExecutorOption {
	return func(e *Executor) {
		e.command = command
		e.args = args
	}
}

func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		command: "claude",
		args:    []string{"-p", "--allowedTools=*"},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *Executor) Run(ctx context.Context, prompt string) (*ExecuteResult, error) {
	args := make([]string, len(e.args))
	copy(args, e.args)

	// Only append prompt if using claude (not test commands)
	if e.command == "claude" {
		args = append(args, prompt)
	}

	cmd := exec.CommandContext(ctx, e.command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &ExecuteResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if ctx.Err() != nil {
		result.TimedOut = ctx.Err() == context.DeadlineExceeded
		return result, fmt.Errorf("execution interrupted: %w", ctx.Err())
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.Sys().(syscall.WaitStatus).ExitStatus()
			return result, nil
		}
		return result, fmt.Errorf("exec failed: %w", err)
	}

	result.ExitCode = 0
	return result, nil
}
