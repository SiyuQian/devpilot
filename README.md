# DevPilot

**Autonomous development workflow automation for [Claude Code](https://claude.ai/code).** Write a plan in markdown, track it in Trello or GitHub Issues, and let DevPilot execute it — creating branches, writing code, opening PRs, running code review, and auto-merging.

## How It Works

Pick your task backend:

**GitHub Issues** (recommended — uses `gh` auth you already have):
```
Create Issue with devpilot label → devpilot run --source github → claude -p → Branch + PR
```

**Trello** (great if your team already uses it):
```
Push plan to Trello → devpilot run --board "My Board" → claude -p → Branch + PR
```

DevPilot polls your task source, prioritizes by labels (P0/P1/P2), and executes each task via `claude -p`. A real-time TUI dashboard shows tool calls, Claude's output, token usage, and progress. When done, it auto-merges the PR.

## Features

- **GitHub Issues & Trello support** — No vendor lock-in; pick what your team uses
- **Autonomous execution** — Tasks flow through Ready → In Progress → Done/Failed without human intervention
- **Priority scheduling** — P0/P1/P2 labels control order; GitHub Issues auto-sorted by creation time within priority
- **Real-time TUI dashboard** — Bubble Tea terminal UI with tool history, file changes, token stats, and scrollable output
- **Automated code review** — A second `claude -p` invocation validates the diff before merging
- **OpenSpec integration** — Sync spec-driven changes to Trello or GitHub Issues
- **Gmail AI digest** — `devpilot gmail summary` creates Claude-powered email summaries (dry run by default)
- **Slack integration** — Send summaries to channels or DMs
- **Project scaffolding** — `devpilot init` detects your stack and generates config, hooks, and skills

## Getting Started

### Prerequisites

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated
- [GitHub CLI (`gh`)](https://cli.github.com/) installed and authenticated
- Git repository initialized in your project
- *(Trello source only)* [Trello account with API credentials](https://trello.com/power-ups/admin)
- *(Optional)* Google OAuth for Gmail integration
- *(Optional)* Slack OAuth for Slack integration

### Installation

**From release:**
```bash
curl -sSL https://raw.githubusercontent.com/siyuqian/devpilot/main/install.sh | sh
```

**From source (Go 1.25+):**
```bash
git clone https://github.com/siyuqian/devpilot.git
cd devpilot
make build
sudo mv bin/devpilot /usr/local/bin/
```

Verify: `devpilot --version`

### Quick Start: GitHub Issues (no extra accounts)

```bash
# Initialize (select "github" for task source)
devpilot init

# Create an issue with the devpilot label
gh issue create --title "Add dark mode" --label devpilot

# Run the runner
devpilot run --source github
```

DevPilot auto-creates labels (`devpilot`, `in-progress`, `failed`, `P0-critical`, `P1-high`, `P2-normal`).

### Quick Start: Trello

```bash
# Initialize
devpilot init

# Authenticate
devpilot login trello

# Push a plan to Trello
devpilot push docs/plans/my-feature.md --board "Sprint Board"

# Run the runner
devpilot run --board "Sprint Board"
```

## CLI Reference

### Core Commands

| Command | Description |
|---------|-------------|
| `devpilot init` | Project setup wizard (detects stack, generates config) |
| `devpilot run` | Execute tasks from Trello or GitHub Issues |
| `devpilot push <file> --board "Board"` | Create Trello card from markdown plan |
| `devpilot sync` | Sync OpenSpec changes to task backend |

### Service Commands

| Command | Description |
|---------|-------------|
| `devpilot login <service>` | Authenticate (`trello`, `gmail`, `slack`) |
| `devpilot logout <service>` | Remove stored credentials |
| `devpilot status` | Show auth status for all services |

### Email Commands

| Command | Description |
|---------|-------------|
| `devpilot gmail summary` | AI digest of unread emails |
| `devpilot gmail list` | List emails with filters |
| `devpilot gmail read <id>` | Display full email |
| `devpilot gmail mark-read <id>...` | Mark as read |
| `devpilot gmail bulk-mark-read` | Bulk mark by query |

### Other Commands

| Command | Description |
|---------|-------------|
| `devpilot slack send --channel "#channel"` | Send Slack message |
| `devpilot commit` | Generate commit message from staged changes |
| `devpilot readme` | Generate or improve README.md |

### `devpilot run` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--source` | `trello` | `trello` or `github` |
| `--board` | *(required for Trello)* | Trello board name |
| `--interval` | `300` | Poll interval (seconds) |
| `--timeout` | `30` | Per-task timeout (minutes) |
| `--review-timeout` | `10` | Code review timeout (0 to disable) |
| `--once` | `false` | Run one task and exit |
| `--dry-run` | `false` | Print actions without executing |
| `--no-tui` | `false` | Disable TUI dashboard |

### `devpilot push` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--board` | *(required)* | Trello board name |
| `--list` | `Ready` | Target list name |

## Task Execution

Each task follows a state machine:

**GitHub Issues:**
```
open + devpilot → open + in-progress → closed (Done)
                                      → open + failed
```

**Trello:**
```
Ready → In Progress → Done
                    → Failed
```

For each task, DevPilot:
1. Marks as "In Progress"
2. Creates branch `task/{id}-{slug}` from main
3. Runs `claude -p` with the plan (streaming output)
4. Pushes branch and creates PR via `gh`
5. Optionally runs automated code review
6. Auto-merges PR (`gh pr merge --squash --auto`)
7. Marks as "Done" (with PR link) or "Failed" (with error)

Task logs: `~/.config/devpilot/logs/{task-id}.log`

**GitHub Issues Ordering:** Sorted by priority label (P0 > P1 > P2), then creation time (FIFO within priority). No configuration needed — fully automatic.

### TUI Dashboard

In TTY mode, displays real-time dashboard with:
- **Header:** Board, runner phase, token stats
- **Status:** Trello lists or GitHub issue counts
- **Active Task:** Current card details
- **Tool History:** Recent tool calls with durations
- **Files:** Reads and edits
- **Output:** Claude's text (scrollable)
- **Footer:** Completed tasks and errors

Keys: `q`/`Ctrl-C` quit, `Tab` switch pane, `j/k/↑/↓` scroll, `g/G` top/bottom

## Architecture

DevPilot turns markdown plans into shipped code via three systems:

1. **Task Source** — Pluggable interface (Trello API or GitHub Issues) with priority sorting
2. **Executor** — Wraps `claude -p --output-format stream-json` for real-time structured output
3. **Event Pipeline** — EventBridge translates stream-json events; TUI and logger consume them

```
Runner (orchestrator) → TaskSource + Executor → EventBridge → TUI/Logger
```

All three are decoupled via Go channels, so task execution doesn't block the dashboard.

## Development

```bash
make build      # Build to bin/devpilot
make test       # Run all tests
make lint       # Run linter (required before commit)
make lint-fix   # Auto-fix lint issues
make clean      # Remove build artifacts
```

Tests and lint must pass before committing. CI enforces this.

## Tech Stack

- **Language:** Go 1.25.6
- **CLI:** [Cobra](https://github.com/spf13/cobra)
- **TUI:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **AI:** [Claude Code](https://claude.ai/code) headless mode
- **Task Backends:** [Trello API](https://developer.atlassian.com/cloud/trello/) and [GitHub Issues](https://docs.github.com/en/issues)
- **Git/CI:** GitHub CLI (`gh`) for PRs and auto-merge

## License

MIT
