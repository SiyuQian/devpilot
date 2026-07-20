package prreview

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls     [][]string
	responses map[string]string
	errs      map[string]error
}

func (r *fakeRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	return r.RunInput(ctx, nil, args...)
}

func (r *fakeRunner) RunInput(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	call := append([]string(nil), args...)
	if stdin != nil {
		call = append(call, "<stdin>"+string(stdin))
	}
	r.calls = append(r.calls, call)
	key := strings.Join(args, "\x00")
	if err := r.errs[key]; err != nil {
		return nil, err
	}
	if out, ok := r.responses[key]; ok {
		return []byte(out), nil
	}
	return []byte("[]"), nil
}

func key(args ...string) string { return strings.Join(args, "\x00") }

const testDiff = `diff --git a/main.go b/main.go
index 111..222 100644
--- a/main.go
+++ b/main.go
@@ -1,4 +2,5 @@
 package main
-import "fmt"
+import "fmt"
+import "github.com/fake/pkg"

 func main() {}
`

func TestParsePRURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		owner   string
		repo    string
		num     int
		wantErr bool
	}{
		{name: "valid", url: "https://github.com/siyuqian/devpilot/pull/42", owner: "siyuqian", repo: "devpilot", num: 42},
		{name: "trailing path", url: "https://github.com/a/b/pull/7/files", owner: "a", repo: "b", num: 7},
		{name: "not a pr", url: "https://github.com/a/b/issues/7", wantErr: true},
		{name: "gitlab", url: "https://gitlab.com/a/b/-/merge_requests/1", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			owner, repo, num, err := parsePRURL(tc.url)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parsePRURL(%q) = %v, want error", tc.url, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePRURL(%q): %v", tc.url, err)
			}
			if owner != tc.owner || repo != tc.repo || num != tc.num {
				t.Errorf("parsePRURL(%q) = %s/%s#%d, want %s/%s#%d", tc.url, owner, repo, num, tc.owner, tc.repo, tc.num)
			}
		})
	}
}

func TestParseDiffAnchors(t *testing.T) {
	anchors := diffAnchors(parseDiff(testDiff))
	tests := []struct {
		name string
		path string
		side string
		line int
		want bool
	}{
		{name: "added line right", path: "main.go", side: "RIGHT", line: 4, want: true},
		{name: "context line right", path: "main.go", side: "RIGHT", line: 2, want: true},
		{name: "deleted line left", path: "main.go", side: "LEFT", line: 2, want: true},
		{name: "deleted line not on right", path: "main.go", side: "RIGHT", line: 44, want: false},
		{name: "other file", path: "other.go", side: "RIGHT", line: 4, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := anchors[anchorKey(tc.path, tc.side, tc.line)]; got != tc.want {
				t.Errorf("anchor (%s,%s,%d) = %v, want %v", tc.path, tc.side, tc.line, got, tc.want)
			}
		})
	}
}

func TestExtractManifest(t *testing.T) {
	tests := []struct {
		name string
		diff string
		base baseManifests
		want map[string][]string // ecosystem -> names
	}{
		{
			name: "go mod and new import",
			diff: "diff --git a/go.mod b/go.mod\n--- a/go.mod\n+++ b/go.mod\n@@ -5,2 +5,3 @@\n require (\n+\tgithub.com/new/dep v1.2.3\n )\n" + testDiff,
			want: map[string][]string{"go": {"github.com/new/dep", "github.com/fake/pkg"}},
		},
		{
			name: "go import already in base go.mod",
			diff: testDiff,
			base: baseManifests{goMod: "module m\n\nrequire (\n\tgithub.com/fake/pkg v1.0.0\n)\n"},
			want: map[string][]string{},
		},
		{
			name: "stdlib go import skipped",
			diff: "diff --git a/a.go b/a.go\n--- a/a.go\n+++ b/a.go\n@@ -1,1 +1,2 @@\n package a\n+import \"strings\"\n",
			want: map[string][]string{},
		},
		{
			name: "npm dep and js import",
			diff: "diff --git a/package.json b/package.json\n--- a/package.json\n+++ b/package.json\n@@ -3,3 +3,4 @@\n   \"dependencies\": {\n+    \"react-llm-toolkit\": \"^3.2.1\",\n     \"left\": \"1.0.0\"\n   },\n" +
				"diff --git a/src/app.ts b/src/app.ts\n--- a/src/app.ts\n+++ b/src/app.ts\n@@ -1,1 +1,3 @@\n // x\n+import x from \"cool-lib\"\n+import fs from \"node:fs\"\n",
			want: map[string][]string{"npm": {"react-llm-toolkit", "cool-lib"}},
		},
		{
			name: "js import in base package json skipped",
			diff: "diff --git a/src/app.ts b/src/app.ts\n--- a/src/app.ts\n+++ b/src/app.ts\n@@ -1,1 +1,2 @@\n // x\n+import x from \"react\"\n",
			base: baseManifests{packageJSON: `{"dependencies":{"react":"^18.0.0"}}`},
			want: map[string][]string{},
		},
		{
			name: "python requirements",
			diff: "diff --git a/requirements.txt b/requirements.txt\n--- a/requirements.txt\n+++ b/requirements.txt\n@@ -1,1 +1,3 @@\n old==1.0\n+anthropic-codetools==0.9.0\n+# comment\n",
			want: map[string][]string{"python": {"anthropic-codetools"}},
		},
		{
			name: "cargo dependencies section only",
			diff: "diff --git a/Cargo.toml b/Cargo.toml\n--- a/Cargo.toml\n+++ b/Cargo.toml\n@@ -1,4 +1,6 @@\n [package]\n+name = \"not-a-dep\"\n [dependencies]\n+claude-tool-bridge = \"0.2\"\n+complex = { version = \"1.1\", features = [\"x\"] }\n",
			want: map[string][]string{"rust": {"claude-tool-bridge", "complex"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := extractManifest(parseDiff(tc.diff), tc.base)
			got := map[string][]string{}
			collect := func(eco string, arts []Artifact) {
				for _, a := range arts {
					name := a.Module + a.Package + a.Crate
					got[eco] = append(got[eco], name)
				}
			}
			collect("go", m.Go)
			collect("npm", m.NPM)
			collect("python", m.Python)
			collect("rust", m.Rust)
			for eco, want := range tc.want {
				if strings.Join(got[eco], ",") != strings.Join(want, ",") {
					t.Errorf("%s = %v, want %v", eco, got[eco], want)
				}
			}
			for eco, names := range got {
				if _, ok := tc.want[eco]; !ok && len(names) > 0 {
					t.Errorf("unexpected %s artifacts: %v", eco, names)
				}
			}
		})
	}
}

func prResponse(overrides map[string]any) string {
	base := map[string]any{
		"state": "open", "merged": false, "draft": false,
		"title": "T", "body": "B",
		"user": map[string]any{"login": "siyuqian"},
		"base": map[string]any{"sha": "basesha"},
		"head": map[string]any{"sha": "headsha"},
	}
	maps.Copy(base, overrides)
	b, _ := json.Marshal(base)
	return string(b)
}

func graphOK(ctx context.Context, base, head string) ([]byte, error) {
	return []byte(`{"ok":true}`), nil
}

func TestRunPreflightGates(t *testing.T) {
	prKey := key("api", "repos/o/r/pulls/1")
	tests := []struct {
		name       string
		responses  map[string]string
		force      bool
		wantReason string
		wantMode   string
	}{
		{
			name:       "merged stops",
			responses:  map[string]string{prKey: prResponse(map[string]any{"merged": true, "state": "closed"})},
			wantReason: "merged",
		},
		{
			name:       "closed stops",
			responses:  map[string]string{prKey: prResponse(map[string]any{"state": "closed"})},
			wantReason: "closed",
		},
		{
			name:       "draft stops",
			responses:  map[string]string{prKey: prResponse(map[string]any{"draft": true})},
			wantReason: "draft",
		},
		{
			name:       "bot author stops",
			responses:  map[string]string{prKey: prResponse(map[string]any{"user": map[string]any{"login": "dependabot[bot]"}})},
			wantReason: "automation_only",
		},
		{
			name: "generated only stops",
			responses: map[string]string{
				prKey: prResponse(nil),
				key("api", "--paginate", "repos/o/r/pulls/1/files"): `[{"filename":"go.sum"},{"filename":"package-lock.json"}]`,
			},
			wantReason: "generated_only",
		},
		{
			name: "empty diff stops",
			responses: map[string]string{
				prKey: prResponse(nil),
				key("api", "--paginate", "repos/o/r/pulls/1/files"): `[]`,
			},
			wantReason: "empty_diff",
		},
		{
			name: "already reviewed at head stops",
			responses: map[string]string{
				prKey: prResponse(nil),
				key("api", "--paginate", "repos/o/r/pulls/1/files"):   `[{"filename":"main.go","additions":1,"deletions":1}]`,
				key("api", "--paginate", "repos/o/r/pulls/1/reviews"): `[{"body":"<!-- devpilot:pr-review -->","commit_id":"headsha","submitted_at":"2026-01-01T00:00:00Z"}]`,
			},
			wantReason: "already_reviewed_at_head",
		},
		{
			name: "force overrides already reviewed",
			responses: map[string]string{
				prKey: prResponse(nil),
				key("api", "--paginate", "repos/o/r/pulls/1/files"):                          `[{"filename":"main.go","additions":1,"deletions":1}]`,
				key("api", "--paginate", "repos/o/r/pulls/1/reviews"):                        `[{"body":"<!-- devpilot:pr-review -->","commit_id":"headsha","submitted_at":"2026-01-01T00:00:00Z"}]`,
				key("api", "-H", "Accept: application/vnd.github.diff", "repos/o/r/pulls/1"): testDiff,
			},
			force:    true,
			wantMode: "full",
		},
		{
			name: "moved head goes incremental",
			responses: map[string]string{
				prKey: prResponse(nil),
				key("api", "--paginate", "repos/o/r/pulls/1/files"):                                           `[{"filename":"main.go","additions":1,"deletions":1}]`,
				key("api", "--paginate", "repos/o/r/pulls/1/reviews"):                                         `[{"body":"<!-- devpilot:pr-review -->","commit_id":"oldsha","submitted_at":"2026-01-01T00:00:00Z"}]`,
				key("api", "-H", "Accept: application/vnd.github.diff", "repos/o/r/compare/oldsha...headsha"): testDiff,
			},
			wantMode: "incremental",
		},
		{
			name: "clean pr proceeds full",
			responses: map[string]string{
				prKey: prResponse(nil),
				key("api", "--paginate", "repos/o/r/pulls/1/files"):                          `[{"filename":"main.go","additions":1,"deletions":1}]`,
				key("api", "-H", "Accept: application/vnd.github.diff", "repos/o/r/pulls/1"): testDiff,
			},
			wantMode: "full",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			outPath := filepath.Join(dir, "out.json")
			r := &fakeRunner{responses: tc.responses}
			res, err := RunPreflight(context.Background(), r, PreflightOptions{
				URL: "https://github.com/o/r/pull/1", OutPath: outPath,
				Force: tc.force, GraphPreflight: graphOK,
			})
			if err != nil {
				t.Fatalf("RunPreflight: %v", err)
			}
			if tc.wantReason != "" {
				if res.Gate.Decision != "stop" || res.Gate.StopReason == nil || *res.Gate.StopReason != tc.wantReason {
					t.Fatalf("gate = %+v, want stop %q", res.Gate, tc.wantReason)
				}
			} else {
				if res.Gate.Decision != "proceed" {
					t.Fatalf("gate = %+v, want proceed", res.Gate)
				}
				if res.ReviewMode != tc.wantMode {
					t.Errorf("review_mode = %q, want %q", res.ReviewMode, tc.wantMode)
				}
			}
			var onDisk PreflightResult
			b, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("out file missing: %v", err)
			}
			if err := json.Unmarshal(b, &onDisk); err != nil {
				t.Fatalf("out file not JSON: %v", err)
			}
			if onDisk.Schema != schemaPreflight {
				t.Errorf("schema = %q, want %q", onDisk.Schema, schemaPreflight)
			}
		})
	}
}

func TestRunPreflightProceedPayload(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "out.json")
	r := &fakeRunner{responses: map[string]string{
		key("api", "repos/o/r/pulls/1"):                                              prResponse(nil),
		key("api", "--paginate", "repos/o/r/pulls/1/files"):                          `[{"filename":"main.go","additions":2,"deletions":1}]`,
		key("api", "--paginate", "repos/o/r/pulls/1/comments"):                       `[{"path":"main.go","line":4,"side":"RIGHT","body":"old note","commit_id":"c1","user":{"login":"alice"}}]`,
		key("api", "-H", "Accept: application/vnd.github.diff", "repos/o/r/pulls/1"): testDiff,
	}}
	res, err := RunPreflight(context.Background(), r, PreflightOptions{
		URL: "https://github.com/o/r/pull/1", OutPath: outPath, GraphPreflight: graphOK,
	})
	if err != nil {
		t.Fatalf("RunPreflight: %v", err)
	}
	if res.DiffPath == "" {
		t.Fatal("diff_path empty")
	}
	diff, err := os.ReadFile(res.DiffPath)
	if err != nil || string(diff) != testDiff {
		t.Errorf("diff file = %q, %v; want testDiff", diff, err)
	}
	if len(res.ExistingComments) != 1 || res.ExistingComments[0].User != "alice" {
		t.Errorf("existing_comments = %+v, want one from alice", res.ExistingComments)
	}
	if string(res.Graph) != `{"ok":true}` {
		t.Errorf("graph = %s, want ok payload", res.Graph)
	}
	if len(res.DependencyManifest.Go) != 1 || res.DependencyManifest.Go[0].Module != "github.com/fake/pkg" {
		t.Errorf("go manifest = %+v, want github.com/fake/pkg", res.DependencyManifest.Go)
	}
}

func TestRunPreflightGraphFallback(t *testing.T) {
	dir := t.TempDir()
	r := &fakeRunner{responses: map[string]string{
		key("api", "repos/o/r/pulls/1"):                                              prResponse(nil),
		key("api", "--paginate", "repos/o/r/pulls/1/files"):                          `[{"filename":"main.go"}]`,
		key("api", "-H", "Accept: application/vnd.github.diff", "repos/o/r/pulls/1"): testDiff,
	}}
	res, err := RunPreflight(context.Background(), r, PreflightOptions{
		URL: "https://github.com/o/r/pull/1", OutPath: filepath.Join(dir, "out.json"),
		GraphPreflight: func(ctx context.Context, base, head string) ([]byte, error) {
			return nil, errors.New("no cache")
		},
	})
	if err != nil {
		t.Fatalf("RunPreflight: %v", err)
	}
	var g map[string]string
	if err := json.Unmarshal(res.Graph, &g); err != nil || g["mode"] != "fallback" {
		t.Errorf("graph = %s, want fallback envelope", res.Graph)
	}
}

func TestRunPreflightInfraFailure(t *testing.T) {
	r := &fakeRunner{errs: map[string]error{
		key("api", "repos/o/r/pulls/1"): errors.New("network down"),
	}}
	_, err := RunPreflight(context.Background(), r, PreflightOptions{
		URL: "https://github.com/o/r/pull/1", OutPath: filepath.Join(t.TempDir(), "out.json"),
	})
	if err == nil {
		t.Fatal("RunPreflight = nil error, want infra failure")
	}
}

func writeFindings(t *testing.T, findings []Finding) string {
	t.Helper()
	b, err := json.Marshal(findings)
	if err != nil {
		t.Fatalf("marshal findings: %v", err)
	}
	p := filepath.Join(t.TempDir(), "findings.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatalf("write findings: %v", err)
	}
	return p
}

func writeBody(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write body: %v", err)
	}
	return p
}

func postRunner() *fakeRunner {
	return &fakeRunner{responses: map[string]string{
		key("api", "repos/o/r/pulls/1"):                                              prResponse(nil),
		key("api", "-H", "Accept: application/vnd.github.diff", "repos/o/r/pulls/1"): testDiff,
		key("api", "-X", "POST", "repos/o/r/pulls/1/reviews", "--input", "-"):        `{"html_url":"https://github.com/o/r/pull/1#pullrequestreview-9"}`,
	}}
}

func TestRunPost(t *testing.T) {
	valid := []Finding{{Path: "main.go", Line: 4, Side: "RIGHT", Body: "### [Blocking] x"}}
	three := 3
	tests := []struct {
		name     string
		findings []Finding
		event    string
		body     string
		dryRun   bool
		expect   string
		wantCode int
		wantErr  string
	}{
		{name: "valid posts", findings: valid, event: "COMMENT", body: "b", wantCode: 0},
		{name: "dry run posts nothing", findings: valid, event: "REQUEST_CHANGES", body: "b", dryRun: true, wantCode: 0},
		{name: "bad event", findings: valid, event: "LGTM", body: "b", wantCode: 1, wantErr: "--event"},
		{name: "approve with findings", findings: valid, event: "APPROVE", body: "b", wantCode: 1, wantErr: "mismatch"},
		{name: "approve clean", findings: nil, event: "APPROVE", body: "b", wantCode: 0},
		{name: "anchor not in diff", findings: []Finding{{Path: "main.go", Line: 99, Body: "x"}}, event: "COMMENT", body: "b", wantCode: 1, wantErr: "finding 0"},
		{name: "left side deleted line ok", findings: []Finding{{Path: "main.go", Line: 2, Side: "LEFT", Body: "x"}}, event: "COMMENT", body: "b", wantCode: 0},
		{name: "start line after line", findings: []Finding{{Path: "main.go", Line: 2, StartLine: &three, Side: "RIGHT", Body: "x"}}, event: "COMMENT", body: "b", wantCode: 1, wantErr: "start_line"},
		{name: "body too big", findings: valid, event: "COMMENT", body: strings.Repeat("a", maxReviewBodyBytes), wantCode: 1, wantErr: "limit"},
		{name: "head moved", findings: valid, event: "COMMENT", body: "b", expect: "oldsha", wantCode: 1, wantErr: "head_moved"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := postRunner()
			out, code, err := RunPost(context.Background(), r, PostOptions{
				URL:           "https://github.com/o/r/pull/1",
				FindingsPath:  writeFindings(t, tc.findings),
				BodyPath:      writeBody(t, tc.body),
				Event:         tc.event,
				DryRun:        tc.dryRun,
				ExpectHeadSHA: tc.expect,
			})
			if code != tc.wantCode {
				t.Fatalf("code = %d (err %v), want %d", code, err, tc.wantCode)
			}
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				for _, call := range r.calls {
					if len(call) > 1 && call[1] == "-X" {
						t.Fatal("POST issued despite validation failure")
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("RunPost: %v", err)
			}
			posted := false
			for _, call := range r.calls {
				if len(call) > 1 && call[1] == "-X" {
					posted = true
				}
			}
			if tc.dryRun && posted {
				t.Fatal("dry-run issued a POST")
			}
			if !tc.dryRun && !posted {
				t.Fatal("no POST issued")
			}
			if !tc.dryRun && !strings.Contains(out, "pullrequestreview") {
				t.Errorf("out = %q, want review URL", out)
			}
		})
	}
}

func TestRunPostGitHubRejection(t *testing.T) {
	r := postRunner()
	r.errs = map[string]error{
		key("api", "-X", "POST", "repos/o/r/pulls/1/reviews", "--input", "-"): errors.New("HTTP 422"),
	}
	_, code, err := RunPost(context.Background(), r, PostOptions{
		URL:          "https://github.com/o/r/pull/1",
		FindingsPath: writeFindings(t, []Finding{{Path: "main.go", Line: 4, Side: "RIGHT", Body: "x"}}),
		BodyPath:     writeBody(t, "b"),
		Event:        "COMMENT",
	})
	if code != postExitRejected || err == nil {
		t.Fatalf("code = %d, err = %v; want %d with error", code, err, postExitRejected)
	}
}
