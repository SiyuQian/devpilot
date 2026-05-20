package cache

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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
