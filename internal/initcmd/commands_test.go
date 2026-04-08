package initcmd

import (
	"bufio"
	"strings"
	"testing"
)

func TestFormatStatusConfigured(t *testing.T) {
	s := &Status{
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
		HasSkills: true,
		IsGitRepo: true,
		Source:    "github",
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
		HasSkills: true,
		IsGitRepo: true,
		Source:    "github",
	}
	if !allConfigured(githubDone) {
		t.Error("allConfigured returned false for fully configured github status")
	}

	// GitHub: missing skills
	githubPartial := &Status{
		HasSkills: false,
		IsGitRepo: true,
		Source:    "github",
	}
	if allConfigured(githubPartial) {
		t.Error("allConfigured returned true for github status missing skills")
	}
}

func TestShouldGenerateSkipsOnNo(t *testing.T) {
	input := strings.NewReader("n\n")
	opts := GenerateOpts{
		Dir:         t.TempDir(),
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	if shouldGenerate(opts, "Configure task source? [Y/n]: ") {
		t.Error("shouldGenerate returned true for 'n' input, want false")
	}
}

func TestShouldGenerateAcceptsDefault(t *testing.T) {
	input := strings.NewReader("\n")
	opts := GenerateOpts{
		Dir:         t.TempDir(),
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	if !shouldGenerate(opts, "Configure task source? [Y/n]: ") {
		t.Error("shouldGenerate returned false for empty input, want true")
	}
}

func TestShouldGenerateNonInteractiveReturnsTrue(t *testing.T) {
	opts := GenerateOpts{
		Dir:         t.TempDir(),
		Interactive: false,
	}

	if !shouldGenerate(opts, "Configure task source? [Y/n]: ") {
		t.Error("shouldGenerate returned false in non-interactive mode, want true")
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
