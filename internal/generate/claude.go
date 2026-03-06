package generate

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Generate calls `claude --print` with the given prompt and optional model.
func Generate(ctx context.Context, prompt, model string) (string, error) {
	args := buildArgs(model)
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, "claude", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude failed: %w\nstderr: %s", err, stderr.String())
	}

	return cleanOutput(stdout.String()), nil
}

func buildArgs(model string) []string {
	args := []string{"--print"}
	if model != "" {
		args = append(args, "--model", model)
	}
	return args
}

var preambleRe = regexp.MustCompile(`(?i)^(here('s| is).*?:)\s*\n`)

func cleanOutput(s string) string {
	s = strings.TrimSpace(s)
	// Strip markdown code fences
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	// Strip language hints after opening fence
	if idx := strings.Index(s, "\n"); idx >= 0 && !strings.Contains(s[:idx], " ") && len(s[:idx]) < 15 {
		first := strings.TrimSpace(s[:idx])
		if first == "markdown" || first == "text" || first == "json" || first == "" {
			s = s[idx+1:]
		}
	}
	// Strip AI preamble like "Here is the commit message:"
	s = preambleRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
