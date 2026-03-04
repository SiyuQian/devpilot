# DevPilot

**Autonomous development workflow automation for [Claude Code](https://claude.ai/code).** Write a plan in markdown, push it to Trello, and let DevPilot execute it — creating branches, writing code, opening PRs, running code review, and auto-merging.

## How It Works

```
Plan (markdown) → devpilot push → Trello card → devpilot run → claude -p → Branch + PR
```

1. **Write a plan** — A markdown file with a `# Title` and implementation steps
2. **Push to Trello** — `devpilot push plan.md --board "My Board"` creates a card in the "Ready" list
3. **Runner picks it up** — `devpilot run` polls the board, prioritizes by P0/P1/P2 labels, and executes each plan via `claude -p`
4. **Watch it work** — A real-time TUI dashboard shows tool calls, Claude output, token stats, and progress
5. **Ship it** — Branch created, code written, PR opened, AI code review, auto-merge

## Features

- **Autonomous task execution** — Cards flow through Ready → In Progress → Done/Failed without human intervention
- **Priority scheduling** — P0/P1/P2 labels control execution order
- **Real-time TUI dashboard** — Bubble Tea terminal UI with tool call history, file tracking, token stats, and scrollable output
- **Automated code review** — A second `claude -p` invocation reviews the diff against the original plan before merging
- **Built-in Claude Code skills** — PM research, Trello management, task refinement, and more
- **Project scaffolding** — `devpilot init` detects your stack and generates config, hooks, and skills

## Getting Started

### Prerequisites

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated
- [GitHub CLI (`gh`)](https://cli.github.com/) installed and authenticated
- A [Trello](https://trello.com/) account with an [API key and token](https://trello.com/power-ups/admin)
- Git repository initialized in your project

### Installation

**From release (recommended):**

```bash
curl -sSL https://raw.githubusercontent.com/siyuqian/devpilot/main/install.sh | sh
```

Optionally specify a version or directory:

```bash
curl -sSL https://raw.githubusercontent.com/siyuqian/devpilot/main/install.sh | sh -s -- --version v0.1.0 --dir ~/.local/bin
```

**From source** (requires Go 1.25+):

```bash
git clone https://github.com/siyuqian/devpilot.git
cd devpilot
make build
sudo mv bin/devpilot /usr/local/bin/
```

Verify: `devpilot --version`

### Quick Start

```bash
# 1. Initialize your project
cd your-project
devpilot init          # interactive wizard; use -y for defaults

# 2. Authenticate with Trello
devpilot login trello

# 3. Push a plan
devpilot push docs/plans/my-feature-plan.md --board "Sprint Board"

# 4. Run the task runner
devpilot run --board "Sprint Board"
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `devpilot init` | Interactive project setup wizard |
| `devpilot login <service>` | Authenticate with a service (currently: `trello`) |
| `devpilot logout <service>` | Remove stored credentials |
| `devpilot status` | Show authentication status |
| `devpilot push <file>` | Create a Trello card from a plan markdown file |
| `devpilot run` | Autonomously process tasks from a Trello board |
| `devpilot commit` | Generate a commit message from staged changes |
| `devpilot readme` | Generate or improve README.md |

### `devpilot push` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--board` | *(required)* | Trello board name |
| `--list` | `Ready` | Target list name |

### `devpilot run` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--board` | *(required)* | Trello board name |
| `--interval` | `300` | Poll interval in seconds |
| `--timeout` | `30` | Per-task timeout in minutes |
| `--review-timeout` | `10` | Code review timeout in minutes (0 to disable) |
| `--once` | `false` | Process one card and exit |
| `--dry-run` | `false` | Print actions without executing |
| `--no-tui` | `false` | Disable TUI dashboard |

## Task Runner Workflow

The runner uses Trello lists as a state machine:

```
Ready → In Progress → Done
                    → Failed
```

For each card:
1. Polls "Ready" list and sorts by priority (P0 > P1 > P2; default P2)
2. Validates the card has a description (the plan)
3. Moves card to "In Progress"
4. Creates branch `task/{cardID}-{slug}` from main
5. Runs `claude -p` with the plan, streaming output via `stream-json`
6. Pushes branch and creates a PR via `gh`
7. Optionally runs automated code review via a second `claude -p` invocation
8. Auto-merges PR (`gh pr merge --squash --auto`)
9. Moves card to "Done" (with PR link) or "Failed" (with error details)

Per-card logs: `~/.config/devpilot/logs/{card-id}.log`

### TUI Dashboard

In TTY mode, the runner displays a real-time terminal dashboard:

```
┌─────────────────────────────────────────────┐
│ Header: Board / Phase / Token Stats         │
├──────────────────────┬──────────────────────┤
│ Trello Lists Status  │ Active Card Info     │
├──────────────────────┼──────────────────────┤
│ Tool Call History     │ Files Read/Edited    │
├──────────────────────┴──────────────────────┤
│ Claude Text Output (scrollable)             │
├─────────────────────────────────────────────┤
│ Footer: Completed Tasks / Errors            │
└─────────────────────────────────────────────┘
```

Keys: `q`/`Ctrl-C` quit, `Tab` switch pane, `j/k/↑/↓` scroll, `g/G` top/bottom.

## Architecture

DevPilot turns **markdown plans into shipped code** by orchestrating three systems: a task queue (Trello), an AI coding agent (`claude -p`), and standard Git/GitHub workflows.

### Event-Driven Pipeline

```
┌──────────────────────────────────────────────────────────┐
│  Runner (orchestrator)                                   │
│  Polls Trello → manages card lifecycle → emits events    │
├──────────────────────────────────────────────────────────┤
│  EventBridge (translator)                                │
│  Parses claude -p stream-json → translates to events     │
├──────────────────────────────────────────────────────────┤
│  TUI / Logger (consumers)                                │
│  Receives events via channel → renders dashboard / logs  │
└──────────────────────────────────────────────────────────┘
```

- **Runner** owns the card state machine and drives the full lifecycle: branch, execute, push, PR, review, merge
- **Executor** wraps `claude -p --output-format stream-json` for real-time structured output
- **EventBridge** translates stream-json events into typed runner events (`ToolStart`, `TextOutput`, `TokenUsage`, etc.)
- **TUI** and **Logger** subscribe via buffered Go channels, decoupling execution from presentation

## Built-in Skills

DevPilot ships with Claude Code skills in `.claude/skills/`:

| Skill | Description |
|-------|-------------|
| `skill-creator` | Guide and scripts for creating new Claude Code skills |
| `devpilot:pm` | Product manager — market research, competitor analysis, feature prioritization |
| `devpilot:trello` | Direct Trello board and card management from Claude Code |
| `devpilot:task-executor` | Autonomous plan execution (used internally by `devpilot run`) |
| `devpilot:task-refiner` | Improve and expand Trello card task plans |

## Project Structure

```
devpilot/
├── cmd/devpilot/            CLI entry point
├── internal/
│   ├── auth/                Authentication & credential management
│   ├── generate/            AI-powered commit & readme generation
│   ├── initcmd/             Project initialization wizard
│   ├── project/             Project config (.devpilot.json)
│   ├── trello/              Trello API client & push command
│   └── taskrunner/          Runner, executor, TUI dashboard
├── .claude/skills/          Built-in Claude Code skills
├── docs/plans/              Design & implementation plans
├── Makefile                 Build targets
└── CLAUDE.md                Project instructions for Claude Code
```

## Tech Stack

- **Language:** Go 1.25
- **CLI framework:** [Cobra](https://github.com/spf13/cobra)
- **TUI:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **AI engine:** [Claude Code](https://claude.ai/code) (`claude -p` headless mode)
- **Task queue:** [Trello API](https://developer.atlassian.com/cloud/trello/)
- **Git/CI:** GitHub CLI (`gh`) for PRs and auto-merge

## Development

```bash
make build    # Build binary to bin/devpilot
make test     # Run all tests
make clean    # Remove build artifacts
```

## License

MIT
