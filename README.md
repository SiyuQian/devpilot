# Claude Code Developer Kit

A Go CLI tool and collection of [Claude Code](https://claude.ai/code) skills for automating development workflows. Write a plan, push it to Trello, and let an autonomous runner execute it — creating branches, PRs, and code reviews automatically.

## How It Works

```
Plan (markdown) → devkit push → Trello card → devkit run → claude -p → Branch + PR
```

1. **Write a plan** — A markdown file with a `# Title` and implementation steps
2. **Push to Trello** — `devkit push plan.md --board "Sprint Board"` creates a card in the "Ready" list
3. **Runner picks it up** — `devkit run --board "Sprint Board"` polls the board, prioritizes by P0/P1/P2 labels, and executes each card's plan via `claude -p`
4. **Real-time dashboard** — A TUI dashboard shows tool calls, Claude output, token stats, and task progress in real time
5. **Automatic output** — Branch created, code written with TDD, PR opened, auto code review, auto-merge

## Installation

Requires Go 1.25+ and [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed.

```bash
git clone https://github.com/siyuqian/developer-kit.git
cd developer-kit
make build
```

The binary is built to `bin/devkit`.

## Quick Start

### 1. Initialize Your Project

```bash
devkit init
```

The interactive wizard detects your project setup (git, CLAUDE.md, Trello credentials, skills) and generates any missing pieces. Use `-y` to accept all defaults.

### 2. Authenticate with Trello

```bash
devkit login trello
```

Follow the prompts to enter your [Trello API key and token](https://trello.com/power-ups/admin).

### 3. Push a Plan

```bash
devkit push docs/plans/my-feature-plan.md --board "Sprint Board"
```

### 4. Run the Task Runner

```bash
# Continuous mode — polls every 5 minutes, shows TUI dashboard
devkit run --board "Sprint Board"

# Plain text mode (no TUI)
devkit run --board "Sprint Board" --no-tui

# Test mode — one card, no execution
devkit run --board "Sprint Board" --once --dry-run
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `devkit init` | Interactive project setup wizard |
| `devkit login <service>` | Authenticate with a service (currently: `trello`) |
| `devkit logout <service>` | Remove stored credentials |
| `devkit status` | Show authentication status |
| `devkit push <file>` | Create a Trello card from a plan markdown file |
| `devkit run` | Autonomously process tasks from a Trello board |

### `devkit init` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-y, --yes` | `false` | Accept all defaults without prompting |

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
| `--no-tui` | `false` | Disable TUI dashboard, use plain text output |

## Task Runner Workflow

The runner uses Trello lists as a state machine:

```
Ready → In Progress → Done
                    → Failed
```

For each card:
1. Polls "Ready" list and sorts by priority (P0 > P1 > P2 labels; default P2)
2. Validates the card has a description (the plan)
3. Moves card to "In Progress"
4. Creates a branch `task/{cardID}-{slug}` from main
5. Runs `claude -p` with the plan as prompt, streaming output via `stream-json`
6. Pushes the branch and creates a PR via `gh`
7. Optionally runs automated code review via a second `claude -p` invocation
8. Auto-merges PR (`gh pr merge --squash --auto`)
9. Moves card to "Done" (with PR link) or "Failed" (with error details)

Per-card logs are saved to `~/.config/devkit/logs/{card-id}.log`.

### TUI Dashboard

In TTY mode, the runner displays a real-time terminal dashboard (Bubble Tea):

```
┌─────────────────────────────────────────────┐
│ Header: Board / Phase / Token Stats         │
├──────────────────────┬──────────────────────┤
│ Trello Lists Status  │ Active Card Info     │
├──────────────────────┼──────────────────────┤
│ Tool Call History    │ Files Read/Edited     │
├──────────────────────┴──────────────────────┤
│ Claude Text Output (scrollable)             │
├─────────────────────────────────────────────┤
│ Footer: Completed Tasks / Errors            │
└─────────────────────────────────────────────┘
```

Keyboard shortcuts: `q`/`Ctrl-C` quit, `Tab` switch pane, `j/k/↑/↓` scroll, `g/G` top/bottom.

## Built-in Skills

The kit includes Claude Code skills in `.claude/skills/`:

| Skill | Description |
|-------|-------------|
| `skill-creator` | Guide and scripts for creating new Claude Code skills |
| `mcp-builder` | Guide for building MCP (Model Context Protocol) servers |
| `developerkit:pm` | Product manager — market research, competitor analysis, feature prioritization |
| `developerkit:trello` | Direct Trello board and card management from Claude Code |
| `developerkit:task-executor` | Autonomous plan execution (used internally by `devkit run`) |
| `developerkit:task-refiner` | Improve and expand Trello card task plans |

## Project Structure

```
developer-kit/
├── cmd/devkit/            CLI entry point
├── internal/
│   ├── auth/              Authentication, credentials, service registry
│   ├── initcmd/           Project initialization wizard (devkit init)
│   ├── project/           Project config (.devkit.json)
│   ├── trello/            Trello API client + push command
│   └── taskrunner/        Task runner, executor, TUI dashboard
│       ├── runner.go        Poll loop + card processing
│       ├── executor.go      claude -p wrapper (stream-json)
│       ├── reviewer.go      Automated code review
│       ├── git.go           Branch, push, PR, merge
│       ├── priority.go      P0/P1/P2 card sorting
│       ├── eventbridge.go   Claude events → runner events
│       ├── tui.go           Bubble Tea model
│       └── tui_view.go      Dashboard rendering
├── .claude/skills/        Built-in Claude Code skills
├── docs/
│   ├── plans/             Design and implementation plans
│   └── rejected/          Rejected idea records (prevents re-recommendation)
├── Makefile               Build targets
└── CLAUDE.md              Project instructions for Claude Code
```

## Development

```bash
make build    # Build binary
make test     # Run tests
make clean    # Clean build artifacts
```

## License

MIT
