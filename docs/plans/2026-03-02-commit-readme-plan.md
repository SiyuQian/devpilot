# Commit & README Commands Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `devpilot commit` and `devpilot readme` commands that use `claude --print` to generate AI-powered commit messages and README files.

**Architecture:** New `internal/generate` package following existing RegisterCommands pattern. A shared `claude.go` wrapper calls `claude --print --model <model>`. Prompt templates embedded via `go:embed`. Config extended with `Models` map in `.devpilot.json`.

**Tech Stack:** Go, Cobra, os/exec, text/template, go:embed

---

### Task 1: Extend Config with Models field

**Files:**
- Modify: `internal/project/config.go`
- Test: `internal/project/config_test.go`

**Step 1: Write the failing test**

```go
// In config_test.go, add:
func TestLoadConfigWithModels(t *testing.T) {
	dir := t.TempDir()
	data := `{"board":"myboard","models":{"commit":"claude-haiku-4-5","default":"claude-sonnet-4-6"}}`
	os.WriteFile(filepath.Join(dir, ".devpilot.json"), []byte(data), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Models["commit"] != "claude-haiku-4-5" {
		t.Errorf("got %q, want claude-haiku-4-5", cfg.Models["commit"])
	}
	if cfg.Models["default"] != "claude-sonnet-4-6" {
		t.Errorf("got %q, want claude-sonnet-4-6", cfg.Models["default"])
	}
}

func TestModelForCommand(t *testing.T) {
	cfg := &Config{Models: map[string]string{"commit": "claude-haiku-4-5", "default": "claude-sonnet-4-6"}}
	if got := cfg.ModelFor("commit"); got != "claude-haiku-4-5" {
		t.Errorf("got %q, want claude-haiku-4-5", got)
	}
	if got := cfg.ModelFor("readme"); got != "claude-sonnet-4-6" {
		t.Errorf("got %q, want claude-sonnet-4-6 (default fallback)", got)
	}
	if got := cfg.ModelFor("unknown"); got != "claude-sonnet-4-6" {
		t.Errorf("got %q, want claude-sonnet-4-6 (default fallback)", got)
	}

	empty := &Config{}
	if got := empty.ModelFor("commit"); got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/project/ -run TestLoadConfigWithModels -v`
Expected: FAIL — `Models` field doesn't exist yet

**Step 3: Implement**

Add to `internal/project/config.go`:

```go
type Config struct {
	Board  string            `json:"board,omitempty"`
	Models map[string]string `json:"models,omitempty"`
}

// ModelFor returns the configured model for a command, falling back to "default", then "".
func (c *Config) ModelFor(command string) string {
	if c.Models == nil {
		return ""
	}
	if m, ok := c.Models[command]; ok {
		return m
	}
	return c.Models["default"]
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/project/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/project/config.go internal/project/config_test.go
git commit -m "feat(config): add Models map and ModelFor helper"
```

---

### Task 2: Create claude.go wrapper

**Files:**
- Create: `internal/generate/claude.go`
- Test: `internal/generate/claude_test.go`

**Step 1: Write the failing test**

```go
package generate

import (
	"context"
	"strings"
	"testing"
)

func TestCleanOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "feat: add feature", "feat: add feature"},
		{"markdown fences", "```\nfeat: add feature\n```", "feat: add feature"},
		{"leading whitespace", "\n\n  feat: add feature\n\n", "feat: add feature"},
		{"ai preamble", "Here is the commit message:\nfeat: add feature", "feat: add feature"},
		{"ai preamble with blank", "Here's a commit message:\n\nfeat: add feature", "feat: add feature"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanOutput(tt.input)
			if got != tt.want {
				t.Errorf("cleanOutput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildArgs(t *testing.T) {
	args := buildArgs("claude-haiku-4-5")
	if args[0] != "--print" {
		t.Errorf("first arg should be --print, got %q", args[0])
	}
	found := false
	for i, a := range args {
		if a == "--model" {
			found = true
			if args[i+1] != "claude-haiku-4-5" {
				t.Errorf("model arg = %q, want claude-haiku-4-5", args[i+1])
			}
		}
	}
	if !found {
		t.Error("--model flag not found")
	}

	argsNoModel := buildArgs("")
	for _, a := range argsNoModel {
		if a == "--model" {
			t.Error("--model should not be present when model is empty")
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/generate/ -run 'TestCleanOutput|TestBuildArgs' -v`
Expected: FAIL — package doesn't exist

**Step 3: Implement**

Create `internal/generate/claude.go`:

```go
package generate

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Generate calls `claude --print` with the given prompt and optional model.
// Returns the cleaned output text.
func Generate(ctx context.Context, prompt, model string) (string, error) {
	args := buildArgs(model)
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, "claude", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude failed: %w\nstderr: %s", err, stderr.String())
	}

	return cleanOutput(stdout.String()), nil
}

func buildArgs(model string) []string {
	args := []string{"--print"}
	if model != "" {
		args = append(args, "--model", model)
	}
	return args
}

var preambleRe = regexp.MustCompile(`(?i)^(here('s| is).*?:)\s*\n`)

func cleanOutput(s string) string {
	s = strings.TrimSpace(s)
	// Strip markdown code fences
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	// Strip language hints after opening fence
	if idx := strings.Index(s, "\n"); idx >= 0 && !strings.Contains(s[:idx], " ") && len(s[:idx]) < 15 {
		first := strings.TrimSpace(s[:idx])
		if first == "markdown" || first == "text" || first == "" {
			s = s[idx+1:]
		}
	}
	// Strip AI preamble like "Here is the commit message:"
	s = preambleRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/generate/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/generate/claude.go internal/generate/claude_test.go
git commit -m "feat(generate): add claude --print wrapper with output cleaning"
```

---

### Task 3: Create commit prompt template and logic

**Files:**
- Create: `internal/generate/prompts/commit.tmpl`
- Create: `internal/generate/commit.go`
- Test: `internal/generate/commit_test.go`

**Step 1: Create the prompt template**

Create `internal/generate/prompts/commit.tmpl`:

```
Write a concise commit message in conventional commits format for these changes:

Files changed:
{{.FileList}}

Git diff stat:
{{.DiffStat}}
{{if .Context}}
Additional context: {{.Context}}
{{end}}
Requirements:
- Use conventional commits format (feat:, fix:, refactor:, docs:, chore:, etc.)
- Keep the first line under 72 characters
- Add bullet points for details if the change is complex
- Be specific about what was changed and why
- Output ONLY the raw commit message text, no markdown formatting
```

**Step 2: Write the failing test**

```go
package generate

import (
	"strings"
	"testing"
)

func TestBuildCommitPrompt(t *testing.T) {
	prompt, err := buildCommitPrompt("file1.go\nfile2.go", "2 files changed, 10 insertions", "fixing auth bug")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "file1.go") {
		t.Error("prompt should contain file list")
	}
	if !strings.Contains(prompt, "10 insertions") {
		t.Error("prompt should contain diff stat")
	}
	if !strings.Contains(prompt, "fixing auth bug") {
		t.Error("prompt should contain context")
	}
	if !strings.Contains(prompt, "conventional commits") {
		t.Error("prompt should mention conventional commits")
	}
}

func TestBuildCommitPromptNoContext(t *testing.T) {
	prompt, err := buildCommitPrompt("file1.go", "1 file changed", "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(prompt, "Additional context") {
		t.Error("prompt should not contain context section when empty")
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/generate/ -run TestBuildCommitPrompt -v`
Expected: FAIL — `buildCommitPrompt` doesn't exist

**Step 4: Implement commit.go**

```go
package generate

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

//go:embed prompts/commit.tmpl
var commitTmplFS embed.FS

var commitTmpl = template.Must(template.ParseFS(commitTmplFS, "prompts/commit.tmpl"))

type commitData struct {
	FileList string
	DiffStat string
	Context  string
}

func buildCommitPrompt(fileList, diffStat, context string) (string, error) {
	var buf bytes.Buffer
	err := commitTmpl.Execute(&buf, commitData{
		FileList: fileList,
		DiffStat: diffStat,
		Context:  context,
	})
	return buf.String(), err
}

// gitOutput runs a git command and returns trimmed stdout.
func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RunCommit stages all changes, generates a commit message via Claude, and commits.
func RunCommit(ctx context.Context, model, userContext string, dryRun bool) error {
	// Stage all changes
	if out, err := exec.Command("git", "add", ".").CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w\n%s", err, out)
	}

	// Check for staged changes
	diffCached, err := gitOutput("diff", "--cached", "--stat")
	if err != nil {
		return err
	}
	if diffCached == "" {
		fmt.Println("No changes to commit.")
		return nil
	}

	fileList, err := gitOutput("diff", "--cached", "--name-only")
	if err != nil {
		return err
	}

	prompt, err := buildCommitPrompt(fileList, diffCached, userContext)
	if err != nil {
		return fmt.Errorf("build prompt: %w", err)
	}

	fmt.Println("Generating commit message...")
	message, err := Generate(ctx, prompt, model)
	if err != nil {
		return err
	}

	fmt.Printf("\n%s\n\n", message)

	if dryRun {
		fmt.Println("(dry-run: not committing)")
		return nil
	}

	// Prompt user for confirmation
	fmt.Print("Commit with this message? [y/n/e(dit)] ")
	var choice string
	fmt.Scanln(&choice)
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "y", "yes", "":
		// proceed
	case "e", "edit":
		edited, err := editInTerminal(message)
		if err != nil {
			return err
		}
		message = edited
	default:
		fmt.Println("Aborted.")
		return nil
	}

	if out, err := exec.Command("git", "commit", "-m", message).CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out)
	}
	fmt.Println("Committed.")
	return nil
}

// editInTerminal opens $EDITOR for the user to edit the message.
func editInTerminal(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	tmpFile, err := os.CreateTemp("", "devpilot-commit-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(initial); err != nil {
		return "", err
	}
	tmpFile.Close()

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor: %w", err)
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/generate/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/generate/commit.go internal/generate/commit_test.go internal/generate/prompts/commit.tmpl
git commit -m "feat(generate): add commit command logic and prompt template"
```

---

### Task 4: Create readme prompt template and logic

**Files:**
- Create: `internal/generate/prompts/readme.tmpl`
- Create: `internal/generate/readme.go`
- Test: `internal/generate/readme_test.go`

**Step 1: Create the prompt template**

Create `internal/generate/prompts/readme.tmpl`:

```
Generate a professional README.md for this project.

Project structure:
{{.FileTree}}
{{if .PackageInfo}}
Package metadata:
{{.PackageInfo}}
{{end}}{{if .ExistingReadme}}
Existing README (improve upon it):
{{.ExistingReadme}}
{{end}}
Requirements:
- Include: project title, description, features, installation, usage, tech stack
- Be concise and practical
- Use proper markdown formatting
- Output ONLY the README content, no wrapping code fences
```

**Step 2: Write the failing test**

```go
package generate

import (
	"strings"
	"testing"
)

func TestBuildReadmePrompt(t *testing.T) {
	prompt, err := buildReadmePrompt("src/\n  main.go\n  util.go", "module example.com/foo", "# Old Readme")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "main.go") {
		t.Error("prompt should contain file tree")
	}
	if !strings.Contains(prompt, "module example.com/foo") {
		t.Error("prompt should contain package info")
	}
	if !strings.Contains(prompt, "Old Readme") {
		t.Error("prompt should contain existing readme")
	}
}

func TestCollectFileTree(t *testing.T) {
	// This test uses the actual project directory structure,
	// just verifying it doesn't error and returns something non-empty.
	tree, err := collectFileTree(".")
	if err != nil {
		t.Fatal(err)
	}
	if tree == "" {
		t.Error("file tree should not be empty")
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/generate/ -run 'TestBuildReadmePrompt|TestCollectFileTree' -v`
Expected: FAIL

**Step 4: Implement readme.go**

```go
package generate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed prompts/readme.tmpl
var readmeTmplFS embed.FS // reuse the embed import from commit.go — actually needs separate var

var readmeTmpl = template.Must(template.ParseFS(commitTmplFS, "prompts/readme.tmpl"))

type readmeData struct {
	FileTree       string
	PackageInfo    string
	ExistingReadme string
}

func buildReadmePrompt(fileTree, packageInfo, existingReadme string) (string, error) {
	var buf bytes.Buffer
	err := readmeTmpl.Execute(&buf, readmeData{
		FileTree:       fileTree,
		PackageInfo:    packageInfo,
		ExistingReadme: existingReadme,
	})
	return buf.String(), err
}

// collectFileTree returns a simple file listing, excluding common non-source dirs.
func collectFileTree(dir string) (string, error) {
	var lines []string
	excludes := map[string]bool{
		".git": true, "node_modules": true, "vendor": true,
		"bin": true, ".claude": true, "__pycache__": true,
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		rel, _ := filepath.Rel(dir, path)
		if rel == "." {
			return nil
		}
		// Skip excluded directories
		if d.IsDir() && excludes[d.Name()] {
			return filepath.SkipDir
		}
		lines = append(lines, rel)
		if len(lines) >= 200 {
			return filepath.SkipAll
		}
		return nil
	})
	return strings.Join(lines, "\n"), err
}

// collectPackageInfo reads the first 30 lines of go.mod, package.json, or pyproject.toml.
func collectPackageInfo(dir string) string {
	candidates := []string{"go.mod", "package.json", "pyproject.toml", "Cargo.toml"}
	for _, name := range candidates {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		lines := strings.SplitN(string(data), "\n", 31)
		if len(lines) > 30 {
			lines = lines[:30]
		}
		return fmt.Sprintf("(%s)\n%s", name, strings.Join(lines, "\n"))
	}
	return ""
}

// RunReadme generates a README.md via Claude.
func RunReadme(ctx context.Context, model string, dryRun bool) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	fileTree, err := collectFileTree(dir)
	if err != nil {
		return err
	}

	packageInfo := collectPackageInfo(dir)

	var existingReadme string
	if data, err := os.ReadFile(filepath.Join(dir, "README.md")); err == nil {
		existingReadme = string(data)
	}

	prompt, err := buildReadmePrompt(fileTree, packageInfo, existingReadme)
	if err != nil {
		return fmt.Errorf("build prompt: %w", err)
	}

	fmt.Println("Generating README...")
	content, err := Generate(ctx, prompt, model)
	if err != nil {
		return err
	}

	fmt.Printf("\n%s\n\n", content)

	if dryRun {
		fmt.Println("(dry-run: not writing)")
		return nil
	}

	fmt.Print("Save to README.md? [y/n] ")
	var choice string
	fmt.Scanln(&choice)
	if strings.ToLower(strings.TrimSpace(choice)) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(content+"\n"), 0644); err != nil {
		return err
	}
	fmt.Println("Saved to README.md")
	return nil
}
```

**Note:** The embed FS needs to be shared. In actual implementation, use a single `go:embed` directive for the `prompts` directory. The template loading should be:

```go
//go:embed prompts
var promptsFS embed.FS

var commitTmpl = template.Must(template.ParseFS(promptsFS, "prompts/commit.tmpl"))
var readmeTmpl = template.Must(template.ParseFS(promptsFS, "prompts/readme.tmpl"))
```

Move the `embed` declaration to `claude.go` or a shared `embed.go` file.

**Step 5: Run tests**

Run: `go test ./internal/generate/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/generate/readme.go internal/generate/readme_test.go internal/generate/prompts/readme.tmpl
git commit -m "feat(generate): add readme command logic and prompt template"
```

---

### Task 5: Create commands.go and register in main

**Files:**
- Create: `internal/generate/commands.go`
- Modify: `cmd/devpilot/main.go`

**Step 1: Create commands.go**

```go
package generate

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/siyuqian/devpilot/internal/project"
)

func RegisterCommands(parent *cobra.Command) {
	commitCmd.Flags().StringP("message", "m", "", "Additional context for AI")
	commitCmd.Flags().String("model", "", "Override Claude model")
	commitCmd.Flags().Bool("dry-run", false, "Generate message without committing")

	readmeCmd.Flags().String("model", "", "Override Claude model")
	readmeCmd.Flags().Bool("dry-run", false, "Generate without writing file")

	parent.AddCommand(commitCmd)
	parent.AddCommand(readmeCmd)
}

func resolveModel(cmd *cobra.Command, command string) string {
	if m, _ := cmd.Flags().GetString("model"); m != "" {
		return m
	}
	dir, _ := os.Getwd()
	cfg, _ := project.Load(dir)
	return cfg.ModelFor(command)
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate an AI-powered commit message and commit",
	Long:  "Stages all changes with git add ., generates a conventional commit message using Claude AI, and commits after user confirmation.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		model := resolveModel(cmd, "commit")
		msg, _ := cmd.Flags().GetString("message")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := RunCommit(ctx, model, msg, dryRun); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

var readmeCmd = &cobra.Command{
	Use:   "readme",
	Short: "Generate a README.md using AI",
	Long:  "Analyzes your project structure and generates a professional README.md using Claude AI.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		model := resolveModel(cmd, "readme")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := RunReadme(ctx, model, dryRun); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}
```

**Step 2: Register in main.go**

Add to `cmd/devpilot/main.go`:

```go
import "github.com/siyuqian/devpilot/internal/generate"

// In main(), after existing registrations:
generate.RegisterCommands(rootCmd)
```

**Step 3: Build and verify**

Run: `make build && bin/devpilot commit --help && bin/devpilot readme --help`
Expected: Both commands show their help text with flags

**Step 4: Commit**

```bash
git add internal/generate/commands.go cmd/devpilot/main.go
git commit -m "feat: register commit and readme commands in CLI"
```

---

### Task 6: Integration test — end to end

**Files:**
- Modify: `internal/generate/commit_test.go`
- Modify: `internal/generate/readme_test.go`

**Step 1: Build and manual smoke test**

```bash
make build

# Test commit (in a git repo with changes):
echo "test" >> /tmp/test-file.txt
bin/devpilot commit --dry-run -m "test context"

# Test readme:
bin/devpilot readme --dry-run

# Test model override:
bin/devpilot commit --dry-run --model claude-haiku-4-5
```

**Step 2: Run all tests**

Run: `make test`
Expected: All tests pass

**Step 3: Final commit**

```bash
git add -A
git commit -m "test: add integration smoke tests for commit and readme"
```

---

## Summary of deliverables

| File | Action |
|------|--------|
| `internal/project/config.go` | Add `Models` field + `ModelFor()` |
| `internal/project/config_test.go` | Tests for Models config |
| `internal/generate/claude.go` | `Generate()` wrapper for `claude --print` |
| `internal/generate/claude_test.go` | Tests for output cleaning + arg building |
| `internal/generate/commit.go` | Commit logic + prompt building |
| `internal/generate/commit_test.go` | Tests for prompt building |
| `internal/generate/readme.go` | README logic + context gathering |
| `internal/generate/readme_test.go` | Tests for prompt building + file tree |
| `internal/generate/prompts/commit.tmpl` | Commit prompt template |
| `internal/generate/prompts/readme.tmpl` | README prompt template |
| `internal/generate/commands.go` | Cobra commands + registration |
| `cmd/devpilot/main.go` | Add `generate.RegisterCommands` |
