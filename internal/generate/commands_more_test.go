package generate

import (
	"os"
	"testing"

	"github.com/siyuqian/devpilot/internal/project"
	"github.com/spf13/cobra"
)

func TestGenerateRegisterCommandsAndResolveModel(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	RegisterCommands(root)
	cmd, _, err := root.Find([]string{"commit"})
	if err != nil {
		t.Fatalf("Find(commit): %v", err)
	}
	if cmd.Name() != "commit" {
		t.Fatalf("cmd = %q, want commit", cmd.Name())
	}
	if err := cmd.Flags().Set("model", "custom-model"); err != nil {
		t.Fatalf("set model: %v", err)
	}
	if got := resolveModel(cmd, "commit"); got != "custom-model" {
		t.Fatalf("resolveModel() = %q, want custom-model", got)
	}
}

func TestResolveModelFromProjectConfig(t *testing.T) {
	dir := t.TempDir()
	if err := project.Save(dir, &project.Config{Models: map[string]string{"commit": "sonnet"}}); err != nil {
		t.Fatalf("Save config: %v", err)
	}
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

	cmd := &cobra.Command{}
	cmd.Flags().String("model", "", "")
	if got := resolveModel(cmd, "commit"); got != "sonnet" {
		t.Fatalf("resolveModel() = %q, want sonnet", got)
	}
}
