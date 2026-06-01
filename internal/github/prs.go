package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PR describes a pull request returned by the GitHub search command.
type PR struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	State     string    `json:"state"`
	IsDraft   bool      `json:"isDraft"`
	Author    User      `json:"author"`
	Repo      Repo      `json:"repository"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// User is a GitHub user fragment.
type User struct {
	Login string `json:"login"`
}

// Repo is a GitHub repository fragment.
type Repo struct {
	Name          string `json:"name"`
	NameWithOwner string `json:"nameWithOwner"`
}

type PRQuery struct {
	Mode          string
	User          string
	Direct        bool
	State         string
	Limit         int
	Repo          string
	Owner         string
	IncludeDrafts bool
}

func listPRs(ctx context.Context, r Runner, q PRQuery) ([]PR, error) {
	if err := checkAuth(ctx, r); err != nil {
		return nil, err
	}

	state := q.State
	if state == "" {
		state = "open"
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}

	args := []string{"search", "prs"}
	if state != "all" {
		args = append(args, "--state", state)
	}
	args = append(args, "--limit", fmt.Sprint(limit))
	args = append(args, "--json", "repository,number,title,url,author,createdAt,updatedAt,isDraft,state")
	if q.Repo != "" {
		args = append(args, "--repo", q.Repo)
	}
	if q.Owner != "" {
		args = append(args, "--owner", q.Owner)
	}

	user := q.User
	if user == "" {
		user = "@me"
	}
	switch q.Mode {
	case "review-requested":
		if q.Direct {
			args = append(args, "user-review-requested:"+user)
		} else {
			args = append(args, "--review-requested", user)
		}
	case "authored":
		args = append(args, "--author", user)
	default:
		return nil, fmt.Errorf("unknown PR query mode %q", q.Mode)
	}
	if !q.IncludeDrafts {
		args = append(args, "--draft=false")
	}

	out, err := r.Run(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("listing pull requests: %w", err)
	}
	var prs []PR
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, fmt.Errorf("decoding pull requests: %w", err)
	}
	return prs, nil
}

func prRow(pr PR) []string {
	repo := pr.Repo.NameWithOwner
	if repo == "" {
		repo = pr.Repo.Name
	}
	author := pr.Author.Login
	if author == "" {
		author = "-"
	}
	draft := ""
	if pr.IsDraft {
		draft = "draft"
	}
	return []string{
		repo,
		fmt.Sprintf("#%d", pr.Number),
		pr.State,
		draft,
		author,
		trimForTable(pr.Title, 72),
		pr.URL,
	}
}

func trimForTable(s string, max int) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
