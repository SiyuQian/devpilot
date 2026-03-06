package generate

import (
	"strings"
	"testing"
)

func TestParseCommitPlan_ValidJSON(t *testing.T) {
	input := `{
		"commits": [
			{"message": "feat: add auth", "files": ["auth.go", "auth_test.go"]},
			{"message": "fix: typo", "files": ["readme.md"]}
		],
		"excluded": [
			{"file": ".env", "reason": "Contains secrets"}
		]
	}`
	staged := []string{"auth.go", "auth_test.go", "readme.md", ".env"}
	plan := parseCommitPlan(input, staged)

	if len(plan.Commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(plan.Commits))
	}
	if plan.Commits[0].Message != "feat: add auth" {
		t.Errorf("commit 0 message = %q", plan.Commits[0].Message)
	}
	if len(plan.Excluded) != 1 {
		t.Fatalf("expected 1 excluded, got %d", len(plan.Excluded))
	}
	if plan.Excluded[0].File != ".env" {
		t.Errorf("excluded file = %q", plan.Excluded[0].File)
	}
}

func TestParseCommitPlan_MalformedJSON(t *testing.T) {
	input := "Here is a commit message:\nfeat: add something cool"
	staged := []string{"main.go", "util.go"}
	plan := parseCommitPlan(input, staged)

	if len(plan.Commits) != 1 {
		t.Fatalf("expected 1 fallback commit, got %d", len(plan.Commits))
	}
	if len(plan.Commits[0].Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(plan.Commits[0].Files))
	}
}

func TestParseCommitPlan_EmptyCommits(t *testing.T) {
	input := `{"commits": [], "excluded": []}`
	staged := []string{"file.go"}
	plan := parseCommitPlan(input, staged)

	if len(plan.Commits) != 1 {
		t.Fatalf("expected 1 fallback commit, got %d", len(plan.Commits))
	}
}

func TestParseCommitPlan_MarkdownFences(t *testing.T) {
	input := "```json\n{\"commits\": [{\"message\": \"fix: bug\", \"files\": [\"a.go\"]}], \"excluded\": []}\n```"
	staged := []string{"a.go"}
	plan := parseCommitPlan(input, staged)

	if len(plan.Commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(plan.Commits))
	}
	if plan.Commits[0].Message != "fix: bug" {
		t.Errorf("message = %q", plan.Commits[0].Message)
	}
}

func TestValidatePlan_UnknownFile(t *testing.T) {
	plan := CommitPlan{
		Commits: []CommitEntry{
			{Message: "feat: x", Files: []string{"real.go", "ghost.go"}},
		},
	}
	staged := []string{"real.go"}
	validated, warnings := validatePlan(plan, staged)

	if len(validated.Commits[0].Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(validated.Commits[0].Files))
	}
	if len(warnings) == 0 {
		t.Error("expected warnings about unknown file")
	}
}

func TestValidatePlan_MissingFile(t *testing.T) {
	plan := CommitPlan{
		Commits: []CommitEntry{
			{Message: "feat: x", Files: []string{"a.go"}},
		},
	}
	staged := []string{"a.go", "b.go"}
	validated, warnings := validatePlan(plan, staged)

	found := false
	for _, f := range validated.Commits[0].Files {
		if f == "b.go" {
			found = true
		}
	}
	if !found {
		t.Error("expected missing file b.go to be added")
	}
	if len(warnings) == 0 {
		t.Error("expected warnings about missing file")
	}
}

func TestTruncateDiff_SmallDiff(t *testing.T) {
	diff := "diff --git a/file.go b/file.go\n--- a/file.go\n+++ b/file.go\n@@ -1 +1 @@\n-old\n+new\n"
	result := truncateDiff(diff)
	if !strings.Contains(result, "+new") {
		t.Error("small diff should be preserved")
	}
	if strings.Contains(result, "truncated") {
		t.Error("small diff should not be truncated")
	}
}

func TestTruncateDiff_LargeFile(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("diff --git a/big.go b/big.go\n")
	for i := 0; i < 300; i++ {
		sb.WriteString("+line\n")
	}
	result := truncateDiff(sb.String())
	if !strings.Contains(result, "truncated") {
		t.Error("large file diff should be truncated")
	}
}

func TestTruncateDiff_BinaryFile(t *testing.T) {
	diff := "diff --git a/image.png b/image.png\nBinary files differ\n"
	result := truncateDiff(diff)
	if !strings.Contains(result, "Binary file: image.png") {
		t.Errorf("binary file should show path, got: %s", result)
	}
}

func TestTruncateDiff_Empty(t *testing.T) {
	if truncateDiff("") != "" {
		t.Error("empty diff should return empty")
	}
}

func TestSerializeAndParsePlanRoundtrip(t *testing.T) {
	plan := CommitPlan{
		Commits: []CommitEntry{
			{Message: "feat: add auth", Files: []string{"auth.go", "auth_test.go"}},
			{Message: "fix: typo in docs", Files: []string{"README.md"}},
		},
		Excluded: []ExcludedFile{
			{File: ".env", Reason: "Contains secrets"},
		},
	}

	md := serializePlanToMarkdown(plan)
	parsed, err := parsePlanFromMarkdown(md)
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed.Commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(parsed.Commits))
	}
	if parsed.Commits[0].Message != "feat: add auth" {
		t.Errorf("commit 0 message = %q", parsed.Commits[0].Message)
	}
	if len(parsed.Commits[0].Files) != 2 {
		t.Errorf("commit 0 files count = %d", len(parsed.Commits[0].Files))
	}
	if len(parsed.Excluded) != 1 {
		t.Fatalf("expected 1 excluded, got %d", len(parsed.Excluded))
	}
	if parsed.Excluded[0].File != ".env" {
		t.Errorf("excluded file = %q", parsed.Excluded[0].File)
	}
}
