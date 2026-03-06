## Why

The current `devpilot commit` command is a basic commit message generator: it blindly stages everything with `git add .`, sends only file names and stat summaries to Claude (no actual diff content), and produces a single commit regardless of how many unrelated changes exist. This leads to low-quality commit messages, bloated commits mixing unrelated changes, and accidental commits of sensitive files like `.env`. The command also provides poor UX — no progress feedback during AI generation, no diff preview, plain unformatted output, and minimal post-commit information.

## What Changes

- **Intelligent commit planning**: Instead of generating a single commit message, Claude analyzes the full diff and produces a structured plan that groups files into multiple logical, atomic commits — each with its own conventional commit message.
- **Sensitive file exclusion**: Claude identifies files that should not be committed (secrets, debug artifacts, build outputs) and excludes them from the plan, with reasons shown to the user.
- **Full diff context**: Send actual `git diff --cached` content to Claude (with truncation for large diffs) instead of just file names and stat, dramatically improving message quality.
- **Bubble Tea interactive UI**: Replace plain text prompts with a Bubble Tea TUI that shows a spinner during analysis, formatted commit plan with colored file statuses and diff stats, real-time progress during commit execution, and a rich summary on completion.
- **Single commit simplification**: When Claude determines all changes belong to one commit, the UI simplifies to a streamlined single-commit flow.

## Capabilities

### New Capabilities
- `commit-planning`: AI-powered analysis of staged changes to produce a structured commit plan — grouping files into atomic commits, generating conventional commit messages, and identifying files to exclude.
- `commit-tui`: Bubble Tea interactive interface for the commit workflow — spinner during analysis, formatted plan display, execution progress, and completion summary. Reuses lipgloss styles from the existing taskrunner TUI.

### Modified Capabilities

(none — no existing spec-level requirements are changing)

## Impact

- **Code**: `internal/generate/commit.go` rewritten; new files for Bubble Tea model, view, and plan parsing added to `internal/generate/`
- **Prompts**: `prompts/commit.tmpl` updated to request JSON output with commit grouping and exclusion fields
- **Dependencies**: No new external dependencies — reuses existing `bubbletea`, `bubbles`, and `lipgloss` already in `go.mod`
- **CLI**: `devpilot commit` flags unchanged (`-m`, `--model`, `--dry-run`); behavior changes are additive
- **Git workflow**: Now runs `git reset <file>` for excluded files and multiple `git add <files> && git commit` sequences instead of a single commit
