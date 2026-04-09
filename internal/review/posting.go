package review

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// PostReview posts the review results to GitHub as a PR review.
func PostReview(pr *PRInfo, result *PipelineResult, diff string) error {
	ranges := parseDiffRanges(diff)

	body := formatReviewBody(result, ranges)
	event := "APPROVE"
	if result.Verdict == "REQUEST_CHANGES" {
		event = "COMMENT"
	}

	// Build gh api command
	args := []string{
		"api",
		fmt.Sprintf("repos/%s/%s/pulls/%s/reviews", pr.Owner, pr.Repo, pr.Number),
		"--method", "POST",
		"-f", fmt.Sprintf("body=%s", body),
		"-f", fmt.Sprintf("event=%s", event),
	}

	// Add inline comments for findings within diff range
	commentIdx := 0
	for _, f := range result.Findings {
		if !isLineInDiffRange(ranges, f.File, f.Line) {
			continue
		}
		prefix := fmt.Sprintf("comments[%d]", commentIdx)
		args = append(args,
			"-f", fmt.Sprintf("%s[path]=%s", prefix, f.File),
			"-f", fmt.Sprintf("%s[line]=%d", prefix, f.Line),
			"-f", fmt.Sprintf("%s[side]=RIGHT", prefix),
			"-f", fmt.Sprintf("%s[body]=%s", prefix, formatCommentBody(f)),
		)
		commentIdx++
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh api: %w\nOutput: %s", err, string(output))
	}
	return nil
}

func formatReviewBody(result *PipelineResult, ranges []DiffRange) string {
	var b strings.Builder
	b.WriteString("Here are some thoughts from my review.\n\n")
	b.WriteString(result.Summary)
	b.WriteString("\n\n")

	if result.Verdict == "APPROVE" {
		b.WriteString("**Verdict: APPROVED**\n\n")
	} else {
		b.WriteString("**Verdict: NEEDS ATTENTION**\n\n")
	}

	// Include findings outside diff range in body
	for _, f := range result.Findings {
		if !isLineInDiffRange(ranges, f.File, f.Line) {
			fmt.Fprintf(&b, "**[%s]** `%s` Line %d: %s\n\n%s\n\n", f.Severity, f.File, f.Line, f.Title, f.Explanation)
		}
	}

	b.WriteString(result.Assessment)
	b.WriteString("\n\n— Automated review by DevPilot")

	return b.String()
}

func formatCommentBody(f ScoredFinding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s (confidence: %d/100)\n\n%s", f.Severity, f.Title, f.Score, f.Explanation)
	if f.Suggestion != "" {
		b.WriteString("\n\n```suggestion\n")
		b.WriteString(f.Suggestion)
		b.WriteString("\n```")
	}
	return b.String()
}

// DiffRange represents a range of new-side line numbers in a diff hunk.
type DiffRange struct {
	File      string
	StartLine int
	LineCount int
}

// parseDiffRanges extracts valid new-side line ranges from diff hunk headers.
func parseDiffRanges(diff string) []DiffRange {
	var ranges []DiffRange
	var currentFile string

	for _, line := range strings.Split(diff, "\n") {
		if after, ok := strings.CutPrefix(line, "+++ b/"); ok {
			currentFile = after
			continue
		}
		if currentFile != "" && strings.HasPrefix(line, "@@") {
			start, count := parseHunkHeader(line)
			if start > 0 {
				ranges = append(ranges, DiffRange{
					File:      currentFile,
					StartLine: start,
					LineCount: count,
				})
			}
		}
	}
	return ranges
}

// parseHunkHeader extracts the new-side start line and count from a @@ line.
// Format: @@ -old,count +new,count @@
func parseHunkHeader(line string) (start, count int) {
	plusIdx := strings.Index(line, "+")
	if plusIdx < 0 {
		return 0, 0
	}
	rest := line[plusIdx+1:]
	spaceIdx := strings.Index(rest, " ")
	if spaceIdx < 0 {
		// Try @@ at end
		atIdx := strings.Index(rest, "@@")
		if atIdx < 0 {
			return 0, 0
		}
		rest = rest[:atIdx]
	} else {
		rest = rest[:spaceIdx]
	}
	rest = strings.TrimSpace(rest)

	parts := strings.SplitN(rest, ",", 2)
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0
	}
	count = 1
	if len(parts) == 2 {
		count, err = strconv.Atoi(parts[1])
		if err != nil {
			count = 1
		}
	}
	return start, count
}

// isLineInDiffRange checks if a line number falls within any diff hunk for the file.
func isLineInDiffRange(ranges []DiffRange, file string, line int) bool {
	if ranges == nil {
		return false
	}
	for _, r := range ranges {
		if r.File == file && line >= r.StartLine && line < r.StartLine+r.LineCount {
			return true
		}
	}
	return false
}

//go:embed review-posting-fallback.md
var reviewPostingFallbackMD string

// postReviewFallback invokes Haiku to adaptively post the review when Go posting fails.
func postReviewFallback(ctx context.Context, pr *PRInfo, result *PipelineResult, diff string, originalErr error) error {
	findingsJSON, _ := json.MarshalIndent(result.Findings, "", "  ")

	prompt := fmt.Sprintf("%s\n\n---\n\n## Error from Go posting attempt\n\n```\n%s\n```\n\n## PR\n\n%s\n\n## Verdict\n\n%s\n\n## Summary\n\n%s\n\n## Assessment\n\n%s\n\n## Findings\n\n%s\n\n## Diff\n\n```\n%s\n```\n",
		reviewPostingFallbackMD,
		originalErr.Error(),
		pr.URL,
		result.Verdict,
		result.Summary,
		result.Assessment,
		string(findingsJSON),
		diff,
	)

	exec := newRoundExecutor(DefaultScoringModel, nil, nil) // Haiku
	res, err := exec.Run(ctx, prompt)
	if err != nil {
		return fmt.Errorf("fallback execution: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("fallback exited with code %d: %s", res.ExitCode, res.Stderr)
	}
	return nil
}
