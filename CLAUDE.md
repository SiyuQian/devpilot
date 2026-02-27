# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Claude Code Developer Kit** — a collection of CLI tools, skills, and integrations for building Claude Code extensions.

## Repository Structure

- `cmd/devkit/` — CLI entry point
- `internal/cli/` — Cobra command definitions (login, logout, status)
- `internal/config/` — Credential storage (~/.config/devkit/credentials.json)
- `internal/services/` — Service interface + implementations (Trello)
- `.claude/skills/` — Built-in development skills
  - `skill-creator/` — Guide + scripts for creating new skills
  - `mcp-builder/` — Guide + scripts for building MCP servers
  - `pm/` — Product manager skill for market research
  - `trello/` — Trello board and card management skill
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

### Skill Helper Scripts (Python 3)

```bash
python3 .claude/skills/skill-creator/scripts/init_skill.py      # Scaffold a new skill
python3 .claude/skills/skill-creator/scripts/package_skill.py    # Package a skill for distribution
python3 .claude/skills/skill-creator/scripts/quick_validate.py   # Validate skill structure
```

## Architecture

### Devkit CLI

Go CLI tool using Cobra for subcommand routing:
- `cmd/devkit/` — Entry point (`main.go` calls `cli.Execute()`)
- `internal/cli/` — Cobra commands (root, login, logout, status)
- `internal/config/` — Credential storage (~/.config/devkit/credentials.json)
- `internal/services/` — Service interface + implementations (Trello)
- Adding a new service: implement the Service interface in a new file, register in registry.go

### Skills

Skills are defined by a `SKILL.md` file (metadata + instructions) with optional `references/` and `scripts/` directories. They use progressive disclosure: metadata is loaded first, full content on invocation.

## Key Conventions

- CLI is written in Go with Cobra for command routing
- Go tests via `go test ./...`; no additional test framework
- Skill helper scripts use Python 3
- No CI/CD pipeline configured
