package review

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildPrompt assembles the Round 1 review prompt with pre-gathered context.
func BuildPrompt(rc *ReviewContext, diff string) string {
	var b strings.Builder

	b.WriteString(reviewPromptMD)
	b.WriteString("\n\n---\n\n")

	// PR metadata
	b.WriteString("## PR Metadata\n\n")
	fmt.Fprintf(&b, "**Title:** %s\n", rc.Title)
	fmt.Fprintf(&b, "**Author:** %s\n", rc.Author)
	fmt.Fprintf(&b, "**Base:** %s ← **Head:** %s\n\n", rc.BaseBranch, rc.HeadBranch)

	if rc.Body != "" {
		b.WriteString("### Description\n\n")
		b.WriteString(rc.Body)
		b.WriteString("\n\n")
	}

	// Project conventions
	b.WriteString("## Project Conventions\n\n")
	b.WriteString(rc.ConventionsText())
	b.WriteString("\n\n")

	// Diff
	b.WriteString("## Diff\n\n```\n")
	b.WriteString(diff)
	b.WriteString("\n```\n")

	return b.String()
}

// BuildScoringPrompt assembles the Round 2 scoring prompt.
func BuildScoringPrompt(rc *ReviewContext, findings []Finding) string {
	var b strings.Builder

	b.WriteString(reviewScoringMD)
	b.WriteString("\n\n---\n\n")

	// Diff context for the scorer
	b.WriteString("## PR Diff\n\n```\n")
	b.WriteString(rc.Diff)
	b.WriteString("\n```\n\n")

	// Project conventions
	b.WriteString("## Project Conventions\n\n")
	b.WriteString(rc.ConventionsText())
	b.WriteString("\n\n")

	// Findings to score
	b.WriteString("## Findings to Score\n\n")
	findingsJSON, _ := json.MarshalIndent(findings, "", "  ")
	b.Write(findingsJSON)
	b.WriteByte('\n')

	return b.String()
}
