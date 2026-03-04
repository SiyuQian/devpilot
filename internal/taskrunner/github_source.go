package taskrunner

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitHubSource implements TaskSource using the gh CLI.
// Authentication is handled by gh (run 'gh auth login' separately).
type GitHubSource struct{}

func NewGitHubSource() *GitHubSource {
	return &GitHubSource{}
}

func (s *GitHubSource) Init() (SourceInfo, error) {
	if out, err := exec.Command("gh", "auth", "status").CombinedOutput(); err != nil {
		return SourceInfo{}, fmt.Errorf("not authenticated with GitHub CLI: run 'gh auth login'\n%s", string(out))
	}
	out, err := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner").Output()
	if err != nil {
		return SourceInfo{}, fmt.Errorf("detect repo from origin: %w (are you in a GitHub repo?)", err)
	}
	repo := strings.TrimSpace(string(out))
	return SourceInfo{DisplayName: repo}, nil
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghIssue struct {
	Number int       `json:"number"`
	Title  string    `json:"title"`
	Body   string    `json:"body"`
	URL    string    `json:"url"`
	Labels []ghLabel `json:"labels"`
}

func (s *GitHubSource) FetchReady() ([]Task, error) {
	out, err := exec.Command("gh", "issue", "list",
		"--label", "devpilot",
		"--state", "open",
		"--json", "number,title,body,url,labels",
		"--limit", "100",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("gh issue list: %w", err)
	}
	var issues []ghIssue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parse issues: %w", err)
	}
	return issuesToReadyTasks(issues), nil
}

// issuesToReadyTasks filters out in-progress and failed issues, maps the rest to Tasks.
func issuesToReadyTasks(issues []ghIssue) []Task {
	var tasks []Task
	for _, issue := range issues {
		if ghHasLabel(issue, "in-progress") || ghHasLabel(issue, "failed") {
			continue
		}
		tasks = append(tasks, Task{
			ID:          fmt.Sprintf("%d", issue.Number),
			Name:        issue.Title,
			Description: issue.Body,
			URL:         issue.URL,
			Priority:    ghPriority(issue),
		})
	}
	return tasks
}

func (s *GitHubSource) MarkInProgress(id string) error {
	_, err := exec.Command("gh", "issue", "edit", id, "--add-label", "in-progress").Output()
	return err
}

func (s *GitHubSource) MarkDone(id, comment string) error {
	if err := s.addComment(id, comment); err != nil {
		return err
	}
	_, err := exec.Command("gh", "issue", "close", id).Output()
	return err
}

func (s *GitHubSource) MarkFailed(id, comment string) error {
	_, err := exec.Command("gh", "issue", "edit", id,
		"--remove-label", "in-progress",
		"--add-label", "failed",
	).Output()
	if err != nil {
		return err
	}
	return s.addComment(id, comment)
}

func (s *GitHubSource) addComment(id, comment string) error {
	_, err := exec.Command("gh", "issue", "comment", id, "--body", comment).Output()
	return err
}

func ghHasLabel(issue ghIssue, name string) bool {
	for _, l := range issue.Labels {
		if l.Name == name {
			return true
		}
	}
	return false
}

func ghPriority(issue ghIssue) int {
	for _, l := range issue.Labels {
		name := strings.ToUpper(l.Name)
		if strings.HasPrefix(name, "P0") {
			return 0
		}
		if strings.HasPrefix(name, "P1") {
			return 1
		}
		if strings.HasPrefix(name, "P2") {
			return 2
		}
	}
	return 2
}
