package review

import (
	"fmt"
	"strings"
)

// BuildPrompt assembles the complete review prompt from instructions, template,
// and the PR URL. Context discovery is delegated to Claude via prompt instructions.
func BuildPrompt(pr *PRInfo) string {
	var b strings.Builder

	// Review instructions (includes clone + context discovery instructions)
	b.WriteString(reviewPromptMD)
	b.WriteString("\n\n")

	// Output template
	b.WriteString(reviewTemplateMD)
	b.WriteString("\n\n")

	// PR URL
	b.WriteString(fmt.Sprintf("## Task\n\nReview this pull request: %s\n", pr.URL))
	b.WriteString("\nUse `gh pr diff` and `gh pr view` to inspect the changes. Read relevant source files as needed for full context.\n")

	return b.String()
}
