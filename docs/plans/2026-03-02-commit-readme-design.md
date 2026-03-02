# Design: `devpilot commit` + `devpilot readme`

**Date:** 2026-03-02
**Status:** Approved

## Overview

Add two new CLI commands to DevPilot that generate content using Claude AI via `claude --print`: AI-powered commit messages and README generation. Inspired by superclaude's features but implemented natively in Go, following DevPilot's existing architecture.

## Configuration

Extend `.devpilot.json` with a `models` map to control which Claude model is used per command:

```json
{
  "board": "devpilot",
  "models": {
    "commit": "claude-haiku-4-5",
    "readme": "claude-sonnet-4-6",
    "default": "claude-sonnet-4-6"
  }
}
```

- Each command looks up its own key first, then falls back to `"default"`, then omits `--model` (using claude CLI's default).
- CLI flag `--model` overrides config for a single invocation.

## `devpilot commit`

### Usage

```bash
devpilot commit                           # AI commit message, auto git add .
devpilot commit -m "extra context"        # Provide context to AI
devpilot commit --dry-run                 # Generate message only, don't commit
devpilot commit --model claude-haiku-4-5  # Override model
```

### Flow

1. Run `git add .` to stage all changes
2. Check `git diff --cached` — abort if no changes
3. Collect context:
   - `git diff --cached --stat` (diff summary)
   - `git diff --cached --name-only` (file list)
4. Build prompt from embedded Go template, filling in diff info + optional `-m` context
5. Call `claude --print --model <model>` with the prompt
6. Clean output (strip markdown fences, AI preamble)
7. Display generated message, prompt user: confirm (y), reject (n), or edit (e)
8. Execute `git commit -m "<message>"`

### Prompt Template (commit.tmpl)

```
Write a concise commit message in conventional commits format for these changes:

Files changed:
{{.FileList}}

Git diff stat:
{{.DiffStat}}

{{if .Context}}Additional context: {{.Context}}{{end}}

Requirements:
- Use conventional commits format (feat:, fix:, refactor:, docs:, chore:, etc.)
- Keep the first line under 72 characters
- Add bullet points for details if the change is complex
- Be specific about what was changed and why
- Output ONLY the raw commit message text, no markdown formatting
```

## `devpilot readme`

### Usage

```bash
devpilot readme                           # Generate README.md
devpilot readme --dry-run                 # Preview only
devpilot readme --model claude-sonnet-4-6 # Override model
```

### Flow

1. Collect project context:
   - File tree (exclude .git, node_modules, vendor, bin, .claude)
   - First 30 lines of `go.mod` / `package.json` / `pyproject.toml` (whichever exists)
   - Existing `README.md` content (if any, as reference for AI to improve upon)
2. Build prompt from embedded Go template
3. Call `claude --print --model <model>` with the prompt
4. Display generated README
5. Prompt user: save to README.md? (y/n)
6. Write file if confirmed

### Prompt Template (readme.tmpl)

```
Generate a professional README.md for this project.

Project structure:
{{.FileTree}}

{{if .PackageInfo}}Package metadata:
{{.PackageInfo}}{{end}}

{{if .ExistingReadme}}Existing README (use as reference, improve upon it):
{{.ExistingReadme}}{{end}}

Requirements:
- Include: project title, description, features, installation, usage, tech stack
- Be concise and practical
- Use proper markdown formatting
- Output ONLY the README content
```

## Package Structure

```
internal/
  generate/
    commands.go       # RegisterCommands: adds "commit" and "readme" to root
    commit.go         # commit logic + prompt building
    readme.go         # readme logic + context gathering
    claude.go         # claude --print invocation wrapper
    prompts/
      commit.tmpl     # embedded via go:embed
      readme.tmpl     # embedded via go:embed
  project/
    config.go         # extend Config struct with Models map
```

### Config Change

```go
type Config struct {
    Board  string            `json:"board,omitempty"`
    Models map[string]string `json:"models,omitempty"`
}
```

### Claude Wrapper

```go
// generate/claude.go
func Generate(ctx context.Context, prompt, model string) (string, error)
```

- Calls `claude --print --model <model> <prompt>` via `os/exec`
- If model is empty, omits `--model` flag
- Returns cleaned output (stripped of markdown fences and preamble)
- Respects context cancellation for timeout support

## Command Registration

In `cmd/devpilot/main.go`, add:

```go
generate.RegisterCommands(rootCmd)
```

## What We're NOT Doing

- No auto-push after commit (user pushes manually)
- No git notes annotation
- No changelog generation (future scope)
- No Anthropic API direct integration (claude CLI handles auth)
