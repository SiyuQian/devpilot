package openspec

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// GitHubTarget implements SyncTarget using the gh CLI to manage GitHub Issues.
type GitHubTarget struct{}

// NewGitHubTarget creates a GitHubTarget.
func NewGitHubTarget() *GitHubTarget {
	return &GitHubTarget{}
}

func (g *GitHubTarget) findArgs(name string) []string {
	return []string{"issue", "list", "--label", "devpilot", "--state", "open",
		"--search", name + " in:title", "--json", "number,title", "--limit", "5"}
}

func (g *GitHubTarget) FindByName(name string) (string, error) {
	out, err := exec.Command("gh", g.findArgs(name)...).Output()
	if err != nil {
		return "", fmt.Errorf("gh issue list: %w", err)
	}
	var issues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(out, &issues); err != nil {
		return "", fmt.Errorf("parse issues: %w", err)
	}
	for _, issue := range issues {
		if issue.Title == name {
			return fmt.Sprintf("%d", issue.Number), nil
		}
	}
	return "", nil
}

func (g *GitHubTarget) Create(name, desc string) error {
	_, err := exec.Command("gh", "issue", "create",
		"--title", name, "--body", desc, "--label", "devpilot",
	).Output()
	if err != nil {
		return fmt.Errorf("create issue: %w", err)
	}
	return nil
}

func (g *GitHubTarget) Update(id, desc string) error {
	_, err := exec.Command("gh", "issue", "edit", id, "--body", desc).Output()
	if err != nil {
		return fmt.Errorf("update issue %s: %w", id, err)
	}
	return nil
}
