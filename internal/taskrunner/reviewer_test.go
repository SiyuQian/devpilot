package taskrunner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestReviewPrompt_UsesCodeReviewSkill(t *testing.T) {
	prompt := ReviewPrompt("https://github.com/user/repo/pull/42")

	if !strings.Contains(prompt, "Code review:") {
		t.Error("prompt should use code-review skill trigger")
	}
	if !strings.Contains(prompt, "https://github.com/user/repo/pull/42") {
		t.Error("prompt should contain PR URL")
	}
}

func TestFixPrompt(t *testing.T) {
	prompt := FixPrompt("https://github.com/user/repo/pull/42")

	if !strings.Contains(prompt, "https://github.com/user/repo/pull/42") {
		t.Error("prompt should contain PR URL")
	}
	if !strings.Contains(prompt, "Fix") {
		t.Error("prompt should instruct to fix")
	}
}

func TestIsApproved(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		want   bool
	}{
		{"approved", "No issues found. Checked for bugs and CLAUDE.md compliance.", true},
		{"approved partial", "blah blah\nNo issues found\nmore text", true},
		{"rejected", "Found 3 issues:\n1. Bug in foo.go", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsApproved(tt.stdout)
			if got != tt.want {
				t.Errorf("IsApproved(%q) = %v, want %v", tt.stdout, got, tt.want)
			}
		})
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

func TestReviewer_Fix(t *testing.T) {
	reviewer := NewReviewer(WithCommand("echo", "fix done"))
	ctx := context.Background()

	result, err := reviewer.Fix(ctx, "https://github.com/user/repo/pull/42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", result.ExitCode)
	}
}
