package initcmd

import "testing"

func TestFormatStatusConfigured(t *testing.T) {
	s := &Status{
		HasClaudeMD:    true,
		HasTrelloCreds: true,
		HasBoardConfig: true,
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

func TestFormatStatusGitHub(t *testing.T) {
	s := &Status{
		HasClaudeMD: true,
		HasSkills:   true,
		IsGitRepo:   true,
		Source:      "github",
	}

	lines := formatStatus(s)

	foundGitHub := false
	for _, line := range lines {
		if containsSubstring(line, "GitHub Issues") {
			foundGitHub = true
		}
	}
	if !foundGitHub {
		t.Error("expected GitHub Issues in status lines for github source")
	}

	// Should NOT mention Trello when source is github
	for _, line := range lines {
		if containsSubstring(line, "Trello") {
			t.Errorf("unexpected Trello mention in github source status: %s", line)
		}
	}
}

func TestFormatStatusMissing(t *testing.T) {
	s := &Status{
		HasClaudeMD:    false,
		HasTrelloCreds: false,
		HasBoardConfig: false,
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
	// Trello: fully configured
	allDone := &Status{
		HasClaudeMD:    true,
		HasTrelloCreds: true,
		HasBoardConfig: true,
		HasSkills:      true,
		IsGitRepo:      true,
	}
	if !allConfigured(allDone) {
		t.Error("allConfigured returned false for fully configured trello status")
	}

	// Trello: missing board
	partial := &Status{
		HasClaudeMD:    true,
		HasTrelloCreds: true,
		HasBoardConfig: false,
		HasSkills:      true,
		IsGitRepo:      true,
	}
	if allConfigured(partial) {
		t.Error("allConfigured returned true for trello status missing board")
	}

	// GitHub: fully configured (no Trello creds needed)
	githubDone := &Status{
		HasClaudeMD: true,
		HasSkills:   true,
		IsGitRepo:   true,
		Source:      "github",
	}
	if !allConfigured(githubDone) {
		t.Error("allConfigured returned false for fully configured github status")
	}

	// GitHub: missing CLAUDE.md
	githubPartial := &Status{
		HasClaudeMD: false,
		HasSkills:   true,
		IsGitRepo:   true,
		Source:      "github",
	}
	if allConfigured(githubPartial) {
		t.Error("allConfigured returned true for github status missing CLAUDE.md")
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
