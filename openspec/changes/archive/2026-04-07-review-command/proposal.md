## Why

The current code review capability is embedded inside the task runner (`internal/taskrunner/reviewer.go`) with a minimal prompt (`"Code review: <url>"`). This produces low-quality, shallow reviews. We need a standalone `devpilot review` command that delivers high-quality, thorough code reviews using Claude Opus 4.6 with extended thinking — usable both as a standalone CLI tool for any project and as the review step in the automated runner pipeline.

## What Changes

- Add a new `devpilot review <pr-url>` CLI command that runs a thorough AI-powered code review
- Use `claude -p --thinking --model claude-opus-4-6-20250415` for deep reasoning during review
- Ship a high-quality review prompt file (LLM instructions) that guides the review process — covering code correctness, security, performance, maintainability, and style
- Include a structured review comment template for consistent output formatting
- Build robust context gathering: fetch PR diff, read changed files, detect project conventions (CLAUDE.md, linters, test patterns) so the review works well on any project
- Integrate as a drop-in replacement for the runner's existing review step
- Support `--dry-run` to preview review output without posting comments

## Capabilities

### New Capabilities

- `review-command`: Standalone `devpilot review` CLI command with PR URL input, context gathering, prompt assembly, and Claude execution
- `review-prompt`: High-quality LLM review instructions file and structured comment template that guide the review process

### Modified Capabilities

_(none — the runner integration will call the new review package but no spec-level requirements change for existing capabilities)_

## Impact

- **New package**: `internal/review/` with commands, prompt builder, context gatherer, and executor integration
- **CLI**: New `review` subcommand registered in `cmd/devpilot/main.go`
- **Task runner**: `internal/taskrunner/reviewer.go` updated to delegate to the new review package
- **Dependencies**: Reuses existing `Executor` from `internal/taskrunner/` (or extracts to shared package)
- **External tools**: Depends on `gh` CLI for PR data fetching and `claude` CLI for execution
