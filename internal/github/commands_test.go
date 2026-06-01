package github

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestCommandConstructors(t *testing.T) {
	parent := &cobra.Command{Use: "root"}
	RegisterCommands(parent)
	if cmd, _, err := parent.Find([]string{"github"}); err != nil || cmd.Name() != "github" {
		t.Fatalf("registered github command missing: cmd=%v err=%v", cmd, err)
	}

	root := newCommand(&fakeRunner{})
	if root.Use != "github" {
		t.Fatalf("Use = %q, want github", root.Use)
	}
	for _, path := range [][]string{
		{"prs", "review-queue"},
		{"prs", "authored"},
		{"repo", "activity"},
	} {
		cmd := root
		for _, name := range path {
			var nextFound bool
			for _, child := range cmd.Commands() {
				if child.Name() == name {
					cmd = child
					nextFound = true
					break
				}
			}
			if !nextFound {
				t.Fatalf("command path %v missing %q", path, name)
			}
		}
	}
}

func TestReviewQueueCommandExecutesQuery(t *testing.T) {
	r := &fakeRunner{responses: map[string]string{
		"auth\x00status": "ok",
		"search\x00prs\x00--state\x00open\x00--limit\x001\x00--json\x00repository,number,title,url,author,createdAt,updatedAt,isDraft,state\x00--review-requested\x00alice\x00--draft=false": "[]",
	}}
	cmd := newReviewQueueCommand(r)
	cmd.SetArgs([]string{"--user", "alice", "--limit", "1", "--json"})

	out := captureStdout(t, func() {
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("ExecuteContext() error = %v", err)
		}
	})
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("stdout = %q, want []", out)
	}
}

func TestActivityCommandExecutesQuery(t *testing.T) {
	r := &fakeRunner{responses: map[string]string{
		"auth\x00status": "ok",
		"api\x00repos/SiyuQian/devpilot/events\x00--paginate\x00--slurp": "[]",
		"search\x00prs\x00--repo\x00SiyuQian/devpilot\x00--limit\x001\x00--json\x00repository,number,title,url,author,createdAt,updatedAt,isDraft,state\x00updated:>=2026-06-01": "[]",
		"search\x00issues\x00--repo\x00SiyuQian/devpilot\x00--limit\x001\x00--json\x00repository,number,title,url,author,createdAt,updatedAt,state\x00updated:>=2026-06-01":      "[]",
		"run\x00list\x00--repo\x00SiyuQian/devpilot\x00--limit\x001\x00--json\x00name,displayTitle,status,conclusion,event,headBranch,url,createdAt,updatedAt":                   "[]",
		"release\x00list\x00--repo\x00SiyuQian/devpilot\x00--limit\x001\x00--json\x00name,tagName,isDraft,isPrerelease,createdAt,publishedAt":                                    "[]",
	}}
	cmd := newActivityCommand(r)
	cmd.SetArgs([]string{"SiyuQian/devpilot", "--date", "2026-06-01", "--timezone", "UTC", "--limit", "1", "--json"})

	out := captureStdout(t, func() {
		if err := cmd.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("ExecuteContext() error = %v", err)
		}
	})
	for _, want := range []string{`"repo": "SiyuQian/devpilot"`, `"events": []`, `"workflowRuns": []`} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
}

func TestPrintPRsHumanOutput(t *testing.T) {
	prs := []PR{{
		Number:  7,
		Title:   "a title",
		URL:     "https://example.com/pr/7",
		State:   "open",
		IsDraft: true,
		Author:  User{Login: "alice"},
		Repo:    Repo{NameWithOwner: "owner/repo"},
	}}
	out := captureStdout(t, func() { printPRs(prs, false) })
	for _, want := range []string{"REPO", "owner/repo", "#7", "draft", "alice", "a title"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q:\n%s", want, out)
		}
	}

	empty := captureStdout(t, func() { printPRs(nil, false) })
	if !strings.Contains(empty, "No pull requests found.") {
		t.Errorf("empty stdout = %q", empty)
	}
}

func TestPrintActivityHumanOutput(t *testing.T) {
	now := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	digest := &ActivityDigest{
		Repo: "owner/repo",
		From: now,
		To:   now.Add(time.Hour),
		PullRequests: []PR{{
			Number:    1,
			Title:     "feat: x",
			URL:       "https://example.com/pr/1",
			State:     "OPEN",
			UpdatedAt: now,
		}},
		Issues: []Issue{{
			Number:    2,
			Title:     "bug",
			URL:       "https://example.com/issues/2",
			State:     "OPEN",
			UpdatedAt: now,
		}},
		Events: []ActivityEvent{{
			Actor:     "alice",
			CreatedAt: now,
			Summary:   "Created branch main",
		}},
		WorkflowRuns: []WorkflowRun{{
			Name:      "test",
			Status:    "completed",
			Branch:    "main",
			URL:       "https://example.com/actions/1",
			CreatedAt: now,
		}},
		Releases: []Release{{
			Name:        "release",
			TagName:     "v1",
			PublishedAt: now,
		}},
	}
	out := captureStdout(t, func() { printActivity(digest, false) })
	for _, want := range []string{"Pull Requests", "Issues", "Events", "Workflow Runs", "Releases", "feat: x", "release"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q:\n%s", want, out)
		}
	}

	empty := *digest
	empty.PullRequests = nil
	empty.Issues = nil
	empty.Events = nil
	empty.WorkflowRuns = nil
	empty.Releases = nil
	out = captureStdout(t, func() { printActivity(&empty, false) })
	if !strings.Contains(out, "No activity found.") {
		t.Errorf("empty stdout = %q", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("closing writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading stdout: %v", err)
	}
	return buf.String()
}
