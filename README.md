# Claude Code Developer Kit

A Go CLI tool and collection of [Claude Code](https://claude.ai/code) skills for automating development workflows. Write a plan, push it to Trello, and let an autonomous runner execute it — creating branches, PRs, and code reviews automatically.

## How It Works

```
Plan (markdown) → devkit push → Trello card → devkit run → claude -p → Branch + PR
```

1. **Write a plan** — A markdown file with a `# Title` and implementation steps
2. **Push to Trello** — `devkit push plan.md --board "Sprint Board"` creates a card in the "Ready" list
3. **Runner picks it up** — `devkit run --board "Sprint Board"` polls the board, executes each card's plan via `claude -p`
4. **Automatic output** — Branch created, code written with TDD, PR opened, optional automated code review

## Installation

Requires Go 1.25+ and [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed.

```bash
git clone https://github.com/siyuqian/developer-kit.git
cd developer-kit
make build
```

The binary is built to `bin/devkit`.

## Quick Start

### 1. Authenticate with Trello

```bash
devkit login trello
```

Follow the prompts to enter your [Trello API key and token](https://trello.com/power-ups/admin).

### 2. Push a Plan

```bash
devkit push docs/plans/my-feature-plan.md --board "Sprint Board"
```

### 3. Run the Task Runner

```bash
# Continuous mode — polls every 5 minutes
devkit run --board "Sprint Board"

# Test mode — one card, no execution
devkit run --board "Sprint Board" --once --dry-run
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `devkit login <service>` | Authenticate with a service (currently: `trello`) |
| `devkit logout <service>` | Remove stored credentials |
| `devkit status` | Show authentication status |
| `devkit push <file>` | Create a Trello card from a plan markdown file |
| `devkit run` | Autonomously process tasks from a Trello board |

### `devkit push` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--board` | *(required)* | Trello board name |
| `--list` | `Ready` | Target list name |

### `devkit run` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--board` | *(required)* | Trello board name |
| `--interval` | `300` | Poll interval in seconds |
| `--timeout` | `30` | Per-task execution timeout in minutes |
| `--review-timeout` | `10` | Code review timeout in minutes (0 to disable) |
| `--once` | `false` | Process one card and exit |
| `--dry-run` | `false` | Print actions without executing |

## Task Runner Workflow

The runner uses Trello lists as a state machine:

```
Ready → In Progress → Done
                    → Failed
```

For each card:
1. Validates the card has a description (the plan)
2. Moves card to "In Progress"
3. Creates a branch `task/{cardID}-{slug}` from main
4. Runs `claude -p` with the plan as the prompt
5. Pushes the branch and creates a PR via `gh`
6. Optionally runs automated code review via a second `claude -p` invocation
7. Moves card to "Done" (with PR link) or "Failed" (with error details)

Per-card logs are saved to `~/.config/devkit/logs/{card-id}.log`.

## Built-in Skills

The kit includes Claude Code skills in `.claude/skills/`:

| Skill | Description |
|-------|-------------|
| `skill-creator` | Guide and scripts for creating new Claude Code skills |
| `mcp-builder` | Guide for building MCP (Model Context Protocol) servers |
| `developerkit:pm` | Product manager — market research, competitor analysis, feature prioritization |
| `developerkit:trello` | Direct Trello board and card management from Claude Code |
| `developerkit:task-executor` | Autonomous plan execution (used internally by `devkit run`) |

## Project Structure

```
developer-kit/
├── cmd/devkit/          CLI entry point
├── internal/
│   ├── cli/             Cobra command definitions
│   ├── config/          Credential storage
│   ├── services/        Service interface + implementations
│   ├── trello/          Trello REST API client
│   └── runner/          Task runner (executor, git ops, reviewer)
├── .claude/skills/      Built-in Claude Code skills
├── docs/
│   ├── plans/           Design and implementation plans
│   └── rejected/        Rejected idea records (prevents re-recommendation)
├── Makefile             Build targets
└── CLAUDE.md            Project instructions for Claude Code
```

## Development

```bash
make build    # Build binary
make test     # Run tests
make clean    # Clean build artifacts
```

## License

MIT
