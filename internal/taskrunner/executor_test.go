package taskrunner

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestExecute_Success(t *testing.T) {
	exec := NewExecutor(WithCommand("echo", "hello"))
	result, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", result.ExitCode)
	}
	if result.Stdout == "" {
		t.Error("expected stdout output")
	}
}

func TestExecute_Failure(t *testing.T) {
	exec := NewExecutor(WithCommand("false"))
	result, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestExecute_Timeout(t *testing.T) {
	exec := NewExecutor(WithCommand("sleep", "10"))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	result, err := exec.Run(ctx, "test prompt")
	if err == nil && !result.TimedOut {
		t.Error("expected timeout")
	}
}

// --- Streaming tests ---

func TestExecute_StreamHandlerReceivesStdoutLines(t *testing.T) {
	var mu sync.Mutex
	var lines []OutputLine

	handler := func(line OutputLine) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, line)
	}

	// Use printf to emit multiple lines to stdout
	exec := NewExecutor(
		WithCommand("printf", "line1\nline2\nline3\n"),
		WithOutputHandler(handler),
	)
	result, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Handler should have received 3 stdout lines
	stdoutLines := filterStream(lines, "stdout")
	if len(stdoutLines) != 3 {
		t.Fatalf("expected 3 stdout lines, got %d: %v", len(stdoutLines), stdoutLines)
	}
	if stdoutLines[0].Text != "line1" {
		t.Errorf("expected first line 'line1', got %q", stdoutLines[0].Text)
	}
	if stdoutLines[1].Text != "line2" {
		t.Errorf("expected second line 'line2', got %q", stdoutLines[1].Text)
	}
	if stdoutLines[2].Text != "line3" {
		t.Errorf("expected third line 'line3', got %q", stdoutLines[2].Text)
	}

	// Result should still contain the full output
	if !strings.Contains(result.Stdout, "line1") {
		t.Errorf("result.Stdout should contain full output, got %q", result.Stdout)
	}
}

func TestExecute_StreamHandlerReceivesStderrLines(t *testing.T) {
	var mu sync.Mutex
	var lines []OutputLine

	handler := func(line OutputLine) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, line)
	}

	// Write to stderr using sh -c
	exec := NewExecutor(
		WithCommand("sh", "-c", "echo err1 >&2 && echo err2 >&2"),
		WithOutputHandler(handler),
	)
	result, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	stderrLines := filterStream(lines, "stderr")
	if len(stderrLines) != 2 {
		t.Fatalf("expected 2 stderr lines, got %d: %v", len(stderrLines), stderrLines)
	}
	if stderrLines[0].Text != "err1" {
		t.Errorf("expected first stderr line 'err1', got %q", stderrLines[0].Text)
	}

	// Result should still contain stderr
	if !strings.Contains(result.Stderr, "err1") {
		t.Errorf("result.Stderr should contain full output, got %q", result.Stderr)
	}
}

func TestExecute_StreamHandlerMixedStdoutStderr(t *testing.T) {
	var mu sync.Mutex
	var lines []OutputLine

	handler := func(line OutputLine) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, line)
	}

	exec := NewExecutor(
		WithCommand("sh", "-c", "echo out1 && echo err1 >&2 && echo out2"),
		WithOutputHandler(handler),
	)
	result, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	stdoutLines := filterStream(lines, "stdout")
	stderrLines := filterStream(lines, "stderr")

	if len(stdoutLines) != 2 {
		t.Errorf("expected 2 stdout lines, got %d", len(stdoutLines))
	}
	if len(stderrLines) != 1 {
		t.Errorf("expected 1 stderr line, got %d", len(stderrLines))
	}

	// Full output should still be captured
	if !strings.Contains(result.Stdout, "out1") || !strings.Contains(result.Stdout, "out2") {
		t.Errorf("result.Stdout missing content, got %q", result.Stdout)
	}
	if !strings.Contains(result.Stderr, "err1") {
		t.Errorf("result.Stderr missing content, got %q", result.Stderr)
	}
}

func TestExecute_NoHandlerBackwardCompatible(t *testing.T) {
	// No handler set — should work exactly as before
	exec := NewExecutor(WithCommand("printf", "hello\nworld\n"))
	result, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello") || !strings.Contains(result.Stdout, "world") {
		t.Errorf("expected stdout to contain hello and world, got %q", result.Stdout)
	}
}

func TestExecute_StreamHandlerWithTimeout(t *testing.T) {
	var mu sync.Mutex
	var lines []OutputLine

	handler := func(line OutputLine) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, line)
	}

	// Command that outputs then hangs
	exec := NewExecutor(
		WithCommand("sh", "-c", "echo before_timeout && sleep 10"),
		WithOutputHandler(handler),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	result, err := exec.Run(ctx, "test prompt")
	if err == nil && !result.TimedOut {
		t.Error("expected timeout")
	}

	mu.Lock()
	defer mu.Unlock()

	// Handler should have received the line emitted before timeout
	stdoutLines := filterStream(lines, "stdout")
	if len(stdoutLines) < 1 {
		t.Error("expected handler to receive at least one line before timeout")
	}
	if len(stdoutLines) > 0 && stdoutLines[0].Text != "before_timeout" {
		t.Errorf("expected 'before_timeout', got %q", stdoutLines[0].Text)
	}
}

func TestExecute_StreamHandlerWithNonZeroExit(t *testing.T) {
	var mu sync.Mutex
	var lines []OutputLine

	handler := func(line OutputLine) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, line)
	}

	exec := NewExecutor(
		WithCommand("sh", "-c", "echo output_before_fail && exit 1"),
		WithOutputHandler(handler),
	)
	result, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}

	mu.Lock()
	defer mu.Unlock()

	stdoutLines := filterStream(lines, "stdout")
	if len(stdoutLines) != 1 {
		t.Fatalf("expected 1 stdout line, got %d", len(stdoutLines))
	}
	if stdoutLines[0].Text != "output_before_fail" {
		t.Errorf("expected 'output_before_fail', got %q", stdoutLines[0].Text)
	}
}

// --- helpers ---

func filterStream(lines []OutputLine, stream string) []OutputLine {
	var filtered []OutputLine
	for _, l := range lines {
		if l.Stream == stream {
			filtered = append(filtered, l)
		}
	}
	return filtered
}
