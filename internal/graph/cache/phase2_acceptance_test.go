package cache

import (
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestPhase2Acceptance asserts the four acceptance criteria from the plan in one place.
func TestPhase2Acceptance(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	t.Run("repokey_deterministic_12_chars", func(t *testing.T) {
		k := RepoKey("/abs/path/x")
		if len(k) != 12 {
			t.Errorf("len=%d", len(k))
		}
		if k != RepoKey("/abs/path/x") {
			t.Error("not deterministic")
		}
	})

	t.Run("two_full_builds_identical", func(t *testing.T) {
		repo := setupRepo(t, map[string]string{
			"main.go": "package main\nfunc A(){}\nfunc main(){A()}\n",
			"util.ts": "export function x(){}\n",
			"lib.rs":  "pub fn y(){}\n",
		})
		home := t.TempDir()
		b, _ := NewBuilder(home, repo)
		if _, err := b.FullBuild(); err != nil {
			t.Fatal(err)
		}
		d1 := dumpDB(t, GraphDB(home, RepoKey(repo)))
		if _, err := b.FullBuild(); err != nil {
			t.Fatal(err)
		}
		d2 := dumpDB(t, GraphDB(home, RepoKey(repo)))
		if d1 != d2 {
			t.Error("two full builds differ")
		}
	})

	t.Run("incremental_matches_full_5_files", func(t *testing.T) {
		repo := setupGitRepo(t, map[string]string{
			"a.go": "package x\nfunc A(){}\n",
			"b.go": "package x\nfunc B(){ A() }\n",
		})
		homeA := t.TempDir()
		bA, _ := NewBuilder(homeA, repo)
		if _, err := bA.FullBuild(); err != nil {
			t.Fatal(err)
		}
		mutateRepo(t, repo, map[string]string{
			"a.go": "package x\nfunc A(){}\nfunc A2(){}\n",
			"b.go": "package x\nfunc B(){ A(); A2() }\n",
			"c.go": "package x\nfunc C(){}\n",
			"d.go": "package x\nfunc D(){}\n",
			"e.go": "package x\nfunc E(){}\n",
		})
		if _, err := bA.Build(); err != nil {
			t.Fatal(err)
		}

		homeB := t.TempDir()
		bB, _ := NewBuilder(homeB, repo)
		if _, err := bB.FullBuild(); err != nil {
			t.Fatal(err)
		}
		if dumpDB(t, GraphDB(homeA, RepoKey(repo))) != dumpDB(t, GraphDB(homeB, RepoKey(repo))) {
			t.Error("incremental != full")
		}
	})

	t.Run("flock_serializes_concurrent_builders", func(t *testing.T) {
		repo := setupRepo(t, map[string]string{"main.go": "package main\nfunc main(){}\n"})
		home := t.TempDir()
		key := RepoKey(repo)
		_ = EnsureDirs(home, key)
		lockPath := LockFile(home, key)

		var (
			mu    sync.Mutex
			maxIn int
			in    int
			wg    sync.WaitGroup
			fail  error
		)
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				rel, err := AcquireBuildLock(lockPath, 5*time.Second)
				if err != nil {
					fail = err
					return
				}
				defer func() { _ = rel() }()
				mu.Lock()
				in++
				if in > maxIn {
					maxIn = in
				}
				mu.Unlock()
				time.Sleep(30 * time.Millisecond)
				mu.Lock()
				in--
				mu.Unlock()
			}()
		}
		wg.Wait()
		if fail != nil {
			t.Fatal(fail)
		}
		if maxIn != 1 {
			t.Errorf("max concurrent=%d, want 1", maxIn)
		}
	})
}

func setupRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	for p, c := range files {
		mustWrite(t, filepath.Join(repo, p), c)
	}
	return repo
}

func setupGitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := setupRepo(t, files)
	mustGit(t, repo, "init", "-q")
	mustGit(t, repo, "config", "user.email", "t@t")
	mustGit(t, repo, "config", "user.name", "t")
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "init")
	return repo
}

func mutateRepo(t *testing.T, repo string, files map[string]string) {
	t.Helper()
	for p, c := range files {
		mustWrite(t, filepath.Join(repo, p), c)
	}
	mustGit(t, repo, "add", ".")
	mustGit(t, repo, "commit", "-qm", "mutate")
}
