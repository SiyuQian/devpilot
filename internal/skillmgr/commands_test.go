package skillmgr

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/project"
)

func TestParseSkillArg(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
		wantErr bool
	}{
		{input: "pm", name: "pm", version: ""},
		{input: "pm@v1.2.3", name: "pm", version: "v1.2.3"},
		{input: "google-go-style@v0.4.0", name: "google-go-style", version: "v0.4.0"},
		{input: "@v1.0.0", wantErr: true},
	}

	for _, tt := range tests {
		name, version, err := parseSkillArg(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseSkillArg(%q) expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSkillArg(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if name != tt.name {
			t.Errorf("parseSkillArg(%q) name = %q, want %q", tt.input, name, tt.name)
		}
		if version != tt.version {
			t.Errorf("parseSkillArg(%q) version = %q, want %q", tt.input, version, tt.version)
		}
	}
}

func TestSkillAddWithoutConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("requires network")
	}
	if os.Getenv("DEVPILOT_INTEGRATION") == "" {
		t.Skip("set DEVPILOT_INTEGRATION=1 to run live GitHub tests")
	}
	t.Chdir(t.TempDir())
	cmd := skillAddCmd
	cmd.ResetFlags()
	err := cmd.RunE(cmd, []string{"devpilot-pm"})
	if err != nil {
		t.Fatalf("skill add should work without .devpilot.yaml, got: %v", err)
	}

	cfg, err := project.Load(".")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Skills) == 0 {
		t.Fatal("expected skill entry in config, got none")
	}
	if cfg.Skills[0].Name != "devpilot-pm" {
		t.Errorf("skill name = %q, want %q", cfg.Skills[0].Name, "devpilot-pm")
	}
}

func TestPromptInstallLevelDefaultProject(t *testing.T) {
	dir := t.TempDir()
	input := strings.NewReader("\n") // empty = default = project
	reader := bufio.NewReader(input)

	baseDir, userLevel := promptInstallLevel(dir, reader)

	expected := filepath.Join(dir, InstallDir)
	if baseDir != expected {
		t.Errorf("baseDir = %q, want %q", baseDir, expected)
	}
	if userLevel {
		t.Error("userLevel = true, want false")
	}
}

func TestPromptInstallLevelSelectUser(t *testing.T) {
	dir := t.TempDir()
	input := strings.NewReader("2\n")
	reader := bufio.NewReader(input)

	baseDir, userLevel := promptInstallLevel(dir, reader)

	expectedUserDir, err := UserSkillDir()
	if err != nil {
		t.Fatalf("UserSkillDir: %v", err)
	}
	if baseDir != expectedUserDir {
		t.Errorf("baseDir = %q, want %q", baseDir, expectedUserDir)
	}
	if !userLevel {
		t.Error("userLevel = false, want true")
	}
}

func TestPromptInstallLevelSelectProject(t *testing.T) {
	dir := t.TempDir()
	input := strings.NewReader("1\n")
	reader := bufio.NewReader(input)

	baseDir, userLevel := promptInstallLevel(dir, reader)

	expected := filepath.Join(dir, InstallDir)
	if baseDir != expected {
		t.Errorf("baseDir = %q, want %q", baseDir, expected)
	}
	if userLevel {
		t.Error("userLevel = true, want false")
	}
}

func TestPromptInstallLevelNilReader(t *testing.T) {
	dir := t.TempDir()

	baseDir, userLevel := promptInstallLevel(dir, nil)

	expected := filepath.Join(dir, InstallDir)
	if baseDir != expected {
		t.Errorf("baseDir = %q, want %q", baseDir, expected)
	}
	if userLevel {
		t.Error("userLevel = true, want false for nil reader")
	}
}

func TestSkillAddUserLevelWritesConfig(t *testing.T) {
	// Override userConfigDirFn to use a temp directory.
	userCfgDir := t.TempDir()
	origFn := userConfigDirFn
	userConfigDirFn = func() (string, error) { return userCfgDir, nil }
	t.Cleanup(func() { userConfigDirFn = origFn })

	// Simulate the same flow as skillAddCmd when userLevel=true:
	// resolve configDir, load, upsert, save.
	configDir := userCfgDir
	cfg, err := project.Load(configDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.UpsertSkill(project.SkillEntry{
		Name:    "pm",
		Source:  DefaultSource,
		Version: "v1.0.0",
	})
	if err := project.Save(configDir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify config was written to user config dir.
	loaded, err := project.Load(userCfgDir)
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	if len(loaded.Skills) != 1 {
		t.Fatalf("len(Skills) = %d, want 1", len(loaded.Skills))
	}
	if loaded.Skills[0].Name != "pm" {
		t.Errorf("Name = %q, want %q", loaded.Skills[0].Name, "pm")
	}
	if loaded.Skills[0].Version != "v1.0.0" {
		t.Errorf("Version = %q, want %q", loaded.Skills[0].Version, "v1.0.0")
	}
}

func TestSkillListWithoutConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	cmd := skillListCmd
	err := cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("skill list should work without .devpilot.yaml, got: %v", err)
	}
}
