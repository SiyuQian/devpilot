package github

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ActivityDigest is the normalized repo activity payload.
type ActivityDigest struct {
	Repo         string          `json:"repo"`
	From         time.Time       `json:"from"`
	To           time.Time       `json:"to"`
	Events       []ActivityEvent `json:"events"`
	PullRequests []PR            `json:"pullRequests"`
	Issues       []Issue         `json:"issues"`
	WorkflowRuns []WorkflowRun   `json:"workflowRuns"`
	Releases     []Release       `json:"releases"`
}

// Issue describes an issue updated in the requested window.
type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	State     string    `json:"state"`
	Author    User      `json:"author"`
	Repo      Repo      `json:"repository"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ActivityEvent is a normalized GitHub repository event.
type ActivityEvent struct {
	Type      string    `json:"type"`
	Action    string    `json:"action,omitempty"`
	Actor     string    `json:"actor"`
	CreatedAt time.Time `json:"createdAt"`
	Ref       string    `json:"ref,omitempty"`
	Number    int       `json:"number,omitempty"`
	Title     string    `json:"title,omitempty"`
	URL       string    `json:"url,omitempty"`
	Summary   string    `json:"summary"`
}

// WorkflowRun describes a GitHub Actions run in the requested window.
type WorkflowRun struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"displayTitle"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	Event       string    `json:"event"`
	Branch      string    `json:"headBranch"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Release describes a release in the requested window.
type Release struct {
	Name         string    `json:"name"`
	TagName      string    `json:"tagName"`
	IsDraft      bool      `json:"isDraft"`
	IsPrerelease bool      `json:"isPrerelease"`
	CreatedAt    time.Time `json:"createdAt"`
	PublishedAt  time.Time `json:"publishedAt"`
}

type ActivityQuery struct {
	Repo     string
	Date     string
	Since    time.Duration
	Timezone string
	Limit    int
}

type repoEvent struct {
	Type      string          `json:"type"`
	Actor     User            `json:"actor"`
	CreatedAt time.Time       `json:"created_at"`
	Payload   json.RawMessage `json:"payload"`
}

type eventPayload struct {
	Action      string `json:"action"`
	Ref         string `json:"ref"`
	RefType     string `json:"ref_type"`
	Size        int    `json:"size"`
	Number      int    `json:"number"`
	PullRequest *struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		URL    string `json:"html_url"`
		APIURL string `json:"url"`
	} `json:"pull_request"`
	Issue *struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		URL    string `json:"html_url"`
		APIURL string `json:"url"`
	} `json:"issue"`
	Comment *struct {
		URL string `json:"html_url"`
	} `json:"comment"`
	Release *struct {
		Name    string `json:"name"`
		TagName string `json:"tag_name"`
		URL     string `json:"html_url"`
	} `json:"release"`
}

func repoActivity(ctx context.Context, r Runner, q ActivityQuery) (*ActivityDigest, error) {
	if q.Repo == "" {
		return nil, fmt.Errorf("repo is required")
	}
	if err := checkAuth(ctx, r); err != nil {
		return nil, err
	}

	from, to, err := activityWindow(q)
	if err != nil {
		return nil, err
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}

	events, err := fetchEvents(ctx, r, q.Repo, from, to, limit)
	if err != nil {
		return nil, err
	}
	prs, err := fetchUpdatedPRs(ctx, r, q.Repo, from, to, limit)
	if err != nil {
		return nil, err
	}
	issues, err := fetchUpdatedIssues(ctx, r, q.Repo, from, to, limit)
	if err != nil {
		return nil, err
	}
	enrichEvents(events, prs, issues)
	runs, err := fetchWorkflowRuns(ctx, r, q.Repo, from, to, limit)
	if err != nil {
		return nil, err
	}
	releases, err := fetchReleases(ctx, r, q.Repo, from, to, limit)
	if err != nil {
		return nil, err
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})
	sort.Slice(prs, func(i, j int) bool {
		return prs[i].UpdatedAt.After(prs[j].UpdatedAt)
	})
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].UpdatedAt.After(issues[j].UpdatedAt)
	})
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].PublishedAt.After(releases[j].PublishedAt)
	})
	events = emptyIfNil(events)
	prs = emptyIfNil(prs)
	issues = emptyIfNil(issues)
	runs = emptyIfNil(runs)
	releases = emptyIfNil(releases)

	return &ActivityDigest{
		Repo:         q.Repo,
		From:         from,
		To:           to,
		Events:       events,
		PullRequests: prs,
		Issues:       issues,
		WorkflowRuns: runs,
		Releases:     releases,
	}, nil
}

func emptyIfNil[S ~[]E, E any](s S) S {
	if s == nil {
		return S{}
	}
	return s
}

func activityWindow(q ActivityQuery) (time.Time, time.Time, error) {
	loc := time.Local
	if q.Timezone != "" {
		loaded, err := time.LoadLocation(q.Timezone)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("loading timezone %q: %w", q.Timezone, err)
		}
		loc = loaded
	}
	if q.Since > 0 {
		to := time.Now().In(loc)
		return to.Add(-q.Since), to, nil
	}
	date := q.Date
	if date == "" {
		date = time.Now().In(loc).Format(time.DateOnly)
	}
	start, err := time.ParseInLocation(time.DateOnly, date, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parsing date %q: %w", date, err)
	}
	return start, start.AddDate(0, 0, 1), nil
}

func fetchEvents(ctx context.Context, r Runner, repo string, from, to time.Time, limit int) ([]ActivityEvent, error) {
	out, err := r.Run(ctx, "api", "repos/"+repo+"/events", "--paginate", "--slurp")
	if err != nil {
		return nil, fmt.Errorf("fetching repository events: %w", err)
	}
	var pages [][]repoEvent
	if err := json.Unmarshal(out, &pages); err != nil {
		return nil, fmt.Errorf("decoding repository events: %w", err)
	}

	var events []ActivityEvent
	for _, page := range pages {
		for _, ev := range page {
			if !inWindow(ev.CreatedAt, from, to) {
				continue
			}
			events = append(events, normalizeEvent(ev))
			if len(events) >= limit {
				return events, nil
			}
		}
	}
	return events, nil
}

func normalizeEvent(ev repoEvent) ActivityEvent {
	var payload eventPayload
	_ = json.Unmarshal(ev.Payload, &payload)

	out := ActivityEvent{
		Type:      ev.Type,
		Action:    payload.Action,
		Actor:     ev.Actor.Login,
		CreatedAt: ev.CreatedAt,
		Ref:       payload.Ref,
	}
	switch {
	case payload.PullRequest != nil:
		out.Number = payload.PullRequest.Number
		out.Title = payload.PullRequest.Title
		out.URL = firstNonEmpty(payload.PullRequest.URL, webURL(payload.PullRequest.APIURL))
	case payload.Issue != nil:
		out.Number = payload.Issue.Number
		out.Title = payload.Issue.Title
		out.URL = firstNonEmpty(payload.Issue.URL, webURL(payload.Issue.APIURL))
	case payload.Release != nil:
		out.Title = firstNonEmpty(payload.Release.Name, payload.Release.TagName)
		out.URL = payload.Release.URL
	}
	out.Summary = eventSummary(out, payload)
	return out
}

func eventSummary(ev ActivityEvent, payload eventPayload) string {
	switch ev.Type {
	case "PullRequestEvent":
		return fmt.Sprintf("PR #%d %s: %s", ev.Number, ev.Action, ev.Title)
	case "PullRequestReviewEvent":
		return fmt.Sprintf("PR review %s on #%d", ev.Action, ev.Number)
	case "PullRequestReviewCommentEvent":
		return fmt.Sprintf("PR review comment %s on #%d", ev.Action, ev.Number)
	case "IssuesEvent":
		return fmt.Sprintf("Issue #%d %s: %s", ev.Number, ev.Action, ev.Title)
	case "IssueCommentEvent":
		return fmt.Sprintf("Issue comment %s on #%d: %s", ev.Action, ev.Number, ev.Title)
	case "PushEvent":
		if payload.Size == 0 {
			return fmt.Sprintf("Push to %s", shortRef(ev.Ref))
		}
		return fmt.Sprintf("Push to %s (%d commit%s)", shortRef(ev.Ref), payload.Size, plural(payload.Size))
	case "CreateEvent":
		return fmt.Sprintf("Created %s %s", payload.RefType, ev.Ref)
	case "DeleteEvent":
		return fmt.Sprintf("Deleted %s %s", payload.RefType, ev.Ref)
	case "ReleaseEvent":
		return fmt.Sprintf("Release %s: %s", ev.Action, ev.Title)
	default:
		action := ev.Action
		if action == "" {
			return ev.Type
		}
		return ev.Type + " " + action
	}
}

func enrichEvents(events []ActivityEvent, prs []PR, issues []Issue) {
	prsByNumber := make(map[int]PR)
	for _, pr := range prs {
		prsByNumber[pr.Number] = pr
	}
	issuesByNumber := make(map[int]Issue)
	for _, issue := range issues {
		issuesByNumber[issue.Number] = issue
	}
	for i := range events {
		switch events[i].Type {
		case "PullRequestEvent":
			pr, ok := prsByNumber[events[i].Number]
			if !ok {
				continue
			}
			if events[i].Title == "" {
				events[i].Title = pr.Title
			}
			if events[i].URL == "" {
				events[i].URL = pr.URL
			}
			events[i].Summary = eventSummary(events[i], eventPayload{Action: events[i].Action})
		case "IssuesEvent", "IssueCommentEvent":
			issue, ok := issuesByNumber[events[i].Number]
			if !ok {
				continue
			}
			if events[i].Title == "" {
				events[i].Title = issue.Title
			}
			if events[i].URL == "" {
				events[i].URL = issue.URL
			}
			events[i].Summary = eventSummary(events[i], eventPayload{Action: events[i].Action})
		}
	}
}

func fetchUpdatedPRs(ctx context.Context, r Runner, repo string, from, to time.Time, limit int) ([]PR, error) {
	out, err := r.Run(ctx,
		"search", "prs",
		"--repo", repo,
		"--limit", fmt.Sprint(limit),
		"--json", "repository,number,title,url,author,createdAt,updatedAt,isDraft,state",
		"updated:>="+from.Format(time.DateOnly),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching updated pull requests: %w", err)
	}
	var raw []PR
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("decoding updated pull requests: %w", err)
	}
	var prs []PR
	for _, pr := range raw {
		if inWindow(pr.CreatedAt, from, to) || inWindow(pr.UpdatedAt, from, to) {
			prs = append(prs, pr)
		}
	}
	return prs, nil
}

func fetchUpdatedIssues(ctx context.Context, r Runner, repo string, from, to time.Time, limit int) ([]Issue, error) {
	out, err := r.Run(ctx,
		"search", "issues",
		"--repo", repo,
		"--limit", fmt.Sprint(limit),
		"--json", "repository,number,title,url,author,createdAt,updatedAt,state",
		"updated:>="+from.Format(time.DateOnly),
	)
	if err != nil {
		return nil, fmt.Errorf("fetching updated issues: %w", err)
	}
	var raw []Issue
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("decoding updated issues: %w", err)
	}
	var issues []Issue
	for _, issue := range raw {
		if inWindow(issue.CreatedAt, from, to) || inWindow(issue.UpdatedAt, from, to) {
			issues = append(issues, issue)
		}
	}
	return issues, nil
}

func fetchWorkflowRuns(ctx context.Context, r Runner, repo string, from, to time.Time, limit int) ([]WorkflowRun, error) {
	out, err := r.Run(ctx, "run", "list", "--repo", repo, "--limit", fmt.Sprint(limit), "--json", "name,displayTitle,status,conclusion,event,headBranch,url,createdAt,updatedAt")
	if err != nil {
		return nil, fmt.Errorf("fetching workflow runs: %w", err)
	}
	var raw []WorkflowRun
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("decoding workflow runs: %w", err)
	}
	var runs []WorkflowRun
	for _, run := range raw {
		if inWindow(run.CreatedAt, from, to) || inWindow(run.UpdatedAt, from, to) {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

func fetchReleases(ctx context.Context, r Runner, repo string, from, to time.Time, limit int) ([]Release, error) {
	out, err := r.Run(ctx, "release", "list", "--repo", repo, "--limit", fmt.Sprint(limit), "--json", "name,tagName,isDraft,isPrerelease,createdAt,publishedAt")
	if err != nil {
		return nil, fmt.Errorf("fetching releases: %w", err)
	}
	var raw []Release
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("decoding releases: %w", err)
	}
	var releases []Release
	for _, release := range raw {
		if inWindow(release.CreatedAt, from, to) || inWindow(release.PublishedAt, from, to) {
			releases = append(releases, release)
		}
	}
	return releases, nil
}

func inWindow(t, from, to time.Time) bool {
	if t.IsZero() {
		return false
	}
	return !t.Before(from) && t.Before(to)
}

func shortRef(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func webURL(apiURL string) string {
	if apiURL == "" {
		return ""
	}
	url := strings.Replace(apiURL, "https://api.github.com/repos/", "https://github.com/", 1)
	return strings.Replace(url, "/pulls/", "/pull/", 1)
}
