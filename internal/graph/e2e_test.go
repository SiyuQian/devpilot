//go:build e2e

package graph

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/envelope"
)

// TestE2EAllCommands builds the real devpilot binary and exercises every
// `graph` subcommand against a small temp repo. Each invocation must:
//   - exit 0
//   - emit JSON that validates against its v1 schema
//   - report ok=true
//
// Run with: go test -tags=e2e ./internal/graph/...
func TestE2EAllCommands(t *testing.T) {
	bindir := t.TempDir()
	binPath := filepath.Join(bindir, "devpilot")
	out, err := exec.Command("go", "build", "-o", binPath, "../../cmd/devpilot").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	home := t.TempDir()
	repo := makeE2EFixture(t)

	run := func(args ...string) []byte {
		cmd := exec.Command(binPath, args...)
		cmd.Env = append(os.Environ(), "DEVPILOT_HOME="+home)
		buf, err := cmd.Output()
		if err != nil {
			t.Fatalf("%v: %v\nstdout=%s", args, err, buf)
		}
		return buf
	}

	// Build first so the cache exists for the read commands.
	buildOut := run("graph", "build", "--repo", repo)
	if err := envelope.Validate(buildOut, "build.v1.json"); err != nil {
		t.Fatalf("build schema: %v\n%s", err, buildOut)
	}

	head, base := gitHeadAndPrev(t, repo)

	cases := []struct {
		name, schema string
		args         []string
	}{
		{"status", "status.v1.json", []string{"graph", "status", "--repo", repo}},
		{"query.callers_of", "query.v1.json", []string{"graph", "query", "callers_of", "main.go::Greet", "--repo", repo}},
		{"query.callees_of", "query.v1.json", []string{"graph", "query", "callees_of", "main.go::main", "--repo", repo}},
		{"query.tests_for", "query.v1.json", []string{"graph", "query", "tests_for", "main.go::Greet", "--repo", repo}},
		{"query.implementors_of", "query.v1.json", []string{"graph", "query", "implementors_of", "main.go::Speaker", "--repo", repo}},
		{"query.hubs", "query.v1.json", []string{"graph", "query", "hubs", "--repo", repo, "--threshold", "1"}},
		{"query.context", "query.v1.json", []string{"graph", "query", "context", "main.go::Greet", "--repo", repo}},
		{"impact", "impact.v1.json", []string{"graph", "impact", "--repo", repo, "--files", "main.go"}},
		{"hubs", "hubs.v1.json", []string{"graph", "hubs", "--repo", repo, "--threshold", "1"}},
		{"context", "context.v1.json", []string{"graph", "context", "--repo", repo, "--id", "main.go::Greet"}},
		{"detect-changes", "detect_changes.v1.json", []string{"graph", "detect-changes", "--repo", repo, "--base", base, "--head", head}},
		{"preflight", "preflight.v1.json", []string{"graph", "preflight", "--repo", repo, "--base", base, "--head", head}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw := run(c.args...)
			if err := envelope.Validate(raw, c.schema); err != nil {
				t.Fatalf("schema %s: %v\n%s", c.schema, err, raw)
			}
			var env map[string]any
			if err := json.Unmarshal(raw, &env); err != nil {
				t.Fatal(err)
			}
			if env["ok"] != true {
				t.Errorf("not ok: %v", env)
			}
		})
	}

	// Cover the --all branch of clean separately so the earlier read commands
	// keep their cache.
	t.Run("clean", func(t *testing.T) {
		raw := run("graph", "clean", "--all")
		if err := envelope.Validate(raw, "clean.v1.json"); err != nil {
			t.Fatalf("schema: %v\n%s", err, raw)
		}
	})
}

// makeE2EFixture creates a tiny git repo with two commits so detect-changes /
// preflight have a base..head range to walk.
func makeE2EFixture(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	mustGitE2E(t, repo, "init", "-q", "-b", "main")
	mustGitE2E(t, repo, "config", "user.email", "e2e@example.com")
	mustGitE2E(t, repo, "config", "user.name", "e2e")

	first := `package main

import "fmt"

type Speaker interface{ Speak() string }

func Greet(name string) string { return "hi " + name }

func main() { fmt.Println(Greet("world")) }
`
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGitE2E(t, repo, "add", "main.go")
	mustGitE2E(t, repo, "commit", "-q", "-m", "init")

	second := strings.Replace(first, `"hi " + name`, `"hello " + name`, 1)
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGitE2E(t, repo, "commit", "-q", "-am", "tweak")
	return repo
}

func mustGitE2E(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitHeadAndPrev(t *testing.T, repo string) (head, base string) {
	t.Helper()
	out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	head = strings.TrimSpace(string(out))
	out, err = exec.Command("git", "-C", repo, "rev-parse", "HEAD~1").Output()
	if err != nil {
		t.Fatal(err)
	}
	base = strings.TrimSpace(string(out))
	return
}
