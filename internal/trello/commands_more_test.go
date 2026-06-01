package trello

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunPushGitHubAndErrors(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	bin := t.TempDir()
	ghLog := filepath.Join(dir, "gh.log")
	gh := filepath.Join(bin, "gh")
	if err := os.WriteFile(gh, []byte("#!/bin/sh\necho \"$@\" >> "+ghLog+"\necho https://github.com/o/r/issues/1\n"), 0o755); err != nil {
		t.Fatalf("write gh: %v", err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	plan := filepath.Join(dir, "plan.md")
	if err := os.WriteFile(plan, []byte("# My task\n\nBody"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	cmd := pushTestCmd(t, "github", "")
	if code := runPush(cmd, plan); code != 0 {
		t.Fatalf("runPush github code = %d", code)
	}
	log, err := os.ReadFile(ghLog)
	if err != nil {
		t.Fatalf("read gh log: %v", err)
	}
	if !strings.Contains(string(log), "issue create") || !strings.Contains(string(log), "My task") {
		t.Fatalf("gh log = %s", log)
	}

	if code := runPush(cmd, filepath.Join(dir, "missing.md")); code == 0 {
		t.Fatalf("missing file code = 0")
	}
	noTitle := filepath.Join(dir, "no-title.md")
	if err := os.WriteFile(noTitle, []byte("body"), 0o644); err != nil {
		t.Fatalf("write no title: %v", err)
	}
	if code := runPush(cmd, noTitle); code == 0 {
		t.Fatalf("no title code = 0")
	}
	if code := runPush(pushTestCmd(t, "bad", ""), plan); code == 0 {
		t.Fatalf("unknown source code = 0")
	}
	if code := runPush(pushTestCmd(t, "trello", ""), plan); code == 0 {
		t.Fatalf("missing board code = 0")
	}
}

func TestRegisterCommands(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	RegisterCommands(root)
	cmd, _, err := root.Find([]string{"push"})
	if err != nil {
		t.Fatalf("Find(push): %v", err)
	}
	if cmd.Name() != "push" {
		t.Fatalf("cmd = %q, want push", cmd.Name())
	}
}

func pushTestCmd(t *testing.T, source, board string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("source", source, "")
	cmd.Flags().String("board", board, "")
	cmd.Flags().String("list", "Ready", "")
	return cmd
}
