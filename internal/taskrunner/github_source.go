package taskrunner

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	ghLabelDevpilot   = "devpilot"
	ghLabelInProgress = "in-progress"
	ghLabelFailed     = "failed"
)

// GitHubSource implements TaskSource using the gh CLI.
// Authentication is handled by gh (run 'gh auth login' separately).
type GitHubSource struct{}

func NewGitHubSource() *GitHubSource {
	return &GitHubSource{}
}

func (s *GitHubSource) Init() (SourceInfo, error) {
	out, err := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner").Output()
	if err != nil {
		return SourceInfo{}, fmt.Errorf("detect repo: %w (run 'gh auth login' if not authenticated)", err)
	}
	repo := strings.TrimSpace(string(out))
	return SourceInfo{DisplayName: repo}, nil
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	URL       string    `json:"url"`
	Labels    []ghLabel `json:"labels"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *GitHubSource) FetchReady() ([]Task, error) {
	// "sort:created-asc" asks the GitHub API to return issues oldest-first.
	// Combined with the stable priority sort in SortByPriority, this gives a
	// deterministic FIFO queue within each priority tier — without requiring
	// any extra configuration from the user.
	out, err := exec.Command("gh", "issue", "list",
		"--label", ghLabelDevpilot,
		"--state", "open",
		"--search", "sort:created-asc",
		"--json", "number,title,body,url,labels,createdAt",
		"--limit", "25",
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
		if ghHasLabel(issue, ghLabelInProgress) || ghHasLabel(issue, ghLabelFailed) {
			continue
		}
		tasks = append(tasks, Task{
			ID:          fmt.Sprintf("%d", issue.Number),
			Name:        issue.Title,
			Description: issue.Body,
			URL:         issue.URL,
			Priority:    ghPriority(issue),
			CreatedAt:   issue.CreatedAt.Unix(),
		})
	}
	return tasks
}

func (s *GitHubSource) MarkInProgress(id string) error {
	_, err := exec.Command("gh", "issue", "edit", id, "--add-label", ghLabelInProgress).Output()
	if err != nil {
		return fmt.Errorf("add in-progress label to issue %s: %w", id, err)
	}
	return nil
}

func (s *GitHubSource) MarkDone(id, comment string) error {
	_, err := exec.Command("gh", "issue", "close", id).Output()
	if err != nil {
		return fmt.Errorf("close issue %s: %w", id, err)
	}
	return s.addComment(id, comment)
}

func (s *GitHubSource) MarkFailed(id, comment string) error {
	_, err := exec.Command("gh", "issue", "edit", id,
		"--remove-label", ghLabelInProgress,
		"--add-label", ghLabelFailed,
	).Output()
	if err != nil {
		return fmt.Errorf("update labels on issue %s: %w", id, err)
	}
	return s.addComment(id, comment)
}

func (s *GitHubSource) addComment(id, comment string) error {
	_, err := exec.Command("gh", "issue", "comment", id, "--body", comment).Output()
	if err != nil {
		return fmt.Errorf("add comment to issue %s: %w", id, err)
	}
	return nil
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
	names := make([]string, len(issue.Labels))
	for i, l := range issue.Labels {
		names[i] = l.Name
	}
	return priorityFromLabelNames(names)
}
