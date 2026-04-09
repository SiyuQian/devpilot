package review

import (
	"strings"
	"testing"
)

// --- URL validation tests ---

func TestParsePRURL_Valid(t *testing.T) {
	pr, err := ParsePRURL("https://github.com/owner/repo/pull/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.Owner != "owner" {
		t.Errorf("Owner = %q, want %q", pr.Owner, "owner")
	}
	if pr.Repo != "repo" {
		t.Errorf("Repo = %q, want %q", pr.Repo, "repo")
	}
	if pr.Number != "123" {
		t.Errorf("Number = %q, want %q", pr.Number, "123")
	}
}

func TestParsePRURL_WithTrailingPath(t *testing.T) {
	pr, err := ParsePRURL("https://github.com/org/project/pull/456/files")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.Owner != "org" || pr.Repo != "project" || pr.Number != "456" {
		t.Errorf("got %+v", pr)
	}
}

func TestParsePRURL_Invalid(t *testing.T) {
	tests := []string{
		"not-a-url",
		"https://github.com/owner/repo",
		"https://github.com/owner/repo/issues/123",
		"https://gitlab.com/owner/repo/pull/123",
		"",
	}
	for _, url := range tests {
		_, err := ParsePRURL(url)
		if err == nil {
			t.Errorf("ParsePRURL(%q) should return error", url)
		}
	}
}

// --- Verdict parsing tests ---

func TestIsApproved_Approve(t *testing.T) {
	stdout := `## Summary

Good PR.

## Verdict

APPROVE

No blocking issues found.

## Findings

No findings.
`
	if !IsApproved(stdout) {
		t.Error("expected IsApproved to return true for APPROVE verdict")
	}
}

func TestIsApproved_RequestChanges(t *testing.T) {
	stdout := `## Summary

Has issues.

## Verdict

REQUEST_CHANGES

Must fix the SQL injection in auth.go.

## Findings

### auth.go

**[CRITICAL]** Line 42: SQL injection
`
	if IsApproved(stdout) {
		t.Error("expected IsApproved to return false for REQUEST_CHANGES verdict")
	}
}

func TestIsApproved_NoVerdictSection(t *testing.T) {
	if IsApproved("just some random text") {
		t.Error("expected IsApproved to return false when no verdict section")
	}
}

func TestIsApproved_EmptyOutput(t *testing.T) {
	if IsApproved("") {
		t.Error("expected IsApproved to return false for empty output")
	}
}

func TestIsApproved_MalformedVerdict(t *testing.T) {
	stdout := `## Verdict

something unexpected here
`
	if IsApproved(stdout) {
		t.Error("expected IsApproved to return false for malformed verdict")
	}
}

// --- Prompt assembly tests ---

func TestBuildPrompt_ContainsReviewInstructions(t *testing.T) {
	pr := &PRInfo{Owner: "owner", Repo: "repo", Number: "1", URL: "https://github.com/owner/repo/pull/1"}

	prompt := BuildPrompt(pr)

	if !strings.Contains(prompt, "Code Review Instructions") {
		t.Error("prompt should contain review instructions")
	}
	if !strings.Contains(prompt, "https://github.com/owner/repo/pull/1") {
		t.Error("prompt should contain PR URL")
	}
	if !strings.Contains(prompt, "Review Output Format") {
		t.Error("prompt should contain review template")
	}
}

func TestBuildPrompt_ContainsCloneInstructions(t *testing.T) {
	pr := &PRInfo{Owner: "owner", Repo: "repo", Number: "1", URL: "https://github.com/owner/repo/pull/1"}

	prompt := BuildPrompt(pr)

	if !strings.Contains(prompt, "Repository Setup") {
		t.Error("prompt should contain repository setup section")
	}
	if !strings.Contains(prompt, "git clone") {
		t.Error("prompt should contain git clone instructions")
	}
	if !strings.Contains(prompt, "Project Context Discovery") {
		t.Error("prompt should contain context discovery section")
	}
	if !strings.Contains(prompt, "CLAUDE.md") {
		t.Error("prompt should mention CLAUDE.md as a convention file to look for")
	}
}

// --- Fix prompt tests ---

func TestBuildFixPrompt(t *testing.T) {
	prompt := BuildFixPrompt("https://github.com/owner/repo/pull/42")
	if !strings.Contains(prompt, "https://github.com/owner/repo/pull/42") {
		t.Error("fix prompt should contain PR URL")
	}
	if !strings.Contains(prompt, "Fix") {
		t.Error("fix prompt should instruct to fix")
	}
}
