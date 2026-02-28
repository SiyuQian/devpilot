# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Claude Code Developer Kit** — a Go CLI tool and collection of skills for automating development workflows with Claude Code. The core workflow: write a plan, push it to Trello, and let an autonomous runner execute it via `claude -p`, creating branches and PRs automatically.

## Repository Structure

Standard Go project layout: `cmd/devkit/` for the CLI entry point, `internal/` for all packages.

**Package organization rules:**
- Each `internal/` package is a self-contained domain (e.g. `auth`, `trello`, `taskrunner`)
- Each domain package owns its own Cobra commands in `commands.go` — there is no central `cli/` routing layer
- External service clients (API, HTTP) live in the same package as their domain logic
- Shared project-level config lives in `internal/project/`

**Other top-level directories:**
- `.claude/skills/` — Built-in Claude Code skills (each skill is a dir with `SKILL.md`)
- `docs/plans/` — Design and implementation plan documents
- `docs/rejected/` — Rejected/deferred idea records (read by PM skill to avoid re-recommending)

## Build & Development Commands

```bash
make build                         # Build binary to bin/devkit
make test                          # Run all tests (go test ./...)
make run ARGS="--help"             # Build and run with arguments
make clean                         # Remove bin/
```

### CLI Commands

```bash
devkit login trello                # Authenticate with Trello (API key + token)
devkit logout trello               # Remove stored credentials
devkit status                      # Show authentication status for all services

devkit init                        # Interactive project setup wizard
devkit init -y                     # Accept all defaults

devkit push <plan.md> --board "Board Name"              # Create Trello card from plan file
devkit push <plan.md> --board "Board Name" --list "Ready"  # Specify target list (default: Ready)

devkit run --board "Board Name"                          # Start autonomous task runner (TUI mode)
devkit run --board "Board Name" --no-tui                 # Plain text output (no dashboard)
devkit run --board "Board Name" --once --dry-run         # Test with one card, no execution
devkit run --board "Board Name" --interval 60            # Poll every 60s (default: 300)
devkit run --board "Board Name" --timeout 45             # 45min per-task timeout (default: 30)
devkit run --board "Board Name" --review-timeout 0       # Disable auto code review
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

### Project Init (`devkit init`)

Interactive wizard that detects project state and generates missing pieces:
- Detects: `CLAUDE.md`, `.devkit.json`, Trello credentials, git hooks, skills, git repo
- Generates: `CLAUDE.md` template, board config, pre-push hook, skill scaffolding
- Auto-detects project type (Go/Node/Python) for build/test commands

### Task Runner (`devkit run`)

Cards move through Trello as a state machine: **Ready** -> **In Progress** -> **Done** / **Failed**.

1. Polls "Ready" list for cards
2. Sorts cards by priority (P0 > P1 > P2 labels; default P2)
3. Moves card to "In Progress"
4. Creates branch `task/{cardID}-{slug}` from main
5. Executes plan via `claude -p` with `stream-json` output
6. Pushes branch, creates PR via `gh`
7. Optionally runs automated code review via a second `claude -p` invocation
8. Auto-merges PR (`gh pr merge --squash --auto`)
9. Moves card to "Done" (with PR link) or "Failed" (with error log path)

Logs per-card output to `~/.config/devkit/logs/{card-id}.log`.

### TUI Dashboard

When `devkit run` launches in a TTY, it displays a real-time Bubble Tea dashboard:
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

## Key Conventions

- CLI is written in Go with Cobra; tests via `go test ./...`
- Functional options pattern (`WithXxx()`) for testability in Executor and trello.Client
- Design docs come in pairs: `{date}-{feature}-design.md` + `{date}-{feature}-plan.md`
- Skill helper scripts use Python 3
- No CI/CD pipeline configured
