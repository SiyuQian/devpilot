package cache

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestIncrementalMatchesFullRebuild(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	mustGit(t, repo, "init", "-q")
	mustGit(t, repo, "config", "user.email", "t@t")
	mustGit(t, repo, "config", "user.name", "t")
	mustWrite(t, filepath.Join(repo, "a.go"), "package x\nfunc A(){}\n")
	mustWrite(t, filepath.Join(repo, "b.go"), "package x\nfunc B(){ A() }\n")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "first")

	homeA := t.TempDir()
	bA, err := NewBuilder(homeA, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bA.FullBuild(); err != nil {
		t.Fatal(err)
	}

	// Mutate b.go and add c.go, then commit.
	mustWrite(t, filepath.Join(repo, "b.go"), "package x\nfunc B(){ A(); A() }\n")
	mustWrite(t, filepath.Join(repo, "c.go"), "package x\nfunc C(){}\n")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "second")

	res, err := bA.Build()
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != "incremental" {
		t.Errorf("mode=%q want incremental", res.Mode)
	}

	homeB := t.TempDir()
	bB, err := NewBuilder(homeB, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bB.FullBuild(); err != nil {
		t.Fatal(err)
	}

	got := dumpDB(t, GraphDB(homeA, RepoKey(repo)))
	want := dumpDB(t, GraphDB(homeB, RepoKey(repo)))
	if got != want {
		t.Errorf("incremental result differs from full rebuild\n--- incremental ---\n%s\n--- full ---\n%s", got, want)
	}
}

func mustGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE=2026-05-20T00:00:00",
		"GIT_COMMITTER_DATE=2026-05-20T00:00:00",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestIncrementalNativeGoMatchesFullRebuild(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	t.Setenv("DEVPILOT_GRAPH_GO_BACKEND", "native")

	repo := t.TempDir()
	copyGoNativeFixture(t, repo)
	mustGit(t, repo, "init", "-q")
	mustGit(t, repo, "config", "user.email", "t@t")
	mustGit(t, repo, "config", "user.name", "t")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "initial")

	homeA := t.TempDir()
	bA, err := NewBuilder(homeA, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bA.FullBuild(); err != nil {
		t.Fatal(err)
	}

	// Mutate pkg/a/a.go: rename Greet -> Hello (and its call site in Run).
	mustWrite(t, filepath.Join(repo, "pkg/a/a.go"), `package a

func Hello(name string) string {
	return "hi " + name
}

func Run() string {
	return Hello("world")
}
`)
	// pkg/b still references Greet — fix it so the module type-checks.
	mustWrite(t, filepath.Join(repo, "pkg/b/b.go"), `package b

import "example.com/native/pkg/a"

func B() string {
	return a.Hello("y")
}
`)
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "rename")

	res, err := bA.Build()
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != "incremental" {
		t.Errorf("mode=%q want incremental", res.Mode)
	}

	homeB := t.TempDir()
	bB, err := NewBuilder(homeB, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bB.FullBuild(); err != nil {
		t.Fatal(err)
	}

	got := dumpDB(t, GraphDB(homeA, RepoKey(repo)))
	want := dumpDB(t, GraphDB(homeB, RepoKey(repo)))
	if got != want {
		t.Errorf("native incremental differs from full rebuild\n--- incremental ---\n%s\n--- full ---\n%s", got, want)
	}
}

func TestIncrementalNativeGoNonGoChangeSkipsLoadModule(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	t.Setenv("DEVPILOT_GRAPH_GO_BACKEND", "native")

	repo := t.TempDir()
	copyGoNativeFixture(t, repo)
	mustGit(t, repo, "init", "-q")
	mustGit(t, repo, "config", "user.email", "t@t")
	mustGit(t, repo, "config", "user.name", "t")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "initial")

	home := t.TempDir()
	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.FullBuild(); err != nil {
		t.Fatal(err)
	}

	// Add a non-Go file and commit.
	mustWrite(t, filepath.Join(repo, "README.md"), "# hi\n")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "readme")

	start := time.Now()
	res, err := b.Build()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != "incremental" {
		t.Errorf("mode=%q want incremental", res.Mode)
	}
	// A real packages.Load takes seconds; this path must skip it. 1s is a
	// loose ceiling that excludes any whole-module re-typecheck.
	if elapsed > time.Second {
		t.Errorf("non-Go incremental took %v; expected < 1s (LoadModule should not run)", elapsed)
	}
}

// TestIncrementalNativeGoNonModuleFallback covers the case the native flag is
// set but the repo has no go.mod / go.work. BuildIncremental must route Go
// files through tree-sitter instead of the native parser's no-op Parse.
// Regression test for the data-loss bug where edits to *.go files silently
// dropped from the graph.
func TestIncrementalNativeGoNonModuleFallback(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	t.Setenv("DEVPILOT_GRAPH_GO_BACKEND", "native")

	repo := t.TempDir()
	// Bare main.go, no go.mod — non-module repo.
	mustWrite(t, filepath.Join(repo, "main.go"), `package main
func Greet(n string) string { return "hi " + n }
func main() { Greet("x") }
`)
	mustGit(t, repo, "init", "-q")
	mustGit(t, repo, "config", "user.email", "t@t")
	mustGit(t, repo, "config", "user.name", "t")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "initial")

	home := t.TempDir()
	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.FullBuild(); err != nil {
		t.Fatal(err)
	}
	beforeNodes := countNodes(t, GraphDB(home, RepoKey(repo)))
	if beforeNodes == 0 {
		t.Fatal("FullBuild produced no nodes — fallback to tree-sitter expected")
	}

	// Mutate main.go: add a function so the graph must change.
	mustWrite(t, filepath.Join(repo, "main.go"), `package main
func Greet(n string) string { return "hi " + n }
func Farewell(n string) string { return "bye " + n }
func main() { Greet("x"); Farewell("y") }
`)
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "add Farewell")

	res, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != "incremental" {
		t.Errorf("mode=%q want incremental", res.Mode)
	}

	afterNodes := countNodes(t, GraphDB(home, RepoKey(repo)))
	if afterNodes <= beforeNodes {
		t.Errorf("expected node count to grow after adding Farewell; before=%d after=%d", beforeNodes, afterNodes)
	}
	// Specifically, Farewell must have a node — the bug would silently lose it.
	if !nodeExists(t, GraphDB(home, RepoKey(repo)), "main.go::Farewell") {
		t.Errorf("main.go::Farewell missing after incremental rebuild (non-module fallback regression)")
	}
}

func countNodes(t *testing.T, dbPath string) int {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM nodes`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func nodeExists(t *testing.T, dbPath, id string) bool {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM nodes WHERE id = ?`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n > 0
}
