package runner

import (
	"context"
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
