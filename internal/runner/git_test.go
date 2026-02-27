package runner

import (
	"os/exec"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %s %v", args, out, err)
		}
	}
	return dir
}

func TestCreateBranch(t *testing.T) {
	dir := setupGitRepo(t)
	git := NewGitOps(dir)

	err := git.CreateBranch("task/abc123-fix-bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	out, _ := cmd.Output()
	branch := string(out)
	if branch != "task/abc123-fix-bug\n" {
		t.Errorf("expected task/abc123-fix-bug, got %q", branch)
	}
}

func TestCheckoutMain(t *testing.T) {
	dir := setupGitRepo(t)
	git := NewGitOps(dir)

	git.CreateBranch("task/test")

	err := git.CheckoutMain()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	out, _ := cmd.Output()
	branch := string(out)
	if branch != "main\n" && branch != "master\n" {
		t.Errorf("expected main or master, got %q", branch)
	}
}

func TestBranchName(t *testing.T) {
	git := NewGitOps("/tmp")
	name := git.BranchName("abc123", "Fix auth bug")
	if name != "task/abc123-fix-auth-bug" {
		t.Errorf("unexpected branch name: %s", name)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Fix auth bug", "fix-auth-bug"},
		{"Add Login Endpoint!!", "add-login-endpoint"},
		{"hello   world", "hello-world"},
	}
	for _, tt := range tests {
		got := Slugify(tt.input)
		if got != tt.expected {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
