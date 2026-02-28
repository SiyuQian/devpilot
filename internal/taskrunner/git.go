package taskrunner

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type GitOps struct {
	dir string
}

func NewGitOps(dir string) *GitOps {
	return &GitOps{dir: dir}
}

func (g *GitOps) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s %w", strings.Join(args, " "), string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *GitOps) CreateBranch(name string) error {
	_, err := g.run("checkout", "-B", name)
	return err
}

func (g *GitOps) CheckoutMain() error {
	if _, err := g.run("checkout", "main"); err != nil {
		_, err = g.run("checkout", "master")
		return err
	}
	return nil
}

func (g *GitOps) Pull() error {
	_, err := g.run("pull", "--ff-only")
	return err
}

func (g *GitOps) BranchName(cardID, cardName string) string {
	slug := Slugify(cardName)
	if len(slug) > 40 {
		slug = slug[:40]
		slug = strings.TrimRight(slug, "-")
	}
	if slug == "" {
		return fmt.Sprintf("task/%s", cardID)
	}
	return fmt.Sprintf("task/%s-%s", cardID, slug)
}

func (g *GitOps) Push(branch string) error {
	_, err := g.run("push", "-u", "origin", branch)
	return err
}

func (g *GitOps) CreatePR(title, body string) (string, error) {
	cmd := exec.Command("gh", "pr", "create", "--title", title, "--body", body)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr create: %s %w", string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *GitOps) MergePR() error {
	cmd := exec.Command("gh", "pr", "merge", "--squash", "--auto")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh pr merge: %s %w", string(out), err)
	}
	return nil
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
