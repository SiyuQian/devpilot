package initcmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectProjectTypeGo(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/example/myapp\n\ngo 1.21\n"), 0644)

	pt := detectProjectType(dir)
	if pt.Name != "github.com/example/myapp" {
		t.Errorf("Name = %q, want %q", pt.Name, "github.com/example/myapp")
	}
	if pt.BuildCmd != "go build ./..." {
		t.Errorf("BuildCmd = %q, want %q", pt.BuildCmd, "go build ./...")
	}
	if pt.TestCmd != "go test ./..." {
		t.Errorf("TestCmd = %q, want %q", pt.TestCmd, "go test ./...")
	}
}

func TestDetectProjectTypeNode(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name": "my-app"}`), 0644)

	pt := detectProjectType(dir)
	if pt.Name != "my-app" {
		t.Errorf("Name = %q, want %q", pt.Name, "my-app")
	}
	if pt.BuildCmd != "npm run build" {
		t.Errorf("BuildCmd = %q, want %q", pt.BuildCmd, "npm run build")
	}
	if pt.TestCmd != "npm test" {
		t.Errorf("TestCmd = %q, want %q", pt.TestCmd, "npm test")
	}
}

func TestDetectProjectTypePython(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname = \"myapp\"\n"), 0644)

	pt := detectProjectType(dir)
	if pt.BuildCmd != "python -m build" {
		t.Errorf("BuildCmd = %q, want %q", pt.BuildCmd, "python -m build")
	}
	if pt.TestCmd != "python -m pytest" {
		t.Errorf("TestCmd = %q, want %q", pt.TestCmd, "python -m pytest")
	}
}

func TestDetectProjectTypePythonRequirements(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask\n"), 0644)

	pt := detectProjectType(dir)
	if pt.TestCmd != "python -m pytest" {
		t.Errorf("TestCmd = %q, want %q", pt.TestCmd, "python -m pytest")
	}
}

func TestDetectProjectTypeFallback(t *testing.T) {
	dir := t.TempDir()

	pt := detectProjectType(dir)
	if pt.Name != filepath.Base(dir) {
		t.Errorf("Name = %q, want %q", pt.Name, filepath.Base(dir))
	}
	if pt.BuildCmd != "" {
		t.Errorf("BuildCmd = %q, want empty", pt.BuildCmd)
	}
	if pt.TestCmd != "" {
		t.Errorf("TestCmd = %q, want empty", pt.TestCmd)
	}
}

func TestGenerateClaudeMD(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/example/myapp\n\ngo 1.21\n"), 0644)

	opts := GenerateOpts{Dir: dir, Interactive: false}
	if err := GenerateClaudeMD(opts); err != nil {
		t.Fatalf("GenerateClaudeMD failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "github.com/example/myapp") {
		t.Error("CLAUDE.md does not contain module name")
	}
	if !strings.Contains(content, "go build") {
		t.Error("CLAUDE.md does not contain build command")
	}
	if !strings.Contains(content, "go test") {
		t.Error("CLAUDE.md does not contain test command")
	}
}

func TestConfigureBoardNonInteractiveSkips(t *testing.T) {
	dir := t.TempDir()

	opts := GenerateOpts{Dir: dir, Interactive: false}
	if err := ConfigureBoard(opts, nil); err != nil {
		t.Fatalf("ConfigureBoard failed: %v", err)
	}

	// Should not have created .devpilot.json
	if _, err := os.Stat(filepath.Join(dir, ".devpilot.json")); !os.IsNotExist(err) {
		t.Error(".devpilot.json should not exist in non-interactive mode")
	}
}

func TestConfigureBoardInteractiveWithListBoards(t *testing.T) {
	dir := t.TempDir()

	input := strings.NewReader("1\n")
	opts := GenerateOpts{
		Dir:         dir,
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	listBoards := func() ([]Board, error) {
		return []Board{{Name: "Dev Board"}, {Name: "Other Board"}}, nil
	}

	if err := ConfigureBoard(opts, listBoards); err != nil {
		t.Fatalf("ConfigureBoard failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".devpilot.json"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), "Dev Board") {
		t.Errorf(".devpilot.json does not contain board name, got: %s", string(data))
	}
}

func TestConfigureBoardInteractiveFreeText(t *testing.T) {
	dir := t.TempDir()

	input := strings.NewReader("My Custom Board\n")
	opts := GenerateOpts{
		Dir:         dir,
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	if err := ConfigureBoard(opts, nil); err != nil {
		t.Fatalf("ConfigureBoard failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".devpilot.json"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !strings.Contains(string(data), "My Custom Board") {
		t.Errorf(".devpilot.json does not contain board name, got: %s", string(data))
	}
}

func TestCreateSkillDefault(t *testing.T) {
	dir := t.TempDir()

	opts := GenerateOpts{Dir: dir, Interactive: false}
	if err := CreateSkill(opts); err != nil {
		t.Fatalf("CreateSkill failed: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "my-skill", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "name: my-skill") {
		t.Errorf("SKILL.md does not contain correct name, got: %s", content)
	}
}

func TestCreateSkillInteractive(t *testing.T) {
	dir := t.TempDir()

	input := strings.NewReader("custom-skill\n")
	opts := GenerateOpts{
		Dir:         dir,
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	if err := CreateSkill(opts); err != nil {
		t.Fatalf("CreateSkill failed: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "custom-skill", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "name: custom-skill") {
		t.Errorf("SKILL.md does not contain correct name, got: %s", content)
	}
}

func TestCreateSkillInteractiveDefault(t *testing.T) {
	dir := t.TempDir()

	// Just press enter to accept default
	input := strings.NewReader("\n")
	opts := GenerateOpts{
		Dir:         dir,
		Interactive: true,
		Reader:      bufio.NewReader(input),
	}

	if err := CreateSkill(opts); err != nil {
		t.Fatalf("CreateSkill failed: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "my-skill", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Errorf("default skill not created: %v", err)
	}
}
