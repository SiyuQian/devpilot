package openspec

import "testing"

// Compile-time interface compliance check.
var _ SyncTarget = (*GitHubTarget)(nil)

func TestGitHubTarget_interfaceCompliance(t *testing.T) {
	// The var _ line above ensures GitHubTarget implements SyncTarget.
	// This test exists to make the check explicit and visible in test output.
	g := NewGitHubTarget()
	if g == nil {
		t.Fatal("expected non-nil GitHubTarget")
	}
}

func TestGitHubTarget_buildFindCommand(t *testing.T) {
	g := NewGitHubTarget()
	args := g.findArgs("add-auth")

	expected := []string{
		"issue", "list", "--label", "devpilot", "--state", "open",
		"--search", "add-auth in:title", "--json", "number,title", "--limit", "5",
	}

	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}

	for i, want := range expected {
		if args[i] != want {
			t.Errorf("arg[%d]: expected %q, got %q", i, want, args[i])
		}
	}
}
