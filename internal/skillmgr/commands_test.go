package skillmgr

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/project"
	"github.com/spf13/cobra"
)

func TestParseSkillArg(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		ref     string
		wantErr bool
	}{
		{input: "pm", name: "pm", ref: ""},
		{input: "pm@v1.2.3", name: "pm", ref: "v1.2.3"},
		{input: "google-go-style@v0.4.0", name: "google-go-style", ref: "v0.4.0"},
		{input: "@v1.0.0", wantErr: true},
	}

	for _, tt := range tests {
		name, ref, err := parseSkillArg(tt.input)
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
		if ref != tt.ref {
			t.Errorf("parseSkillArg(%q) ref = %q, want %q", tt.input, ref, tt.ref)
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
		Name:   "pm",
		Source: DefaultSource,
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
}

func TestSkillListBothLevels(t *testing.T) {
	userCfgDir := stubUserConfigDir(t)

	projDir := t.TempDir()
	t.Chdir(projDir)
	projCfg := &project.Config{
		Skills: []project.SkillEntry{
			{Name: "pm", Source: "github.com/siyuqian/devpilot"},
		},
	}
	if err := project.Save(projDir, projCfg); err != nil {
		t.Fatalf("Save project config: %v", err)
	}

	userCfg := &project.Config{
		Skills: []project.SkillEntry{
			{Name: "prompt-review", Source: "github.com/siyuqian/devpilot"},
		},
	}
	if err := project.Save(userCfgDir, userCfg); err != nil {
		t.Fatalf("Save user config: %v", err)
	}

	output, err := runSkillListCmd(t, true)
	if err != nil {
		t.Fatalf("skill list error: %v", err)
	}

	if !strings.Contains(output, "pm") {
		t.Errorf("output missing project skill 'pm': %s", output)
	}
	if !strings.Contains(output, "prompt-review") {
		t.Errorf("output missing user skill 'prompt-review': %s", output)
	}
	if !strings.Contains(output, "project") {
		t.Errorf("output missing 'project' level indicator: %s", output)
	}
	if !strings.Contains(output, "user") {
		t.Errorf("output missing 'user' level indicator: %s", output)
	}
}

func TestSkillListOnlyUserLevel(t *testing.T) {
	userCfgDir := stubUserConfigDir(t)
	t.Chdir(t.TempDir()) // no .devpilot.yaml here

	userCfg := &project.Config{
		Skills: []project.SkillEntry{
			{Name: "prompt-review", Source: "github.com/siyuqian/devpilot"},
		},
	}
	if err := project.Save(userCfgDir, userCfg); err != nil {
		t.Fatalf("Save user config: %v", err)
	}

	output, err := runSkillListCmd(t, true)
	if err != nil {
		t.Fatalf("skill list should not error without project config: %v", err)
	}

	if !strings.Contains(output, "prompt-review") {
		t.Errorf("output missing user skill: %s", output)
	}
}

func TestSkillListNoSkillsAnywhere(t *testing.T) {
	stubUserConfigDir(t)
	t.Chdir(t.TempDir())

	output, err := runSkillListCmd(t, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "No skills installed") {
		t.Errorf("expected 'No skills installed' message, got: %s", output)
	}
}

// stubUserConfigDir overrides userConfigDirFn to use a temp directory.
func stubUserConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig := userConfigDirFn
	userConfigDirFn = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userConfigDirFn = orig })
	return dir
}

// stubCatalogFns overrides fetchCatalogFn for tests.
func stubCatalogFns(t *testing.T, catalog []CatalogEntry, catalogErr error) {
	t.Helper()
	origCat := fetchCatalogFn
	fetchCatalogFn = func(_ context.Context, _, _, _ string) ([]CatalogEntry, error) {
		return catalog, catalogErr
	}
	t.Cleanup(func() {
		fetchCatalogFn = origCat
	})
}

func runSkillListCmd(t *testing.T, installed bool) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := skillListCmd
	cmd.ResetFlags()
	cmd.Flags().BoolP("installed", "i", false, "Show only installed skills")
	if installed {
		_ = cmd.Flags().Set("installed", "true")
	}
	err := cmd.RunE(cmd, []string{})

	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out), err
}

func TestSkillListCatalogView(t *testing.T) {
	stubUserConfigDir(t)

	projDir := t.TempDir()
	t.Chdir(projDir)
	projCfg := &project.Config{
		Skills: []project.SkillEntry{
			{Name: "pm", Source: DefaultSource},
		},
	}
	if err := project.Save(projDir, projCfg); err != nil {
		t.Fatalf("Save project config: %v", err)
	}

	stubCatalogFns(t, []CatalogEntry{
		{Name: "pm", Description: "Product manager skill"},
		{Name: "trello", Description: "Trello integration"},
		{Name: "learn", Description: "Summarize articles"},
	}, nil)

	output, err := runSkillListCmd(t, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Installed skill should show install date and level.
	if !strings.Contains(output, "pm") || !strings.Contains(output, "project") {
		t.Errorf("installed skill 'pm' not shown correctly: %s", output)
	}
	// Uninstalled skills should show dashes.
	if !strings.Contains(output, "trello") {
		t.Errorf("catalog skill 'trello' missing: %s", output)
	}
	if !strings.Contains(output, "learn") {
		t.Errorf("catalog skill 'learn' missing: %s", output)
	}
	// Should have DESCRIPTION and INSTALLED columns.
	if !strings.Contains(output, "DESCRIPTION") {
		t.Errorf("missing DESCRIPTION header: %s", output)
	}
	if !strings.Contains(output, "INSTALLED") {
		t.Errorf("missing INSTALLED header: %s", output)
	}
}

func TestSkillListInstalledFlag(t *testing.T) {
	stubUserConfigDir(t)

	projDir := t.TempDir()
	t.Chdir(projDir)
	projCfg := &project.Config{
		Skills: []project.SkillEntry{
			{Name: "pm", Source: DefaultSource},
		},
	}
	if err := project.Save(projDir, projCfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Should NOT call catalog — stub with error to prove it.
	stubCatalogFns(t, nil, fmt.Errorf("should not be called"))

	output, err := runSkillListCmd(t, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "pm") {
		t.Errorf("installed skill missing: %s", output)
	}
	// Should NOT contain DESCRIPTION column in installed-only view.
	if strings.Contains(output, "DESCRIPTION") {
		t.Errorf("--installed view should not have DESCRIPTION column: %s", output)
	}
}

func TestSkillListTruncateDescription(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"short", "short"},
		{"exactly forty characters long!!!!!!!!!!!", "exactly forty characters long!!!!!!!!!!!"},                                               // 40 chars
		{"this description is definitely longer than forty characters and should be truncated", "this description is definitely longer th..."}, // 41+ chars
		{"日本語のテスト", "日本語のテスト"},                                                                                                                 // short multi-byte
	}
	for _, tt := range tests {
		got := truncateDescription(tt.input)
		if got != tt.want {
			t.Errorf("truncateDescription(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestValidateSkillAddArgs(t *testing.T) {
	tests := []struct {
		name    string
		all     bool
		args    []string
		wantErr bool
	}{
		{name: "single name no flag", all: false, args: []string{"pm"}, wantErr: false},
		{name: "all flag no args", all: true, args: []string{}, wantErr: false},
		{name: "no args no flag", all: false, args: []string{}, wantErr: true},
		{name: "all flag with name", all: true, args: []string{"pm"}, wantErr: true},
		{name: "two positional args", all: false, args: []string{"pm", "trello"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool("all", false, "")
			if tt.all {
				_ = cmd.Flags().Set("all", "true")
			}
			err := validateSkillAddArgs(cmd, tt.args)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveInstallLevel(t *testing.T) {
	projectDir := t.TempDir()
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	expectedUser := filepath.Join(userHome, ".claude", "skills")
	expectedProject := filepath.Join(projectDir, InstallDir)

	t.Run("flag project overrides TTY prompt", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("2\n")) // would select user if prompted
		base, userLevel, err := resolveInstallLevel("project", projectDir, reader)
		if err != nil {
			t.Fatal(err)
		}
		if base != expectedProject {
			t.Errorf("base = %q, want %q", base, expectedProject)
		}
		if userLevel {
			t.Error("userLevel = true, want false")
		}
	})

	t.Run("flag user overrides default", func(t *testing.T) {
		base, userLevel, err := resolveInstallLevel("user", projectDir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if base != expectedUser {
			t.Errorf("base = %q, want %q", base, expectedUser)
		}
		if !userLevel {
			t.Error("userLevel = false, want true")
		}
	})

	t.Run("empty flag with TTY prompts", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("2\n"))
		base, userLevel, err := resolveInstallLevel("", projectDir, reader)
		if err != nil {
			t.Fatal(err)
		}
		if !userLevel {
			t.Error("userLevel = false, want true (from prompt)")
		}
		if base != expectedUser {
			t.Errorf("base = %q, want %q", base, expectedUser)
		}
	})

	t.Run("empty flag no TTY defaults to project", func(t *testing.T) {
		base, userLevel, err := resolveInstallLevel("", projectDir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if userLevel {
			t.Error("userLevel = true, want false")
		}
		if base != expectedProject {
			t.Errorf("base = %q, want %q", base, expectedProject)
		}
	})

	t.Run("invalid flag value errors", func(t *testing.T) {
		_, _, err := resolveInstallLevel("system", projectDir, nil)
		if err == nil {
			t.Error("expected error for invalid --level value")
		}
	})
}

// stubFetchSkillFn overrides fetchSkillFn for bulk install tests.
// The provided map keys skill names to the files returned for that skill;
// a skill name in failing returns an error instead.
func stubFetchSkillFn(t *testing.T, files map[string][]SkillFile, failing map[string]error) {
	t.Helper()
	orig := fetchSkillFn
	fetchSkillFn = func(_, _, name, _ string) ([]SkillFile, error) {
		if err, ok := failing[name]; ok {
			return nil, err
		}
		if f, ok := files[name]; ok {
			return f, nil
		}
		return nil, fmt.Errorf("unknown skill %q", name)
	}
	t.Cleanup(func() { fetchSkillFn = orig })
}

// setupBulkStubs installs catalog + fetch-skill stubs in one call. Each name
// in names becomes a catalog entry AND gets a minimal SKILL.md file unless
// listed in failing, in which case fetching it returns the given error.
func setupBulkStubs(t *testing.T, names []string, failing map[string]error) {
	t.Helper()
	catalog := make([]CatalogEntry, 0, len(names))
	files := make(map[string][]SkillFile, len(names))
	for _, name := range names {
		catalog = append(catalog, CatalogEntry{Name: name})
		files[name] = []SkillFile{{Path: "SKILL.md", Content: []byte(name)}}
	}
	stubCatalogFns(t, catalog, nil)
	stubFetchSkillFn(t, files, failing)
}

func TestRunBulkInstallProjectLevel(t *testing.T) {
	dir := t.TempDir()
	setupBulkStubs(t, []string{"pm", "trello"}, nil)

	err := runBulkInstall(dir, "project", nil)
	if err != nil {
		t.Fatalf("runBulkInstall: %v", err)
	}

	// Both skills should exist on disk.
	for _, name := range []string{"pm", "trello"} {
		p := filepath.Join(dir, InstallDir, name, "SKILL.md")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s, got err: %v", p, err)
		}
	}

	// Config should record both.
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Skills) != 2 {
		t.Errorf("len(Skills) = %d, want 2", len(cfg.Skills))
	}
}

func TestRunBulkInstallPartialFailure(t *testing.T) {
	dir := t.TempDir()
	setupBulkStubs(t,
		[]string{"pm", "broken", "trello"},
		map[string]error{"broken": fmt.Errorf("404 not found")},
	)

	err := runBulkInstall(dir, "project", nil)
	if err == nil {
		t.Fatal("expected non-nil error on partial failure")
	}
	if !strings.Contains(err.Error(), "1 skill") {
		t.Errorf("error should mention failure count, got: %v", err)
	}

	// Successful skills should still be installed and recorded.
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Skills) != 2 {
		t.Errorf("len(Skills) = %d, want 2 (pm + trello)", len(cfg.Skills))
	}
	if _, err := os.Stat(filepath.Join(dir, InstallDir, "broken", "SKILL.md")); err == nil {
		t.Error("broken skill should not have been installed")
	}
}

func TestRunBulkInstallUserLevel(t *testing.T) {
	userCfgDir := stubUserConfigDir(t)
	t.Setenv("HOME", userCfgDir) // so UserSkillDir returns <userCfgDir>/.claude/skills

	dir := t.TempDir()
	setupBulkStubs(t, []string{"pm"}, nil)

	if err := runBulkInstall(dir, "user", nil); err != nil {
		t.Fatalf("runBulkInstall: %v", err)
	}

	// Skill should be installed under userCfgDir/.claude/skills, NOT the project dir.
	userSkill := filepath.Join(userCfgDir, ".claude", "skills", "pm", "SKILL.md")
	if _, err := os.Stat(userSkill); err != nil {
		t.Errorf("expected %s, got err: %v", userSkill, err)
	}
	projectSkill := filepath.Join(dir, InstallDir, "pm")
	if _, err := os.Stat(projectSkill); err == nil {
		t.Error("skill should not be installed at project level when --level user is set")
	}

	// Config should be written to user config dir, not project.
	userCfg, err := project.Load(userCfgDir)
	if err != nil {
		t.Fatalf("Load user cfg: %v", err)
	}
	if len(userCfg.Skills) != 1 {
		t.Errorf("user cfg Skills = %d, want 1", len(userCfg.Skills))
	}
	projCfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load project cfg: %v", err)
	}
	if len(projCfg.Skills) != 0 {
		t.Errorf("project cfg should be empty, got %d skills", len(projCfg.Skills))
	}
}

func TestRunBulkInstallCatalogFetchFails(t *testing.T) {
	dir := t.TempDir()
	stubCatalogFns(t, nil, fmt.Errorf("network error"))

	err := runBulkInstall(dir, "project", nil)
	if err == nil {
		t.Fatal("expected error when catalog fetch fails")
	}
	if !strings.Contains(err.Error(), "catalog") {
		t.Errorf("error should mention catalog, got: %v", err)
	}
}

func TestSkillListCatalogFetchFallback(t *testing.T) {
	stubUserConfigDir(t)

	projDir := t.TempDir()
	t.Chdir(projDir)
	projCfg := &project.Config{
		Skills: []project.SkillEntry{
			{Name: "pm", Source: DefaultSource},
		},
	}
	if err := project.Save(projDir, projCfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Stub catalog to fail.
	stubCatalogFns(t, nil, fmt.Errorf("network error"))

	// Capture stderr for warning.
	oldErr := os.Stderr
	re, we, _ := os.Pipe()
	os.Stderr = we

	output, err := runSkillListCmd(t, false)

	_ = we.Close()
	os.Stderr = oldErr
	errOut, _ := io.ReadAll(re)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fall back to installed-only view.
	if !strings.Contains(output, "pm") {
		t.Errorf("fallback should show installed skills: %s", output)
	}
	// Should print warning to stderr.
	if !strings.Contains(string(errOut), "Warning") {
		t.Errorf("expected warning on stderr, got: %s", string(errOut))
	}
}
