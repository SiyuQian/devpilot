package review

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// conventionFiles lists the files to check for project conventions.
var conventionFiles = []struct {
	path        string
	description string
}{
	{"CLAUDE.md", "Project coding conventions (CLAUDE.md)"},
	{"AGENTS.md", "Agent instructions (AGENTS.md)"},
	{".golangci.yml", "Go linter config (golangci-lint)"},
	{".golangci.yaml", "Go linter config (golangci-lint)"},
	{".eslintrc.json", "JavaScript/TypeScript linter config (ESLint)"},
	{".eslintrc.js", "JavaScript/TypeScript linter config (ESLint)"},
	{".eslintrc.yml", "JavaScript/TypeScript linter config (ESLint)"},
	{"eslint.config.js", "JavaScript/TypeScript linter config (ESLint flat config)"},
	{"pyproject.toml", "Python project config (may contain linter settings)"},
}

// ProjectContext holds gathered context about the target repository.
type ProjectContext struct {
	Conventions []ConventionFile
}

// ConventionFile represents a detected convention file and its content.
type ConventionFile struct {
	Path        string
	Description string
	Content     string
}

// GatherContext collects project conventions from the target repository.
// It first checks if the local working directory matches the PR's repo,
// then falls back to fetching via GitHub API.
func GatherContext(pr *PRInfo) *ProjectContext {
	ctx := &ProjectContext{}

	// Check if cwd is a local checkout of the same repo
	if isLocalCheckout(pr) {
		ctx.Conventions = gatherFromLocal()
		return ctx
	}

	// Fall back to GitHub API
	ctx.Conventions = gatherFromGitHub(pr)
	return ctx
}

// isLocalCheckout checks if the cwd is a git repo matching the PR's owner/repo.
func isLocalCheckout(pr *PRInfo) bool {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	remote := strings.TrimSpace(string(out))
	// Match both HTTPS and SSH formats
	return strings.Contains(remote, fmt.Sprintf("github.com/%s/%s", pr.Owner, pr.Repo)) ||
		strings.Contains(remote, fmt.Sprintf("github.com:%s/%s", pr.Owner, pr.Repo))
}

// gatherFromLocal reads convention files from the current working directory.
func gatherFromLocal() []ConventionFile {
	var found []ConventionFile
	for _, cf := range conventionFiles {
		content, err := os.ReadFile(cf.path)
		if err != nil {
			continue
		}
		found = append(found, ConventionFile{
			Path:        cf.path,
			Description: cf.description,
			Content:     string(content),
		})
	}
	return found
}

// gatherFromGitHub fetches convention files via gh CLI from the PR's base branch.
func gatherFromGitHub(pr *PRInfo) []ConventionFile {
	// Get the base branch
	baseBranch := getBaseBranch(pr)
	if baseBranch == "" {
		baseBranch = "main"
	}

	var found []ConventionFile
	for _, cf := range conventionFiles {
		content := fetchFileFromGitHub(pr, cf.path, baseBranch)
		if content == "" {
			continue
		}
		found = append(found, ConventionFile{
			Path:        cf.path,
			Description: cf.description,
			Content:     content,
		})
	}
	return found
}

// getBaseBranch retrieves the PR's base branch via gh CLI.
func getBaseBranch(pr *PRInfo) string {
	cmd := exec.Command("gh", "pr", "view", pr.URL, "--json", "baseRefName", "--jq", ".baseRefName")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// fetchFileFromGitHub fetches a single file's content from a GitHub repo via gh CLI.
func fetchFileFromGitHub(pr *PRInfo, path, ref string) string {
	// Use gh api to fetch raw file content
	apiPath := filepath.ToSlash(fmt.Sprintf("repos/%s/%s/contents/%s", pr.Owner, pr.Repo, path))
	cmd := exec.Command("gh", "api", apiPath,
		"--header", "Accept: application/vnd.github.raw+json",
		"-f", fmt.Sprintf("ref=%s", ref))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return ""
	}
	return stdout.String()
}
