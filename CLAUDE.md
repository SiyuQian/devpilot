# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Claude Code Developer Kit** — a Go CLI tool and collection of skills for automating development workflows with Claude Code. The core workflow: write a plan, push it to Trello, and let an autonomous runner execute it via `claude -p`, creating branches and PRs automatically.

## Repository Structure

- `cmd/devkit/` — CLI entry point (`main.go` calls `cli.Execute()`)
- `internal/cli/` — Cobra command definitions (login, logout, status, run, push)
- `internal/config/` — Credential storage (`~/.config/devkit/credentials.json`)
- `internal/services/` — Service interface + implementations (Trello)
- `internal/trello/` — Trello REST API client (boards, lists, cards)
- `internal/runner/` — Task runner: Executor (`claude -p` wrapper), GitOps (branch/PR), Reviewer (automated code review), Runner (poll loop)
- `.claude/skills/` — Built-in development skills
  - `skill-creator/` — Guide + scripts for creating new skills
  - `mcp-builder/` — Guide + scripts for building MCP servers
  - `pm/` — Product manager skill for market research and feature discovery
  - `trello/` — Trello board and card management skill
  - `task-executor/` — Autonomous task plan execution (used by `devkit run`)
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

devkit push <plan.md> --board "Board Name"              # Create Trello card from plan file
devkit push <plan.md> --board "Board Name" --list "Ready"  # Specify target list (default: Ready)

devkit run --board "Board Name"                          # Start autonomous task runner
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

Go CLI using Cobra for subcommand routing. Adding a new service: implement the `Service` interface in `internal/services/`, register in `registry.go`.

### Task Runner (`devkit run`)

Cards move through Trello as a state machine: **Ready** -> **In Progress** -> **Done** / **Failed**.

1. Polls "Ready" list for cards
2. Moves card to "In Progress"
3. Creates branch `task/{cardID}-{slug}` from main
4. Executes plan via `claude -p` with streaming output
5. Pushes branch, creates PR via `gh`
6. Optionally runs automated code review via a second `claude -p` invocation
7. Moves card to "Done" (with PR link) or "Failed" (with error log path)

Logs per-card output to `~/.config/devkit/logs/{card-id}.log`.

### Skills

Skills are defined by a `SKILL.md` file (YAML frontmatter + markdown body) with optional `references/` and `scripts/` directories. They use progressive disclosure: frontmatter metadata is always in context, body loads on invocation, references load on demand.

## Key Conventions

- CLI is written in Go with Cobra; tests via `go test ./...`
- Functional options pattern (`WithXxx()`) for testability in Executor and trello.Client
- Design docs come in pairs: `{date}-{feature}-design.md` + `{date}-{feature}-plan.md`
- Skill helper scripts use Python 3
- No CI/CD pipeline configured
