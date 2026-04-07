package review

import (
	"fmt"
	"strings"
)

// BuildPrompt assembles the complete review prompt from instructions, template,
// project context, and the PR URL.
func BuildPrompt(pr *PRInfo, ctx *ProjectContext) string {
	var b strings.Builder

	// Review instructions
	b.WriteString(reviewPromptMD)
	b.WriteString("\n\n")

	// Project context (if any)
	if ctx != nil && len(ctx.Conventions) > 0 {
		b.WriteString("## Project Context\n\n")
		b.WriteString("The following convention files were detected in the target repository. Use them to inform your review:\n\n")
		for _, cf := range ctx.Conventions {
			b.WriteString(fmt.Sprintf("### %s\n\n", cf.Description))
			b.WriteString("```\n")
			b.WriteString(cf.Content)
			if !strings.HasSuffix(cf.Content, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("```\n\n")
		}
	}

	// Output template
	b.WriteString(reviewTemplateMD)
	b.WriteString("\n\n")

	// PR URL
	b.WriteString(fmt.Sprintf("## Task\n\nReview this pull request: %s\n", pr.URL))
	b.WriteString("\nUse `gh pr diff` and `gh pr view` to inspect the changes. Read relevant source files as needed for full context.\n")

	return b.String()
}
