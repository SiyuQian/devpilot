package github

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	calls     [][]string
	responses map[string]string
	errs      map[string]error
}

func (r *fakeRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	key := strings.Join(args, "\x00")
	if err := r.errs[key]; err != nil {
		return nil, err
	}
	if out, ok := r.responses[key]; ok {
		return []byte(out), nil
	}
	return []byte("[]"), nil
}

func TestListPRsReviewQueueBuildsDirectQuery(t *testing.T) {
	r := &fakeRunner{
		responses: map[string]string{
			"auth\x00status": "ok",
			"search\x00prs\x00--state\x00open\x00--limit\x0050\x00--json\x00repository,number,title,url,author,createdAt,updatedAt,isDraft,state\x00--repo\x00SiyuQian/devpilot\x00user-review-requested:@me\x00--draft=false": `[{
				"number": 147,
				"title": "docs: sync learn skill",
				"url": "https://github.com/SiyuQian/devpilot/pull/147",
				"state": "open",
				"isDraft": false,
				"author": {"login": "SiyuQian"},
				"repository": {"nameWithOwner": "SiyuQian/devpilot"},
				"createdAt": "2026-06-01T01:00:00Z",
				"updatedAt": "2026-06-01T02:00:00Z"
			}]`,
		},
	}

	prs, err := listPRs(context.Background(), r, PRQuery{
		Mode:   "review-requested",
		Direct: true,
		Repo:   "SiyuQian/devpilot",
		Limit:  50,
	})
	if err != nil {
		t.Fatalf("listPRs() error = %v", err)
	}
	if len(prs) != 1 {
		t.Fatalf("len(prs) = %d, want 1", len(prs))
	}
	if prs[0].Repo.NameWithOwner != "SiyuQian/devpilot" {
		t.Errorf("repo = %q, want SiyuQian/devpilot", prs[0].Repo.NameWithOwner)
	}
	got := strings.Join(r.calls[1], " ")
	if !strings.Contains(got, "user-review-requested:@me") {
		t.Errorf("search args = %q, want direct review qualifier", got)
	}
	if strings.Contains(got, "--review-requested") {
		t.Errorf("search args = %q, should not include --review-requested for direct mode", got)
	}
}

func TestListPRsAuthoredBuildsAuthorQuery(t *testing.T) {
	r := &fakeRunner{responses: map[string]string{
		"auth\x00status": "ok",
		"search\x00prs\x00--limit\x00200\x00--json\x00repository,number,title,url,author,createdAt,updatedAt,isDraft,state\x00--owner\x00SiyuQian\x00--author\x00octocat\x00--draft=false": "[]",
	}}

	_, err := listPRs(context.Background(), r, PRQuery{
		Mode:  "authored",
		User:  "octocat",
		State: "all",
		Owner: "SiyuQian",
	})
	if err != nil {
		t.Fatalf("listPRs() error = %v", err)
	}
	got := strings.Join(r.calls[1], " ")
	for _, want := range []string{"--author octocat", "--owner SiyuQian"} {
		if !strings.Contains(got, want) {
			t.Errorf("search args = %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "--state") {
		t.Errorf("search args = %q, should omit --state for all", got)
	}
}

func TestActivityWindowUsesTimezoneDate(t *testing.T) {
	from, to, err := activityWindow(ActivityQuery{
		Date:     "2026-06-01",
		Timezone: "Pacific/Auckland",
	})
	if err != nil {
		t.Fatalf("activityWindow() error = %v", err)
	}
	if got := from.Format(time.RFC3339); got != "2026-06-01T00:00:00+12:00" {
		t.Errorf("from = %s, want 2026-06-01T00:00:00+12:00", got)
	}
	if got := to.Format(time.RFC3339); got != "2026-06-02T00:00:00+12:00" {
		t.Errorf("to = %s, want 2026-06-02T00:00:00+12:00", got)
	}
}

func TestRepoActivityFiltersAndNormalizesSources(t *testing.T) {
	events := `[[{
		"type": "PullRequestEvent",
		"actor": {"login": "SiyuQian"},
		"created_at": "2026-06-01T06:24:14Z",
		"payload": {"action": "merged", "pull_request": {
			"number": 147,
			"title": "docs: sync learn skill",
			"html_url": "https://github.com/SiyuQian/devpilot/pull/147"
		}}
	}, {
		"type": "PushEvent",
		"actor": {"login": "SiyuQian"},
		"created_at": "2026-05-30T06:24:14Z",
		"payload": {"ref": "refs/heads/main", "size": 1}
	}]]`
	runs := `[{
		"name": "test",
		"displayTitle": "test main",
		"status": "completed",
		"conclusion": "success",
		"event": "push",
		"headBranch": "main",
		"url": "https://github.com/SiyuQian/devpilot/actions/runs/1",
		"createdAt": "2026-06-01T07:00:00Z",
		"updatedAt": "2026-06-01T07:10:00Z"
	}]`
	releases := `[{
		"name": "v1",
		"tagName": "v1.0.0",
		"isDraft": false,
		"isPrerelease": false,
		"createdAt": "2026-06-01T05:00:00Z",
		"publishedAt": "2026-06-01T05:01:00Z"
	}]`
	r := &fakeRunner{responses: map[string]string{
		"auth\x00status": "ok",
		"api\x00repos/SiyuQian/devpilot/events\x00--paginate\x00--slurp": events,
		"search\x00prs\x00--repo\x00SiyuQian/devpilot\x00--limit\x00200\x00--json\x00repository,number,title,url,author,createdAt,updatedAt,isDraft,state\x00updated:>=2026-06-01": `[{
			"number": 147,
			"title": "docs: sync learn skill",
			"url": "https://github.com/SiyuQian/devpilot/pull/147",
			"state": "merged",
			"author": {"login": "SiyuQian"},
			"repository": {"nameWithOwner": "SiyuQian/devpilot"},
			"createdAt": "2026-06-01T06:23:23Z",
			"updatedAt": "2026-06-01T06:24:15Z"
		}]`,
		"search\x00issues\x00--repo\x00SiyuQian/devpilot\x00--limit\x00200\x00--json\x00repository,number,title,url,author,createdAt,updatedAt,state\x00updated:>=2026-06-01": `[{
			"number": 12,
			"title": "track activity",
			"url": "https://github.com/SiyuQian/devpilot/issues/12",
			"state": "open",
			"author": {"login": "SiyuQian"},
			"repository": {"nameWithOwner": "SiyuQian/devpilot"},
			"createdAt": "2026-05-31T06:23:23Z",
			"updatedAt": "2026-06-01T06:24:15Z"
		}]`,
		"run\x00list\x00--repo\x00SiyuQian/devpilot\x00--limit\x00200\x00--json\x00name,displayTitle,status,conclusion,event,headBranch,url,createdAt,updatedAt": runs,
		"release\x00list\x00--repo\x00SiyuQian/devpilot\x00--limit\x00200\x00--json\x00name,tagName,isDraft,isPrerelease,createdAt,publishedAt":                  releases,
	}}

	digest, err := repoActivity(context.Background(), r, ActivityQuery{
		Repo:     "SiyuQian/devpilot",
		Date:     "2026-06-01",
		Timezone: "UTC",
	})
	if err != nil {
		t.Fatalf("repoActivity() error = %v", err)
	}
	if len(digest.Events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(digest.Events))
	}
	if digest.Events[0].Summary != "PR #147 merged: docs: sync learn skill" {
		t.Errorf("summary = %q", digest.Events[0].Summary)
	}
	if len(digest.WorkflowRuns) != 1 {
		t.Errorf("len(workflowRuns) = %d, want 1", len(digest.WorkflowRuns))
	}
	if len(digest.PullRequests) != 1 {
		t.Errorf("len(pullRequests) = %d, want 1", len(digest.PullRequests))
	}
	if len(digest.Issues) != 1 {
		t.Errorf("len(issues) = %d, want 1", len(digest.Issues))
	}
	if len(digest.Releases) != 1 {
		t.Errorf("len(releases) = %d, want 1", len(digest.Releases))
	}
	if _, err := json.Marshal(digest); err != nil {
		t.Errorf("digest should be JSON serializable: %v", err)
	}
}
