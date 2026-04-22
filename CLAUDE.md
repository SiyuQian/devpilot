# CLAUDE.md

Repo-wide context loaded on every prompt. Keep tight. Detail lives in the linked docs.

## Project Overview

**DevPilot** is a Go CLI tool and collection of skills for automating development workflows with Claude Code. The core workflow: write a plan, queue it in Trello or GitHub Issues, and let an autonomous runner execute it via `claude -p`, creating branches and PRs automatically.

## Repository Structure

Standard Go layout: `cmd/devpilot/` entry point, `internal/` packages.

**Package organization rules:**
- Each `internal/` package is a self-contained domain (`auth`, `trello`, `taskrunner`, …)
- Each domain owns its own Cobra commands in `commands.go` — no central `cli/` router
- External service clients live in the same package as their domain logic
- Cross-cutting project config lives in `internal/project/`

**Top-level directories:**
- `skills/` — Distributable skill catalog (`devpilot-<name>/` dirs; register each in `skills/index.json`)
- `.claude/skills/` — Installed skills for this project + local OpenSpec workflow skills
- `.github/workflows/` — CI/CD (test + release)
- `docs/` — Design docs (`plans/`), deferred ideas (`rejected/`), architecture + CLI reference

## Golden Principles

Opinionated taste calls (what a linter can't enforce) live in `GOLDEN_PRINCIPLES.md`. Read it before changes to architecture, public APIs, or shared utilities.

## Build & Test

```bash
make build      # → bin/devpilot
make test       # go test ./...
make lint       # golangci-lint — must pass before commit
```

Full CLI surface and dev commands: `docs/cli-reference.md`.

## Architecture

Subsystem detail (runner, TUI, event system, OpenSpec integration, skills): `docs/architecture.md`.

## Agent Harness

How guides + sensors are wired in this repo, and where the current gaps are: `docs/harness.md`. Read this before adding new linters, hooks, skills, or MCP servers — it will tell you where the new control belongs and whether it's actually needed.

## Key Conventions

- Cobra commands live with their domain in `commands.go`
- Functional options (`WithXxx()`) for constructors with any optional parameters
- Wrap errors with `%w` at layer boundaries: `fmt.Errorf("doing X: %w", err)`
- Design docs come in pairs: `docs/plans/{YYYY-MM-DD}-{feature}-{design,plan}.md`
- Table-driven tests with named subtests; no mocks for our own packages
- Skill helper scripts use Python 3
- When adding/removing/modifying skills in `skills/`, update `skills/index.json`
