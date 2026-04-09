# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**DevPilot** is a Go CLI tool and collection of skills for automating development workflows with Claude Code. The core workflow: write a plan, queue it in Trello or GitHub Issues, and let an autonomous runner execute it via `claude -p`, creating branches and PRs automatically.

## Repository Structure

Standard Go project layout: `cmd/devpilot/` for the CLI entry point, `internal/` for all packages.

**Package organization rules:**
- Each `internal/` package is a self-contained domain (e.g. `auth`, `trello`, `taskrunner`)
- Each domain package owns its own Cobra commands in `commands.go` — there is no central `cli/` routing layer
- External service clients (API, HTTP) live in the same package as their domain logic
- Shared project-level config lives in `internal/project/`

**Other top-level directories:**
- `skills/` — Distributable product skills (each `devpilot-<name>/` dir with `SKILL.md`)
- `.claude/skills/` — Project-local OpenSpec workflow skills (not distributed)
- `.github/workflows/` — CI/CD (test + release pipelines)
- `docs/plans/` — Design and implementation plan documents
- `docs/rejected/` — Rejected/deferred idea records (read by PM skill to avoid re-recommending)

## Build & Development Commands

```bash
make build                         # Build binary to bin/devpilot
make test                          # Run all tests (go test ./...)
make lint                          # Run golangci-lint (must pass before commit)
make lint-fix                      # Auto-fix lint issues where possible
make run ARGS="--help"             # Build and run with arguments
make clean                         # Remove bin/
```

**Important:** Always run `make test` and `make lint` before committing. Both must pass — CI enforces this on every PR.

Run a single test:
```bash
go test ./internal/skillmgr/ -run TestInstallSkill   # Single test by name
go test ./internal/taskrunner/ -v                     # Single package, verbose
```

### CLI Commands

```bash
devpilot login trello                # Authenticate with Trello (API key + token)
devpilot logout trello               # Remove stored credentials
devpilot status                      # Show authentication status for all services

devpilot init                        # Interactive project setup wizard
devpilot init -y                     # Accept all defaults

devpilot push <plan.md> --board "Board Name"              # Create Trello card from plan file
devpilot push <plan.md> --board "Board Name" --list "Ready"  # Specify target list (default: Ready)

devpilot run --board "Board Name"                          # Start autonomous task runner (TUI mode)
devpilot run --board "Board Name" --no-tui                 # Plain text output (no dashboard)
devpilot run --board "Board Name" --once --dry-run         # Test with one card, no execution
devpilot run --board "Board Name" --interval 60            # Poll every 60s (default: 300)
devpilot run --board "Board Name" --timeout 45             # 45min per-task timeout (default: 30)
devpilot run --board "Board Name" --review-timeout 0       # Disable auto code review

devpilot sync                                              # Sync OpenSpec changes to board/issues
devpilot sync --board "Board Name"                         # Override board
devpilot sync --source github                              # Override source

devpilot gmail summary                                     # Dry run: summarize all unread emails (won't mark as read)
devpilot gmail summary --channel daily-digest              # Send summary to a Slack channel (marks as read)
devpilot gmail summary --dm U0123ABCDE                     # Send summary as a DM (marks as read)
devpilot gmail summary --no-mark-read=false                # Explicitly mark emails as read without sending

devpilot skill add <name>                                  # Install a skill (prompts for project/user level)
devpilot skill add <name>@<ref>                            # Install at specific git ref
devpilot skill list                                        # List available skills with install status
devpilot skill list --installed                            # List only installed skills

devpilot review <pr-url>                                   # AI-powered code review, posts to PR as GitHub review
devpilot review <pr-url> --no-post                         # Review without posting to PR
devpilot review <pr-url> --model claude-sonnet-4-6-20250514  # Review with custom model
devpilot review <pr-url> --dry-run                         # Print assembled prompt without executing

devpilot commit                                            # Generate commit message from staged changes
devpilot readme                                            # Generate or improve README.md
```

### Skill Helper Scripts (Python 3)

```bash
python3 .claude/skills/skill-creator/scripts/init_skill.py      # Scaffold a new skill
python3 .claude/skills/skill-creator/scripts/package_skill.py    # Package a skill for distribution
python3 .claude/skills/skill-creator/scripts/quick_validate.py   # Validate skill structure
```

## Architecture

### CLI

Go CLI using Cobra for subcommand routing. Adding a new service: implement the `Service` interface in `internal/auth/`, register in `service.go`.

### Project Init (`devpilot init`)

Interactive wizard that detects project state and generates missing pieces:
- Detects: `.devpilot.yaml`, Trello credentials, skills, git repo
- Generates: board config, gitignore entries, skill installation
- Task source configuration (Trello/GitHub) can be skipped

### Task Runner (`devpilot run`)

Tasks move through a state machine. Trello uses lists (**Ready** → **In Progress** → **Done** / **Failed**); GitHub Issues use labels (`devpilot` → `in-progress` → closed or `failed`).

1. Polls "Ready" list for cards
2. Sorts cards by priority (P0 > P1 > P2 labels; default P2)
3. Moves card to "In Progress"
4. Creates branch `task/{cardID}-{slug}` from main
5. Executes plan via `claude -p` with `stream-json` output
6. Pushes branch, creates PR via `gh`
7. Optionally runs automated code review via a second `claude -p` invocation
8. Auto-merges PR (`gh pr merge --squash --auto`)
9. Moves card to "Done" (with PR link) or "Failed" (with error log path)

Logs per-card output to `~/.config/devpilot/logs/{card-id}.log`.

### OpenSpec Integration

When OpenSpec is installed and `openspec/changes/` exists:
- `devpilot sync` scans changes and creates/updates Trello cards or GitHub Issues
- Card title = change directory name (used as `opsx:apply` argument)
- Card description = full content of proposal.md + tasks.md
- Runner auto-detects OpenSpec and uses `/opsx:apply <change-name>` instead of raw plan text
- Supports resumability: interrupted tasks pick up from last unchecked task

### TUI Dashboard

When `devpilot run` launches in a TTY, it displays a real-time Bubble Tea dashboard:
- **Header**: Board name, runner phase, token stats
- **Status & Active**: Trello list states + current card info
- **Tools & Files**: Tool call history with durations + file access tracking
- **Claude Output**: Scrollable text output
- **Footer**: Completed task history + errors

Keyboard: `q`/`Ctrl-C` quit, `Tab` switch pane, `j/k/↑/↓` scroll, `g/G` top/bottom.

Falls back to plain text mode when not a TTY or `--no-tui` is set.

### Event System

The runner uses an event-driven architecture:
- **Runner** emits lifecycle events (`CardStarted`, `CardDone`, `ToolStart`, `TextOutput`, etc.)
- **EventBridge** parses `claude -p` stream-json output and translates to runner events
- **TUI** receives events via buffered channel (size 100) and updates the Bubble Tea model

### Skills

Skills are defined by a `SKILL.md` file (YAML frontmatter + markdown body) with optional `references/` and `scripts/` directories. They use progressive disclosure: frontmatter metadata is always in context, body loads on invocation, references load on demand.

**Directory distinction:**
- `skills/` at the project root — Catalog source code for distributable skills (prefixed `devpilot-<name>/`). This is what gets fetched by `devpilot skill add`.
- `.claude/skills/` — Where skills are installed in a project (so Claude Code discovers them). Also where project-local OpenSpec workflow skills live.

## Key Conventions

- CLI is written in Go with Cobra; tests via `go test ./...`
- Functional options pattern (`WithXxx()`) for testability in Executor and trello.Client
- Design docs come in pairs: `{date}-{feature}-design.md` + `{date}-{feature}-plan.md`
- Skill helper scripts use Python 3
- CI/CD: GitHub Actions for tests (`test.yml`) and releases (`release.yml`)
- When adding, removing, or modifying skills in `skills/`, update `skills/index.json` accordingly (name, description, and file list)
