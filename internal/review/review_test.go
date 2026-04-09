package review

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

// --- Verdict parsing tests (via IsApproved on raw text) ---

func TestIsApproved_JSONApprove(t *testing.T) {
	stdout := `{"summary":"Good PR","findings":[],"assessment":"Clean code"}`
	if !IsApproved(stdout) {
		t.Error("expected IsApproved to return true for empty findings")
	}
}

func TestIsApproved_JSONWithCritical(t *testing.T) {
	stdout := `{"summary":"Has issues","findings":[{"file":"auth.go","line":42,"severity":"CRITICAL","title":"SQL injection","explanation":"bad"}],"assessment":"Needs work"}`
	if IsApproved(stdout) {
		t.Error("expected IsApproved to return false for CRITICAL finding")
	}
}

func TestIsApproved_JSONWithWarningOnly(t *testing.T) {
	stdout := `{"summary":"Minor issues","findings":[{"file":"main.go","line":10,"severity":"WARNING","title":"Missing check","explanation":"should check"}],"assessment":"Mostly good"}`
	if !IsApproved(stdout) {
		t.Error("expected IsApproved to return true for WARNING-only findings")
	}
}

func TestIsApproved_FallbackTextApprove(t *testing.T) {
	if !IsApproved("The verdict is APPROVE") {
		t.Error("expected fallback to detect APPROVE in text")
	}
}

func TestIsApproved_FallbackTextRequestChanges(t *testing.T) {
	if IsApproved("REQUEST_CHANGES needed") {
		t.Error("expected fallback to detect REQUEST_CHANGES in text")
	}
}

func TestIsApproved_EmptyOutput(t *testing.T) {
	if IsApproved("") {
		t.Error("expected IsApproved to return false for empty output")
	}
}

// --- Prompt assembly tests ---

func TestBuildPrompt_ContainsReviewInstructions(t *testing.T) {
	rc := &ReviewContext{
		Title:      "Test PR",
		Author:     "testuser",
		BaseBranch: "main",
		HeadBranch: "feature",
		Diff:       "diff content",
	}
	prompt := BuildPrompt(rc, rc.Diff)

	if !strings.Contains(prompt, "Code Review Instructions") {
		t.Error("prompt should contain review instructions")
	}
	if !strings.Contains(prompt, "Test PR") {
		t.Error("prompt should contain PR title")
	}
	if !strings.Contains(prompt, "diff content") {
		t.Error("prompt should contain diff")
	}
}

func TestBuildPrompt_NoCloneInstructions(t *testing.T) {
	rc := &ReviewContext{
		Title:      "Test PR",
		BaseBranch: "main",
		HeadBranch: "feature",
		Diff:       "diff",
	}
	prompt := BuildPrompt(rc, rc.Diff)

	if strings.Contains(prompt, "git clone") {
		t.Error("new prompt should NOT contain git clone instructions")
	}
	if strings.Contains(prompt, "Repository Setup") {
		t.Error("new prompt should NOT contain Repository Setup section")
	}
}

func TestBuildPrompt_ContainsFalsePositiveGuidance(t *testing.T) {
	rc := &ReviewContext{
		Title:      "Test PR",
		BaseBranch: "main",
		HeadBranch: "feature",
		Diff:       "diff",
	}
	prompt := BuildPrompt(rc, rc.Diff)

	if !strings.Contains(prompt, "NOT an Issue") {
		t.Error("prompt should contain false positive guidance")
	}
	if !strings.Contains(prompt, "Pre-existing issues") {
		t.Error("prompt should list pre-existing issues as false positive")
	}
}

func TestBuildPrompt_JSONOutputFormat(t *testing.T) {
	rc := &ReviewContext{
		Title:      "Test PR",
		BaseBranch: "main",
		HeadBranch: "feature",
		Diff:       "diff",
	}
	prompt := BuildPrompt(rc, rc.Diff)

	if !strings.Contains(prompt, "valid JSON") {
		t.Error("prompt should instruct JSON output")
	}
}

func TestBuildPrompt_WithConventions(t *testing.T) {
	rc := &ReviewContext{
		Title:       "Test PR",
		BaseBranch:  "main",
		HeadBranch:  "feature",
		Diff:        "diff",
		Conventions: map[string]string{"CLAUDE.md": "# My conventions"},
	}
	prompt := BuildPrompt(rc, rc.Diff)

	if !strings.Contains(prompt, "My conventions") {
		t.Error("prompt should include convention file contents")
	}
}

func TestBuildScoringPrompt(t *testing.T) {
	rc := &ReviewContext{
		Diff: "some diff",
	}
	findings := []Finding{
		{File: "main.go", Line: 10, Severity: "WARNING", Title: "Test", Explanation: "test"},
	}
	prompt := BuildScoringPrompt(rc, findings)

	if !strings.Contains(prompt, "Confidence Scoring") {
		t.Error("scoring prompt should contain scoring instructions")
	}
	if !strings.Contains(prompt, "main.go") {
		t.Error("scoring prompt should contain the finding")
	}
}

// --- Flag tests ---

func TestReviewCmd_Flags(t *testing.T) {
	parent := &cobra.Command{Use: "test"}
	RegisterCommands(parent)
	cmd, _, err := parent.Find([]string{"review"})
	if err != nil {
		t.Fatalf("could not find review command: %v", err)
	}

	// --no-post
	val, err2 := cmd.Flags().GetBool("no-post")
	if err2 != nil {
		t.Fatalf("--no-post flag should exist: %v", err2)
	}
	if val != false {
		t.Error("--no-post default should be false")
	}

	// --threshold
	threshold, err3 := cmd.Flags().GetInt("threshold")
	if err3 != nil {
		t.Fatalf("--threshold flag should exist: %v", err3)
	}
	if threshold != DefaultThreshold {
		t.Errorf("--threshold default = %d, want %d", threshold, DefaultThreshold)
	}

	// --scoring-model
	sm, err4 := cmd.Flags().GetString("scoring-model")
	if err4 != nil {
		t.Fatalf("--scoring-model flag should exist: %v", err4)
	}
	if sm != "" {
		t.Errorf("--scoring-model default should be empty, got %q", sm)
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

// --- Finding parsing tests ---

func TestParseReviewOutput_Valid(t *testing.T) {
	data := `{"summary":"Good PR","findings":[{"file":"main.go","line":10,"severity":"WARNING","title":"Test","explanation":"test"}],"assessment":"OK"}`
	out, err := ParseReviewOutput(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Summary != "Good PR" {
		t.Errorf("Summary = %q, want %q", out.Summary, "Good PR")
	}
	if len(out.Findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(out.Findings))
	}
	if out.Findings[0].File != "main.go" {
		t.Errorf("finding file = %q, want %q", out.Findings[0].File, "main.go")
	}
}

func TestParseReviewOutput_WithMarkdownFences(t *testing.T) {
	data := "```json\n{\"summary\":\"OK\",\"findings\":[],\"assessment\":\"Good\"}\n```"
	out, err := ParseReviewOutput(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Summary != "OK" {
		t.Errorf("Summary = %q, want %q", out.Summary, "OK")
	}
}

func TestParseReviewOutput_Invalid(t *testing.T) {
	_, err := ParseReviewOutput("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseScores_Valid(t *testing.T) {
	data := `[{"index":0,"score":72},{"index":1,"score":15}]`
	scores, err := ParseScores(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scores) != 2 {
		t.Fatalf("want 2 scores, got %d", len(scores))
	}
	if scores[0].Score != 72 {
		t.Errorf("score[0] = %d, want 72", scores[0].Score)
	}
}
