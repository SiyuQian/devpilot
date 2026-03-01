# DevPilot

A Go CLI tool and collection of [Claude Code](https://claude.ai/code) skills for automating development workflows. Write a plan, push it to Trello, and let an autonomous runner execute it — creating branches, PRs, and code reviews automatically.

## How It Works

```
Plan (markdown) → devpilot push → Trello card → devpilot run → claude -p → Branch + PR
```

1. **Write a plan** — A markdown file with a `# Title` and implementation steps
2. **Push to Trello** — `devpilot push plan.md --board "Sprint Board"` creates a card in the "Ready" list
3. **Runner picks it up** — `devpilot run --board "Sprint Board"` polls the board, prioritizes by P0/P1/P2 labels, and executes each card's plan via `claude -p`
4. **Real-time dashboard** — A TUI dashboard shows tool calls, Claude output, token stats, and task progress in real time
5. **Automatic output** — Branch created, code written with TDD, PR opened, auto code review, auto-merge

## Getting Started

### Prerequisites

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated
- [GitHub CLI (`gh`)](https://cli.github.com/) installed and authenticated (for PR creation and auto-merge)
- A [Trello](https://trello.com/) account with an [API key and token](https://trello.com/power-ups/admin)
- Git repository initialized in your project

### Installation

**Option A: Install from release (recommended)**

```bash
curl -sSL https://raw.githubusercontent.com/siyuqian/devpilot/main/install.sh | sh
```

You can specify a version or install directory:

```bash
curl -sSL https://raw.githubusercontent.com/siyuqian/devpilot/main/install.sh | sh -s -- --version v0.1.0 --dir ~/.local/bin
```

**Option B: Build from source**

Requires Go 1.25+.

```bash
git clone https://github.com/siyuqian/devpilot.git
cd devpilot
make build
# Binary is at bin/devpilot — add it to your PATH or move it:
sudo mv bin/devpilot /usr/local/bin/
```

Verify the installation:

```bash
devpilot --version
```

### Setup

**1. Initialize your project**

```bash
cd your-project
devpilot init
```

The interactive wizard detects your project setup (git, CLAUDE.md, Trello credentials, skills) and generates any missing pieces. Use `-y` to accept all defaults.

**2. Authenticate with Trello**

```bash
devpilot login trello
```

Follow the prompts to enter your [Trello API key and token](https://trello.com/power-ups/admin). You can verify with `devpilot status`.

**3. Push a plan**

Write a markdown file with a `# Title` and implementation steps, then push it:

```bash
devpilot push docs/plans/my-feature-plan.md --board "Sprint Board"
```

**4. Run the task runner**

```bash
# Continuous mode — polls every 5 minutes, shows TUI dashboard
devpilot run --board "Sprint Board"

# Plain text mode (no TUI)
devpilot run --board "Sprint Board" --no-tui

# Test mode — one card, no execution
devpilot run --board "Sprint Board" --once --dry-run
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

### `devpilot init` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-y, --yes` | `false` | Accept all defaults without prompting |

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

Per-card logs are saved to `~/.config/devpilot/logs/{card-id}.log`.

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

## Architecture

### Core Concept

DevPilot turns **markdown plans into shipped code** by orchestrating three systems: a task queue (Trello), an AI coding agent (`claude -p`), and standard Git/GitHub workflows. The human writes *what* to build; the machine handles *how*.

### Event-Driven Pipeline

The runner is built on an event-driven architecture with three layers:

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

- **Runner** owns the card state machine (Ready → In Progress → Done/Failed) and drives the full lifecycle: git branch, execute, push, PR, review, merge.
- **Executor** wraps `claude -p` with `--output-format stream-json`, which produces a stream of structured JSON events (tool calls, text output, token usage, etc.) instead of plain text.
- **EventBridge** parses this stream in real-time and translates each JSON event into typed runner events (`ToolStart`, `ToolEnd`, `TextOutput`, `TokenUsage`, etc.).
- **TUI** and **Logger** subscribe to these events via a buffered Go channel, decoupling the execution pipeline from presentation.

### How `claude -p` Is Used

The key integration point is Claude Code's headless mode:

```bash
claude -p "your prompt here" --output-format stream-json
```

This runs Claude Code non-interactively with a prompt (the plan from the Trello card). The `stream-json` format emits one JSON object per line as Claude works, allowing the runner to track progress, tool usage, and token consumption in real-time without waiting for completion.

### Task Prioritization

Cards are sorted before execution using Trello labels:
- **P0** (critical) → **P1** (high) → **P2** (normal, default)
- Cards without a priority label default to P2
- Within the same priority, cards are processed in list order

### Automated Code Review

After the PR is created, the runner optionally spawns a *second* `claude -p` invocation that reviews the diff against the original plan. This acts as an AI code reviewer, posting feedback as PR comments before auto-merging.

### Skills System

Skills extend Claude Code's capabilities through structured markdown files:

```
.claude/skills/my-skill/
├── SKILL.md          # YAML frontmatter (metadata) + markdown body (instructions)
├── references/       # Additional context loaded on demand
└── scripts/          # Helper scripts the skill can invoke
```

Skills use **progressive disclosure**: frontmatter metadata is always visible to Claude (for skill discovery), the body loads only when the skill is invoked, and references load only when explicitly requested. This keeps context usage efficient.

## Built-in Skills

The kit includes Claude Code skills in `.claude/skills/`:

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
│   ├── auth/              Authentication, credentials, service registry
│   ├── initcmd/           Project initialization wizard (devpilot init)
│   ├── project/           Project config (.devpilot.json)
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
