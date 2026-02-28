package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestReviewPrompt(t *testing.T) {
	prompt := ReviewPrompt("https://github.com/user/repo/pull/42")

	// Must contain the PR URL
	if !strings.Contains(prompt, "https://github.com/user/repo/pull/42") {
		t.Error("prompt should contain PR URL")
	}

	// Must instruct to use gh pr diff
	if !strings.Contains(prompt, "gh pr diff") {
		t.Error("prompt should instruct to use gh pr diff")
	}

	// Must instruct to use gh pr review
	if !strings.Contains(prompt, "gh pr review") {
		t.Error("prompt should instruct to use gh pr review")
	}
}

func TestReviewer_Review(t *testing.T) {
	// Use echo as a mock command — simulates a successful review
	reviewer := NewReviewer(WithCommand("echo", "review done"))
	ctx := context.Background()

	result, err := reviewer.Review(ctx, "https://github.com/user/repo/pull/42")
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

func TestReviewer_ReviewTimeout(t *testing.T) {
	reviewer := NewReviewer(WithCommand("sleep", "10"))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := reviewer.Review(ctx, "https://github.com/user/repo/pull/42")
	if err == nil && !result.TimedOut {
		t.Error("expected timeout")
	}
}

func TestReviewer_ReviewFailure(t *testing.T) {
	// Use false to simulate a failed review
	reviewer := NewReviewer(WithCommand("false"))

	result, err := reviewer.Review(context.Background(), "https://github.com/user/repo/pull/42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for failed review")
	}
}

