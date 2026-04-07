# README Enhancement Design

**Date:** 2026-04-06
**Goal:** Improve `devpilot readme` to generate more comprehensive, project-specific READMEs by letting the LLM autonomously explore the project.

## Problem

The current `devpilot readme` command pre-collects minimal context in Go code (file tree capped at 200 entries, first 30 lines of one package manifest, existing README), then passes it to `claude --print` as a single prompt. This approach:

- Produces generic READMEs missing CLI usage, configuration docs, architecture, and skill/plugin systems
- Cannot adapt its exploration strategy to different project types
- Requires maintaining Go-side file detection logic for every ecosystem

## Design: LLM-Driven Exploration

Instead of collecting files in Go and stuffing them into a prompt, switch to `claude -p` with tools enabled. The LLM explores the project itself using Read, Glob, Grep, and Bash, then writes the README based on what it discovers.

### 1. Execution Model

**Current:** `claude --print <prompt>` (no tools, pure text in/out)
**New:** `claude -p --print <prompt> --allowedTools "Read,Glob,Grep,Bash(find:*,ls:*,head:*,cat:*)" --permission-mode default`

The Go code becomes minimal:
1. Read existing README (if any) to pass as context
2. Build prompt from template
3. Exec `claude -p` with allowed tools
4. Show result, ask user to confirm save

```go
func RunReadme(ctx context.Context, model string, dryRun bool) error {
    dir, _ := os.Getwd()

    var existingReadme string
    if data, err := os.ReadFile(filepath.Join(dir, "README.md")); err == nil {
        existingReadme = string(data)
    }

    prompt, err := buildReadmePrompt(existingReadme)
    // ... exec claude -p with tools ...
}
```

**Removed:** `collectFileTree()`, `collectPackageInfo()`, `collectSupplementaryFiles()`, `readmeData` struct (all replaced by LLM tool calls).

### 2. Allowed Tools

Give the LLM read-only access to explore the project:

| Tool | Purpose |
|------|---------|
| `Read` | Read file contents (CLAUDE.md, configs, source code, etc.) |
| `Glob` | Find files by pattern (e.g., `**/*.go`, `Makefile`, `*.yml`) |
| `Grep` | Search for patterns (CLI flags, API endpoints, exports) |
| `Bash` | Limited to read-only commands: `find`, `ls`, `head`, `cat`, `wc`, `git log`, `git remote` |

No `Edit`, `Write`, or destructive Bash — the LLM only explores, never modifies.

### 3. Prompt Template

The prompt guides the LLM's exploration strategy and output format. Key sections:

```
You are generating a README.md for the project in the current directory.
You have tools (Read, Glob, Grep, Bash) to explore the project. Use them.

## Exploration Strategy

1. Start by running `ls` and reading the package manifest (go.mod, package.json, pyproject.toml, Cargo.toml, etc.) to determine project type
2. Read CLAUDE.md if it exists — this is an internal developer doc with rich project context. Extract facts but rewrite for end users. Do NOT copy verbatim.
3. Read Makefile/justfile for build commands, install scripts for installation steps
4. Read CI config (.github/workflows/, .gitlab-ci.yml) for supported platforms and test commands
5. Read key source files as needed to understand architecture and CLI flags
6. Read Dockerfile/docker-compose.yml if present for deployment info
7. Check for plugin/skill/extension systems (look for plugin directories, registries, install commands)

## Output Format

Generate a complete README.md in raw markdown. Start with `#` title line. No preamble, no code fences wrapping the output.

Required sections (include only what applies — skip sections that don't apply):
- Title + badges (preserve existing badges if updating)
- One-line description: what this project does and who it's for
- How it works: brief architecture or workflow overview
- Features: bullet list of key capabilities
- Prerequisites: what must be installed first
- Installation: actual commands with real flags (read install scripts to get the actual flags)
- Quick start: end-to-end workflow example from setup to first result
- Project-type-specific sections:
  - CLI tools: full command reference with flags, configuration file format with commented example
  - Libraries: API usage with real import paths and code examples
  - Web apps: environment setup, deployment instructions
  - Frameworks: getting started guide, plugin/extension system docs
- Development: build, test, lint commands
- Tech stack: languages, frameworks, key dependencies
- License

## Quality Rules

- Every claim must come from files you actually read — no guessing
- Show actual CLI flags, real config formats, real import paths
- No placeholder text ("your-project", "TODO", "example.com")
- No "N/A" sections — omit what doesn't apply
- Keep sections proportional to importance
- If updating an existing README, keep what's accurate, fix what's outdated, fill gaps
{{- if .ExistingReadme }}

## Existing README

The project already has a README. Improve upon it:

<existing-readme>
{{ .ExistingReadme }}
</existing-readme>
{{- end }}
```

### 4. Updated `readmeData` Struct

Simplified to only what Go needs to provide (the LLM discovers everything else):

```go
type readmeData struct {
    ExistingReadme string
}
```

### 5. `buildArgs` Changes

Update `Generate()` or add a new `GenerateWithTools()` to support tool-enabled invocations:

```go
func buildReadmeArgs(model string) []string {
    args := []string{
        "-p", "--print",
        "--allowedTools", "Read,Glob,Grep,Bash(find:*,ls:*,head:*,cat:*,wc:*,git log:*,git remote:*)",
    }
    if model != "" {
        args = append(args, "--model", model)
    }
    return args
}
```

### 6. Timeout

Increase from 2 minutes to 5 minutes. The LLM now makes multiple tool calls to explore, which takes longer than a single prompt. The commit command timeout stays at 2 minutes.

## Files Changed

| File | Change |
|------|--------|
| `internal/generate/readme.go` | Remove `collectFileTree()`, `collectPackageInfo()`, `readmeData` simplification. `RunReadme` now just builds prompt + execs `claude -p` with tools |
| `internal/generate/prompts/readme.tmpl` | Rewrite: exploration strategy + output format + quality rules |
| `internal/generate/claude.go` | Add `GenerateWithTools()` (or extend `Generate()`) to support `--allowedTools` flag |
| `internal/generate/commands.go` | Update readme timeout from 2min to 5min |
| `internal/generate/claude_test.go` | Remove `TestCollectFileTree`, `TestBuildReadmePrompt` updated for simplified struct, add `TestBuildReadmeArgs` |

## Not In Scope

- Screenshot/GIF generation for TUI dashboard
- Multi-file README generation (e.g., separate CONTRIBUTING.md)
- Monorepo-specific README generation — treated as single projects for now
