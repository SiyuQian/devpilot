package runner

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
)

type ExecuteResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	TimedOut bool
}

// OutputLine represents a single line of output from a running command.
type OutputLine struct {
	Stream string // "stdout" or "stderr"
	Text   string
}

// OutputHandler is called for each line of output during execution.
type OutputHandler func(line OutputLine)

type Executor struct {
	command       string
	args          []string
	outputHandler OutputHandler
}

type ExecutorOption func(*Executor)

func WithCommand(command string, args ...string) ExecutorOption {
	return func(e *Executor) {
		e.command = command
		e.args = args
	}
}

func WithOutputHandler(handler OutputHandler) ExecutorOption {
	return func(e *Executor) {
		e.outputHandler = handler
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

	if e.outputHandler == nil {
		return e.runBuffered(ctx, cmd)
	}
	return e.runStreaming(ctx, cmd)
}

// runBuffered is the original behavior: capture all output at once.
func (e *Executor) runBuffered(ctx context.Context, cmd *exec.Cmd) (*ExecuteResult, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &ExecuteResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	return e.handleResult(ctx, err, result)
}

// runStreaming reads stdout/stderr line-by-line, calling the handler for each.
func (e *Executor) runStreaming(ctx context.Context, cmd *exec.Cmd) (*ExecuteResult, error) {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	// Use a process group so we can kill all child processes on cancellation.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}

	// Kill the entire process group when the context is cancelled.
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		case <-done:
		}
	}()

	var stdout, stderr bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		e.scanStream(stdoutPipe, "stdout", &stdout)
	}()
	go func() {
		defer wg.Done()
		e.scanStream(stderrPipe, "stderr", &stderr)
	}()

	wg.Wait()
	close(done)

	waitErr := cmd.Wait()

	result := &ExecuteResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	return e.handleResult(ctx, waitErr, result)
}

// scanStream reads lines from a pipe, calls the handler, and accumulates output.
func (e *Executor) scanStream(pipe io.Reader, stream string, buf *bytes.Buffer) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteByte('\n')
		e.outputHandler(OutputLine{Stream: stream, Text: line})
	}
}

// handleResult processes the command's exit status and context errors.
func (e *Executor) handleResult(ctx context.Context, err error, result *ExecuteResult) (*ExecuteResult, error) {
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
