package initcmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/project"
)

func TestConfigureGitHubSource(t *testing.T) {
	dir := t.TempDir()
	bin := t.TempDir()
	logPath := filepath.Join(dir, "gh.log")
	gh := filepath.Join(bin, "gh")
	script := "#!/bin/sh\necho \"$@\" >> " + logPath + "\nexit 0\n"
	if err := os.WriteFile(gh, []byte(script), 0o755); err != nil {
		t.Fatalf("write gh: %v", err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := ConfigureGitHubSource(GenerateOpts{Dir: dir}); err != nil {
		t.Fatalf("ConfigureGitHubSource() error = %v", err)
	}
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if cfg.Source != "github" {
		t.Fatalf("Source = %q, want github", cfg.Source)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read gh log: %v", err)
	}
	if got := string(data); got == "" {
		t.Fatalf("gh log is empty")
	}
}

func TestRunInitNonInteractive(t *testing.T) {
	dir := t.TempDir()
	if code := runInit(dir, true, bufio.NewReader(strings.NewReader(""))); code != 0 {
		t.Fatalf("runInit code = %d", code)
	}
}
