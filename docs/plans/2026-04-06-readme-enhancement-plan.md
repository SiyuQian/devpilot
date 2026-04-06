# README Enhancement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the pre-collection approach in `devpilot readme` with LLM-driven exploration via `claude -p` with read-only tools, producing richer READMEs for any project type.

**Architecture:** Go code builds a prompt from a template (passing only the existing README if present), then invokes `claude -p` with `--allowedTools` restricted to read-only tools (Read, Glob, Grep, Bash). The LLM autonomously explores the project and generates the README. All file-collection Go functions are removed.

**Tech Stack:** Go, Cobra CLI, Go `text/template`, Claude Code CLI (`claude -p`)

---

### Task 1: Add `GenerateWithTools` to `claude.go`

**Files:**
- Modify: `internal/generate/claude.go`
- Test: `internal/generate/claude_test.go`

- [ ] **Step 1: Write the failing test for `GenerateWithTools`**

Add to `internal/generate/claude_test.go`:

```go
func TestBuildReadmeArgs(t *testing.T) {
	args := buildReadmeArgs("claude-haiku-4-5")
	// Must contain -p and --print
	if args[0] != "-p" {
		t.Errorf("first arg should be -p, got %q", args[0])
	}

	// Must contain --allowedTools
	foundAllowed := false
	for i, a := range args {
		if a == "--allowedTools" {
			foundAllowed = true
			if !strings.Contains(args[i+1], "Read") {
				t.Errorf("allowedTools should contain Read, got %q", args[i+1])
			}
			if !strings.Contains(args[i+1], "Glob") {
				t.Errorf("allowedTools should contain Glob, got %q", args[i+1])
			}
		}
	}
	if !foundAllowed {
		t.Error("--allowedTools flag not found")
	}

	// Must contain --model
	foundModel := false
	for i, a := range args {
		if a == "--model" {
			foundModel = true
			if args[i+1] != "claude-haiku-4-5" {
				t.Errorf("model arg = %q, want claude-haiku-4-5", args[i+1])
			}
		}
	}
	if !foundModel {
		t.Error("--model flag not found")
	}

	// Without model
	argsNoModel := buildReadmeArgs("")
	for _, a := range argsNoModel {
		if a == "--model" {
			t.Error("--model should not be present when model is empty")
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/generate/ -run TestBuildReadmeArgs -v`
Expected: FAIL — `buildReadmeArgs` undefined

- [ ] **Step 3: Implement `buildReadmeArgs` and `GenerateWithTools`**

Add to `internal/generate/claude.go`:

```go
// readmeAllowedTools defines the read-only tools for README generation.
const readmeAllowedTools = "Read,Glob,Grep,Bash(find:*,ls:*,head:*,cat:*,wc:*,git log:*,git remote:*,git describe:*)"

func buildReadmeArgs(model string) []string {
	args := []string{
		"-p", "--print",
		"--allowedTools", readmeAllowedTools,
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	return args
}

// GenerateWithTools calls `claude -p` with tools enabled for autonomous exploration.
func GenerateWithTools(ctx context.Context, prompt, model string) (string, error) {
	args := buildReadmeArgs(model)
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/generate/ -run TestBuildReadmeArgs -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/generate/claude.go internal/generate/claude_test.go
git commit -m "feat(generate): add GenerateWithTools for tool-enabled claude invocations"
```

---

### Task 2: Rewrite `readme.tmpl` Prompt

**Files:**
- Modify: `internal/generate/prompts/readme.tmpl`

- [ ] **Step 1: Replace `readme.tmpl` with the LLM-exploration prompt**

Write to `internal/generate/prompts/readme.tmpl`:

```
You are generating a README.md for the project in the current directory.
You have tools (Read, Glob, Grep, Bash) to explore the project. Use them to understand the project thoroughly before writing.

## Exploration Strategy

Follow these steps to gather context. Skip steps that don't apply.

1. Run `ls` to see the top-level structure. Read the package manifest (go.mod, package.json, pyproject.toml, Cargo.toml, etc.) to determine project type, name, and dependencies.
2. Read CLAUDE.md if it exists — this is an internal developer doc with rich project context (architecture, CLI commands, conventions). Extract facts but rewrite for end users. Do NOT copy sections verbatim — the audience is different.
3. Read Makefile or justfile for build/test/lint commands.
4. Read install scripts (install.sh, install.ps1) to document actual installation flags and options.
5. Read CI config (.github/workflows/*.yml or .gitlab-ci.yml) to understand supported platforms and test pipeline.
6. Read Dockerfile or docker-compose.yml if present for deployment context.
7. Read CONTRIBUTING.md if present — reference it but don't duplicate.
8. Read key source files as needed to understand:
   - CLI flags and subcommands (look for Cobra commands, argparse, clap, etc.)
   - Configuration file formats (look for config loading code or example configs)
   - Plugin/skill/extension systems (look for plugin directories, registries, install commands)
   - Public API surface (for libraries)
9. Run `git remote get-url origin` to get the repo URL for install instructions.
10. Run `git describe --tags --abbrev=0 2>/dev/null` to get the latest version tag if any.

## Output Format

Generate a complete README.md in raw markdown. Start with the `#` title line. No preamble, no commentary, no wrapping code fences around the entire output.

Include these sections in order (skip sections that don't apply to this project type):

1. **Title + badges** — preserve existing badges if updating an existing README
2. **One-line description** — what this project does and who it's for
3. **How it works** — brief architecture or workflow overview (keep it short)
4. **Features** — bullet list of key capabilities
5. **Prerequisites** — what must be installed first
6. **Installation** — actual commands with real flags (read install scripts to get the actual options)
7. **Quick start** — end-to-end workflow example from setup to first result
8. **Project-type-specific sections** (include only what applies):
   - CLI tools: full command reference with flags and examples, configuration file format with a commented example
   - Libraries: API usage with real import paths and code examples
   - Web apps: environment setup, deployment instructions
   - Frameworks: getting started guide, plugin/extension system docs
9. **Development** — build, test, lint commands for contributors
10. **Tech stack** — languages, frameworks, key dependencies with links
11. **License**

## Quality Rules

- Every claim must come from files you actually read — do not guess or hallucinate
- Show actual CLI flags, real config formats, real import paths from the source code
- No placeholder text ("your-project", "TODO", "example.com")
- No "N/A" sections — omit sections that don't apply entirely
- Keep each section proportional to its importance — don't pad short sections
- For end-to-end workflow examples, show a realistic use case, not a trivial hello-world
{{- if .ExistingReadme }}

## Existing README

The project already has a README. Improve upon it: keep what's accurate (especially badges, links, and verified facts), fix what's outdated based on current project state, and fill in gaps.

<existing-readme>
{{ .ExistingReadme }}
</existing-readme>
{{- end }}
```

- [ ] **Step 2: Verify template parses**

Run: `go test ./internal/generate/ -run TestBuildReadmePrompt -v`
Expected: May fail (test uses old signature) — that's OK, we fix it in the next task.

- [ ] **Step 3: Commit**

```bash
git add internal/generate/prompts/readme.tmpl
git commit -m "feat(generate): rewrite readme prompt for LLM-driven exploration"
```

---

### Task 3: Simplify `readme.go`

**Files:**
- Modify: `internal/generate/readme.go`
- Test: `internal/generate/claude_test.go`

- [ ] **Step 1: Update `TestBuildReadmePrompt` for the new signature**

Replace the existing test in `internal/generate/claude_test.go`:

```go
func TestBuildReadmePrompt(t *testing.T) {
	prompt, err := buildReadmePrompt("# Old Readme\nSome content")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt, "Old Readme") {
		t.Error("prompt should contain existing readme")
	}
	if !strings.Contains(prompt, "Exploration Strategy") {
		t.Error("prompt should contain exploration strategy section")
	}
}

func TestBuildReadmePromptEmpty(t *testing.T) {
	prompt, err := buildReadmePrompt("")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(prompt, "existing-readme") {
		t.Error("prompt should not contain existing-readme section when empty")
	}
	if !strings.Contains(prompt, "Exploration Strategy") {
		t.Error("prompt should contain exploration strategy section")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/generate/ -run TestBuildReadmePrompt -v`
Expected: FAIL — `buildReadmePrompt` signature mismatch (currently takes 3 args)

- [ ] **Step 3: Rewrite `readme.go`**

Replace `internal/generate/readme.go` with:

```go
package generate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var readmeTmpl = template.Must(template.ParseFS(promptsFS, "prompts/readme.tmpl"))

type readmeData struct {
	ExistingReadme string
}

func buildReadmePrompt(existingReadme string) (string, error) {
	var buf bytes.Buffer
	err := readmeTmpl.Execute(&buf, readmeData{
		ExistingReadme: existingReadme,
	})
	return buf.String(), err
}

// RunReadme generates a README by letting Claude explore the project with tools.
func RunReadme(ctx context.Context, model string, dryRun bool) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	var existingReadme string
	if data, err := os.ReadFile(filepath.Join(dir, "README.md")); err == nil {
		existingReadme = string(data)
	}

	prompt, err := buildReadmePrompt(existingReadme)
	if err != nil {
		return fmt.Errorf("build prompt: %w", err)
	}

	fmt.Println("Generating README (exploring project)...")
	content, err := GenerateWithTools(ctx, prompt, model)
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
	_, _ = fmt.Scanln(&choice)
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

- [ ] **Step 4: Remove `TestCollectFileTree` from tests**

Remove the `TestCollectFileTree` function from `internal/generate/claude_test.go` since `collectFileTree` no longer exists.

- [ ] **Step 5: Run all tests**

Run: `go test ./internal/generate/ -v`
Expected: All PASS

- [ ] **Step 6: Run lint**

Run: `make lint`
Expected: PASS (no unused imports, no dead code)

- [ ] **Step 7: Commit**

```bash
git add internal/generate/readme.go internal/generate/claude_test.go
git commit -m "refactor(generate): simplify readme.go to use LLM-driven exploration"
```

---

### Task 4: Update Timeout in `commands.go`

**Files:**
- Modify: `internal/generate/commands.go`

- [ ] **Step 1: Change readme timeout from 2 minutes to 5 minutes**

In `internal/generate/commands.go`, change line 63:

```go
// Before:
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
// After:
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
```

This is the timeout inside `readmeCmd.Run`. The `commitCmd` timeout on line 44 stays at `2*time.Minute`.

- [ ] **Step 2: Run all tests**

Run: `make test`
Expected: All PASS

- [ ] **Step 3: Run lint**

Run: `make lint`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/generate/commands.go
git commit -m "chore(generate): increase readme timeout to 5min for tool-based exploration"
```

---

### Task 5: Manual Integration Test

**Files:** None (manual test)

- [ ] **Step 1: Dry-run on DevPilot itself**

Run: `go run ./cmd/devpilot readme --dry-run`

Verify:
- Output starts with `Generating README (exploring project)...`
- Claude explores files (you'll see it take a few seconds as it makes tool calls)
- Generated README contains: title, description, features, installation, CLI reference, architecture, development commands, tech stack
- Generated README includes actual flags from `devpilot run --help`
- Generated README documents `.devpilot.yaml` config format (if Claude found it relevant)
- Generated README mentions the skills system with `devpilot skill add`

- [ ] **Step 2: Verify quality vs current README**

Compare the dry-run output against the existing `README.md`. The new output should:
- Have an end-to-end workflow example (not just "Quick Start" with 3 commands)
- Include config file documentation
- Include skills system documentation
- Not have placeholder text

- [ ] **Step 3: Dry-run on a different project (optional)**

Navigate to a different project (e.g., a Node.js or Python project) and run:

```bash
/path/to/devpilot readme --dry-run
```

Verify it produces a project-appropriate README, not a Go-centric one.
