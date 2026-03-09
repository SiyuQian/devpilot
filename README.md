# DevPilot

[![Test](https://github.com/siyuqian/devpilot/actions/workflows/test.yml/badge.svg)](https://github.com/siyuqian/devpilot/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/siyuqian/devpilot/branch/main/graph/badge.svg)](https://codecov.io/gh/siyuqian/devpilot)
[![GitHub Downloads](https://img.shields.io/github/downloads/siyuqian/devpilot/total)](https://github.com/siyuqian/devpilot/releases)

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
- **OpenSpec integration** — Sync spec-driven changes to Trello or GitHub Issues with `devpilot sync`
- **Gmail AI digest** — `devpilot gmail summary` summarizes today's unread emails via Claude and optionally sends to Slack
- **Slack integration** — Send messages to channels or DMs, used as an output target for Gmail summaries
- **Built-in Claude Code skills** — PM research, Trello management, task refinement, Confluence review, and more
- **Project scaffolding** — `devpilot init` detects your stack and generates config, hooks, and skills

## Getting Started

### Prerequisites

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated
- [GitHub CLI (`gh`)](https://cli.github.com/) installed and authenticated
- A [Trello](https://trello.com/) account with an [API key and token](https://trello.com/power-ups/admin)
- Git repository initialized in your project
- *(Optional)* Google OAuth credentials for Gmail integration
- *(Optional)* Slack OAuth credentials for Slack integration

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
| `devpilot login <service>` | Authenticate with a service (`trello`, `gmail`, `slack`) |
| `devpilot logout <service>` | Remove stored credentials |
| `devpilot status` | Show authentication status |
| `devpilot push <file>` | Create a Trello card from a plan markdown file |
| `devpilot run` | Autonomously process tasks from a Trello board |
| `devpilot sync` | Sync OpenSpec changes to Trello board or GitHub Issues |
| `devpilot gmail list` | List emails with search filters |
| `devpilot gmail read <id>` | Display full email content |
| `devpilot gmail mark-read <id>...` | Mark emails as read |
| `devpilot gmail bulk-mark-read` | Bulk mark emails as read by query |
| `devpilot gmail summary` | AI-powered email digest via Claude |
| `devpilot slack send` | Send message to a Slack channel or DM |
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

### `devpilot sync` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--board` | *(from config)* | Override Trello board name |
| `--source` | `trello` | Task source (`trello` or `github`) |

### `devpilot gmail list` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--unread` | `false` | Show only unread emails |
| `--after` | | Filter emails after date (YYYY-MM-DD) |
| `--limit` | `20` | Maximum number of emails to list |

### `devpilot gmail bulk-mark-read` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--query` | *(required)* | Gmail search query (e.g. `category:promotions`) |

### `devpilot gmail summary` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--channel` | | Send summary to a Slack channel |
| `--dm` | | Send summary as DM to a Slack user ID |
| `--no-mark-read` | `false` | Preview mode (don't mark emails as read) |

### `devpilot slack send` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--channel` | *(required)* | Channel name or user ID for DM |
| `--message` | | Message text (reads from stdin if omitted) |

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
| `devpilot:pm` | Product manager — market research, competitor analysis, feature prioritization |
| `devpilot:trello` | Direct Trello board and card management from Claude Code |
| `devpilot:task-executor` | Autonomous plan execution (used internally by `devpilot run`) |
| `devpilot:task-refiner` | Improve and expand Trello card task plans |
| `devpilot:confluence-reviewer` | Review Atlassian Confluence pages and leave comments |

## Project Structure

```
devpilot/
├── cmd/devpilot/            CLI entry point
├── internal/
│   ├── auth/                Authentication & credential management (OAuth 2.0)
│   ├── generate/            AI-powered commit & readme generation
│   ├── gmail/               Gmail API client, email listing & AI summary
│   ├── initcmd/             Project initialization wizard
│   ├── openspec/            OpenSpec integration & sync command
│   ├── project/             Project config (.devpilot.json)
│   ├── slack/               Slack API client & message sending
│   ├── trello/              Trello API client & push command
│   └── taskrunner/          Runner, executor, TUI dashboard
├── .claude/skills/          Built-in Claude Code skills
├── .github/workflows/       CI/CD (test + release pipelines)
├── docs/plans/              Design & implementation plans
├── Makefile                 Build targets
└── CLAUDE.md                Project instructions for Claude Code
```

## Tech Stack

- **Language:** Go 1.25.6
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
