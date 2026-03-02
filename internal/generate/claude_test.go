package generate

import (
	"testing"
)

func TestCleanOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "feat: add feature", "feat: add feature"},
		{"markdown fences", "```\nfeat: add feature\n```", "feat: add feature"},
		{"leading whitespace", "\n\n  feat: add feature\n\n", "feat: add feature"},
		{"ai preamble", "Here is the commit message:\nfeat: add feature", "feat: add feature"},
		{"ai preamble with blank", "Here's a commit message:\n\nfeat: add feature", "feat: add feature"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanOutput(tt.input)
			if got != tt.want {
				t.Errorf("cleanOutput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildArgs(t *testing.T) {
	args := buildArgs("claude-haiku-4-5")
	if args[0] != "--print" {
		t.Errorf("first arg should be --print, got %q", args[0])
	}
	found := false
	for i, a := range args {
		if a == "--model" {
			found = true
			if args[i+1] != "claude-haiku-4-5" {
				t.Errorf("model arg = %q, want claude-haiku-4-5", args[i+1])
			}
		}
	}
	if !found {
		t.Error("--model flag not found")
	}

	argsNoModel := buildArgs("")
	for _, a := range argsNoModel {
		if a == "--model" {
			t.Error("--model should not be present when model is empty")
		}
	}
}
