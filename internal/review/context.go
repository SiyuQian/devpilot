package review

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReviewContext holds all pre-gathered context needed for a review.
type ReviewContext struct {
	// PR metadata
	Title      string
	Body       string
	Author     string
	BaseBranch string
	HeadBranch string

	// Diff text
	Diff string

	// Project conventions (file name → content)
	Conventions map[string]string
}

// prViewJSON matches the JSON output of gh pr view.
type prViewJSON struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
}

// GatherContext collects PR metadata, diff, and project conventions.
func GatherContext(pr *PRInfo) (*ReviewContext, error) {
	// Get PR metadata
	meta, err := fetchPRMetadata(pr)
	if err != nil {
		return nil, fmt.Errorf("fetch PR metadata: %w", err)
	}

	// Get diff
	diff, err := fetchDiff(pr)
	if err != nil {
		return nil, fmt.Errorf("fetch diff: %w", err)
	}

	// Get conventions
	conventions := gatherConventions(pr, meta.BaseRefName)

	return &ReviewContext{
		Title:       meta.Title,
		Body:        meta.Body,
		Author:      meta.Author.Login,
		BaseBranch:  meta.BaseRefName,
		HeadBranch:  meta.HeadRefName,
		Diff:        diff,
		Conventions: conventions,
	}, nil
}

func fetchPRMetadata(pr *PRInfo) (*prViewJSON, error) {
	out, err := exec.Command("gh", "pr", "view", pr.URL,
		"--json", "title,body,baseRefName,headRefName,author").Output()
	if err != nil {
		return nil, fmt.Errorf("gh pr view: %w", err)
	}
	var meta prViewJSON
	if err := json.Unmarshal(out, &meta); err != nil {
		return nil, fmt.Errorf("parse pr metadata: %w", err)
	}
	return &meta, nil
}

func fetchDiff(pr *PRInfo) (string, error) {
	out, err := exec.Command("gh", "pr", "diff", pr.URL).Output()
	if err != nil {
		return "", fmt.Errorf("gh pr diff: %w", err)
	}
	return string(out), nil
}

// conventionFiles lists filenames to look for in the repo root.
var conventionFiles = []string{
	"CLAUDE.md",
	"AGENTS.md",
	"CONTRIBUTING.md",
	".golangci.yml",
	".golangci.yaml",
	".eslintrc.js",
	".eslintrc.json",
	".eslintrc.yml",
	"eslint.config.js",
	"eslint.config.mjs",
	"pyproject.toml",
	".editorconfig",
	".prettierrc",
	".prettierrc.json",
	".prettierrc.yml",
}

// gatherConventions tries local disk first (if cwd matches the PR repo),
// then falls back to GitHub API.
func gatherConventions(pr *PRInfo, baseBranch string) map[string]string {
	conventions := make(map[string]string)

	if isLocalRepo(pr) {
		for _, name := range conventionFiles {
			content, err := os.ReadFile(name)
			if err == nil {
				conventions[name] = string(content)
			}
		}
		return conventions
	}

	// Fetch via GitHub API
	for _, name := range conventionFiles {
		content, err := fetchFileFromGitHub(pr, name, baseBranch)
		if err == nil {
			conventions[name] = content
		}
	}
	return conventions
}

// isLocalRepo checks if the current directory is a checkout of the PR's repo.
func isLocalRepo(pr *PRInfo) bool {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return false
	}
	remoteURL := strings.TrimSpace(string(out))
	repoSuffix := pr.Owner + "/" + pr.Repo
	return strings.Contains(remoteURL, repoSuffix)
}

// ghContentJSON matches the GitHub API contents response.
type ghContentJSON struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func fetchFileFromGitHub(pr *PRInfo, path, ref string) (string, error) {
	apiPath := fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", pr.Owner, pr.Repo, path, ref)
	out, err := exec.Command("gh", "api", apiPath).Output()
	if err != nil {
		return "", err
	}

	// Handle directory listings (returns array) — skip
	trimmed := strings.TrimSpace(string(out))
	if strings.HasPrefix(trimmed, "[") {
		return "", fmt.Errorf("path is a directory")
	}

	var content ghContentJSON
	if err := json.Unmarshal(out, &content); err != nil {
		return "", err
	}
	if content.Encoding != "base64" {
		return "", fmt.Errorf("unexpected encoding: %s", content.Encoding)
	}
	decoded, err := base64.StdEncoding.DecodeString(
		strings.ReplaceAll(content.Content, "\n", ""))
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// ConventionsText formats the conventions map into a string for prompt injection.
func (rc *ReviewContext) ConventionsText() string {
	if len(rc.Conventions) == 0 {
		return "No project conventions detected."
	}
	var b strings.Builder
	for name, content := range rc.Conventions {
		fmt.Fprintf(&b, "### %s\n\n", name)
		// Truncate very large config files
		if len(content) > 5000 {
			content = content[:5000] + "\n... (truncated)"
		}
		b.WriteString(content)
		b.WriteString("\n\n")
	}
	return b.String()
}

// FilesInDiff extracts the list of file paths from a unified diff.
func FilesInDiff(diff string) []string {
	var files []string
	for line := range strings.SplitSeq(diff, "\n") {
		if after, ok := strings.CutPrefix(line, "+++ b/"); ok {
			files = append(files, after)
		}
	}
	return files
}

// DiffDir returns the convention file path resolved relative to the repo root.
func DiffDir() string {
	dir, err := filepath.Abs(".")
	if err != nil {
		return "."
	}
	return dir
}
