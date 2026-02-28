package initcmd

import "testing"

func TestFormatStatusConfigured(t *testing.T) {
	s := &Status{
		HasClaudeMD:    true,
		HasTrelloCreds: true,
		HasBoardConfig: true,
		HasGitHooks:    true,
		HasSkills:      true,
		IsGitRepo:      true,
	}

	lines := formatStatus(s)

	expected := []struct {
		prefix string
		label  string
	}{
		{"✓", "CLAUDE.md"},
		{"✓", "Trello board configured"},
		{"✓", "Trello credentials"},
		{"✓", "Git hooks"},
		{"✓", "Skills"},
	}

	if len(lines) != len(expected) {
		t.Fatalf("got %d lines, want %d", len(lines), len(expected))
	}

	for i, exp := range expected {
		if !containsSubstring(lines[i], exp.prefix) {
			t.Errorf("line %d missing prefix %q: %s", i, exp.prefix, lines[i])
		}
		if !containsSubstring(lines[i], exp.label) {
			t.Errorf("line %d missing label %q: %s", i, exp.label, lines[i])
		}
	}
}

func TestFormatStatusMissing(t *testing.T) {
	s := &Status{
		HasClaudeMD:    false,
		HasTrelloCreds: false,
		HasBoardConfig: false,
		HasGitHooks:    false,
		HasSkills:      false,
		IsGitRepo:      true,
	}

	lines := formatStatus(s)

	for _, line := range lines {
		if containsSubstring(line, "✓") {
			t.Errorf("expected all ✗ but got ✓ in line: %s", line)
		}
	}
}

func TestFormatStatusNotGitRepo(t *testing.T) {
	s := &Status{
		IsGitRepo: false,
	}

	lines := formatStatus(s)

	foundGitWarning := false
	for _, line := range lines {
		if containsSubstring(line, "Not a git repository") {
			foundGitWarning = true
		}
	}
	if !foundGitWarning {
		t.Error("expected git repo warning in status lines")
	}
}

func TestAllConfigured(t *testing.T) {
	allDone := &Status{
		HasClaudeMD:    true,
		HasTrelloCreds: true,
		HasBoardConfig: true,
		HasGitHooks:    true,
		HasSkills:      true,
		IsGitRepo:      true,
	}
	if !allConfigured(allDone) {
		t.Error("allConfigured returned false for fully configured status")
	}

	partial := &Status{
		HasClaudeMD:    true,
		HasTrelloCreds: true,
		HasBoardConfig: false,
		HasGitHooks:    true,
		HasSkills:      true,
		IsGitRepo:      true,
	}
	if allConfigured(partial) {
		t.Error("allConfigured returned true for partial status")
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
