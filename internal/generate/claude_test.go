package generate

import (
	"strings"
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

func TestBuildCommitPrompt(t *testing.T) {
	prompt, err := buildCommitPrompt("file1.go\nfile2.go", "2 files changed, 10 insertions", "fixing auth bug")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "file1.go") {
		t.Error("prompt should contain file list")
	}
	if !strings.Contains(prompt, "10 insertions") {
		t.Error("prompt should contain diff stat")
	}
	if !strings.Contains(prompt, "fixing auth bug") {
		t.Error("prompt should contain context")
	}
	if !strings.Contains(prompt, "conventional commits") {
		t.Error("prompt should mention conventional commits")
	}
}

func TestBuildCommitPromptNoContext(t *testing.T) {
	prompt, err := buildCommitPrompt("file1.go", "1 file changed", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(prompt, "Additional context") {
		t.Error("prompt should not contain context section when empty")
	}
}

func TestBuildReadmePrompt(t *testing.T) {
	prompt, err := buildReadmePrompt("src/\n  main.go\n  util.go", "module example.com/foo", "# Old Readme")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "main.go") {
		t.Error("prompt should contain file tree")
	}
	if !strings.Contains(prompt, "module example.com/foo") {
		t.Error("prompt should contain package info")
	}
	if !strings.Contains(prompt, "Old Readme") {
		t.Error("prompt should contain existing readme")
	}
}

func TestCollectFileTree(t *testing.T) {
	tree, err := collectFileTree(".")
	if err != nil {
		t.Fatal(err)
	}
	if tree == "" {
		t.Error("file tree should not be empty")
	}
}
