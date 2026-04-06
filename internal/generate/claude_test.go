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
	prompt, err := buildCommitPrompt("M\tfile1.go\nA\tfile2.go", "2 files changed, 10 insertions", "+some diff content", "fixing auth bug")
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
	if !strings.Contains(prompt, "conventional commit") {
		t.Error("prompt should mention conventional commits")
	}
	if !strings.Contains(prompt, "some diff content") {
		t.Error("prompt should contain diff content")
	}
}

func TestBuildCommitPromptNoContext(t *testing.T) {
	prompt, err := buildCommitPrompt("M\tfile1.go", "1 file changed", "+change", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(prompt, "Additional context") {
		t.Error("prompt should not contain context section when empty")
	}
}

func TestBuildReadmePrompt(t *testing.T) {
	prompt, err := buildReadmePrompt("# Old Readme\nSome content")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "Old Readme") {
		t.Error("prompt should contain existing readme")
	}
	if !strings.Contains(prompt, "Exploration Strategy") {
		t.Error("prompt should contain exploration strategy section")
	}
}

func TestBuildReadmePromptEmpty(t *testing.T) {
	prompt, err := buildReadmePrompt("")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(prompt, "existing-readme") {
		t.Error("prompt should not contain existing-readme section when empty")
	}
	if !strings.Contains(prompt, "Exploration Strategy") {
		t.Error("prompt should contain exploration strategy section")
	}
}

func TestBuildReadmeArgs(t *testing.T) {
	args := buildReadmeArgs("claude-haiku-4-5")
	// Must contain -p and --print
	if args[0] != "-p" {
		t.Errorf("first arg should be -p, got %q", args[0])
	}

	// Must contain --allowedTools
	foundAllowed := false
	for i, a := range args {
		if a == "--allowedTools" {
			foundAllowed = true
			if !strings.Contains(args[i+1], "Read") {
				t.Errorf("allowedTools should contain Read, got %q", args[i+1])
			}
			if !strings.Contains(args[i+1], "Glob") {
				t.Errorf("allowedTools should contain Glob, got %q", args[i+1])
			}
		}
	}
	if !foundAllowed {
		t.Error("--allowedTools flag not found")
	}

	// Must contain --model
	foundModel := false
	for i, a := range args {
		if a == "--model" {
			foundModel = true
			if args[i+1] != "claude-haiku-4-5" {
				t.Errorf("model arg = %q, want claude-haiku-4-5", args[i+1])
			}
		}
	}
	if !foundModel {
		t.Error("--model flag not found")
	}

	// Without model
	argsNoModel := buildReadmeArgs("")
	for _, a := range argsNoModel {
		if a == "--model" {
			t.Error("--model should not be present when model is empty")
		}
	}
}
