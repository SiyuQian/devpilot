# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Claude Code Developer Kit** — a collection of CLI tools, skills, and integrations for building Claude Code extensions.

## Repository Structure

- `cmd/devkit/` — CLI entry point (wires domain packages)
- `internal/auth/` — Authentication, credentials, service registry, CLI commands (login/logout/status)
- `internal/trello/` — Trello REST API client, types, CLI commands (push)
- `internal/taskrunner/` — Task executor, git ops, code reviewer, runner loop, CLI commands (run)
- `.claude/skills/` — Built-in development skills
  - `skill-creator/` — Guide + scripts for creating new skills
  - `mcp-builder/` — Guide + scripts for building MCP servers
  - `pm/` — Product manager skill for market research
  - `trello/` — Trello board and card management skill
  - `task-executor/` — Autonomous task plan execution (used by `devkit run`)
- `docs/plans/` — Design and planning documents

## Build & Development Commands

### Devkit CLI

```bash
make build                         # Build binary
make test                          # Run all tests
make run ARGS="--help"             # Run with arguments
make run ARGS="login trello"       # Login to Trello
make run ARGS="status"             # Check auth status
```

### Task Runner

```bash
make run ARGS="run --board 'Sprint Board'"                  # Run task runner
make run ARGS="run --board 'Sprint Board' --once --dry-run" # Test with one card, no execution
```

### Skill Helper Scripts (Python 3)

```bash
python3 .claude/skills/skill-creator/scripts/init_skill.py      # Scaffold a new skill
python3 .claude/skills/skill-creator/scripts/package_skill.py    # Package a skill for distribution
python3 .claude/skills/skill-creator/scripts/quick_validate.py   # Validate skill structure
```

## Architecture

### Devkit CLI

Go CLI tool using Cobra, organized by domain:
- `cmd/devkit/` — Entry point, creates root command, registers domain commands
- `internal/auth/` — Credentials storage (~/.config/devkit/credentials.json), Service interface + registry, Trello auth (login/logout/verify), CLI commands: login, logout, status
- `internal/trello/` — Trello REST API client (boards, lists, cards), CLI commands: push
- `internal/taskrunner/` — Executor (claude -p wrapper), GitOps (branch/PR management), Reviewer (automated code review), Runner (poll loop), CLI commands: run
- Adding a new service: implement the Service interface in `internal/auth/`, register in `service.go` init()

### Skills

Skills are defined by a `SKILL.md` file (metadata + instructions) with optional `references/` and `scripts/` directories. They use progressive disclosure: metadata is loaded first, full content on invocation.

## Key Conventions

- CLI is written in Go with Cobra for command routing
- Go tests via `go test ./...`; no additional test framework
- Skill helper scripts use Python 3
- No CI/CD pipeline configured
