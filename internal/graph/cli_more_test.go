package graph

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/siyuqian/devpilot/internal/graph/cache"
)

func TestGraphCommandConstructors(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	RegisterCommands(root)
	graphCmd, _, err := root.Find([]string{"graph"})
	if err != nil {
		t.Fatalf("Find(graph) error = %v", err)
	}
	if graphCmd == nil || graphCmd.Name() != "graph" {
		t.Fatalf("graph command missing")
	}
	for _, name := range []string{"build", "status", "clean", "query", "impact", "hubs", "context", "detect-changes", "preflight"} {
		cmd, _, err := graphCmd.Find([]string{name})
		if err != nil {
			t.Fatalf("Find(%s) error = %v", name, err)
		}
		if cmd == nil || cmd.Name() != name {
			t.Fatalf("command %q missing", name)
		}
	}
}

func TestGraphRunDiffCommandsSuccess(t *testing.T) {
	repo := seedQueryStore(t)
	graphTestGit(t, repo, "init")
	graphTestGit(t, repo, "config", "user.email", "devpilot@example.com")
	graphTestGit(t, repo, "config", "user.name", "DevPilot")
	graphTestGit(t, repo, "add", "src.go")
	graphTestGit(t, repo, "commit", "-m", "base")
	base := strings.TrimSpace(string(graphTestGit(t, repo, "rev-parse", "HEAD")))
	if err := os.WriteFile(filepath.Join(repo, "src.go"), []byte("package p\nfunc Target() {\n\tprintln(\"changed\")\n}\nfunc Caller() {\n\tTarget()\n}\n"), 0o644); err != nil {
		t.Fatalf("write changed source: %v", err)
	}
	graphTestGit(t, repo, "add", "src.go")
	graphTestGit(t, repo, "commit", "-m", "head")
	head := strings.TrimSpace(string(graphTestGit(t, repo, "rev-parse", "HEAD")))

	cases := []struct {
		name string
		run  func() int
		want string
	}{
		{name: "detect changes", run: func() int { return runDetectChanges(repo, base, head) }, want: "graph.detect-changes"},
		{name: "preflight", run: func() int { return runPreflight(repo, base, head) }, want: "graph.preflight"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rc int
			out := captureStdout(t, func() {
				rc = tc.run()
			})
			if rc != 0 {
				t.Fatalf("rc=%d output=%s", rc, out)
			}
			var env map[string]any
			if err := json.Unmarshal(out, &env); err != nil {
				t.Fatalf("json decode: %v\n%s", err, out)
			}
			if env["command"] != tc.want || env["ok"] != true {
				t.Fatalf("unexpected envelope: %v", env)
			}
		})
	}
}

func graphTestGit(t *testing.T, dir string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return out
}

func TestGraphRunCommandsSuccess(t *testing.T) {
	repo := seedQueryStore(t)
	home := cache.Home()
	key := cache.RepoKey(repo)
	if err := cache.WriteMeta(cache.MetaFile(home, key), cache.Meta{
		SchemaVersion: cache.CurrentSchemaVersion,
		HeadSHA:       "abc123",
		Languages:     []string{"go"},
		BuiltAtUnix:   1780300000,
	}); err != nil {
		t.Fatalf("WriteMeta: %v", err)
	}

	cases := []struct {
		name string
		run  func() int
		want string
	}{
		{name: "status", run: func() int { return runStatus(repo) }, want: "graph.status"},
		{name: "context", run: func() int { return runContext(repo, "target", 1) }, want: "graph.context"},
		{name: "hubs", run: func() int { return runHubs(repo, 2) }, want: "graph.hubs"},
		{name: "impact", run: func() int { return runImpact(repo, "src.go", 1) }, want: "graph.impact"},
		{name: "clean repo", run: func() int { return runClean(repo, false) }, want: "graph.clean"},
		{name: "clean all", run: func() int { return runClean("", true) }, want: "graph.clean"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rc int
			out := captureStdout(t, func() {
				rc = tc.run()
			})
			if rc != 0 {
				t.Fatalf("rc=%d output=%s", rc, out)
			}
			var env map[string]any
			if err := json.Unmarshal(out, &env); err != nil {
				t.Fatalf("json decode: %v\n%s", err, out)
			}
			if env["command"] != tc.want || env["ok"] != true {
				t.Fatalf("unexpected envelope: %v", env)
			}
		})
	}
}

func TestGraphRunCommandsValidateArgs(t *testing.T) {
	badRepo := "/path/that/does/not/exist"
	cases := []struct {
		name string
		code int
	}{
		{name: "clean requires args", code: runClean("", false)},
		{name: "clean invalid repo", code: runClean(badRepo, false)},
		{name: "status invalid repo", code: runStatus(badRepo)},
		{name: "context invalid repo", code: runContext(badRepo, "", 0)},
		{name: "hubs invalid repo", code: runHubs(badRepo, 0)},
		{name: "impact invalid repo", code: runImpact(badRepo, "", 0)},
		{name: "preflight invalid repo", code: runPreflight(badRepo, "", "")},
		{name: "detect changes invalid repo", code: runDetectChanges(badRepo, "", "")},
		{name: "query invalid repo", code: runQuery(queryOpts{repo: badRepo}, "unknown", "")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.code == 0 {
				t.Fatalf("exit code = 0, want failure")
			}
		})
	}
}
