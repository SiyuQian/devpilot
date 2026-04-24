# DevPilot

[![Test](https://github.com/SiyuQian/devpilot/actions/workflows/test.yml/badge.svg)](https://github.com/SiyuQian/devpilot/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/SiyuQian/devpilot/branch/main/graph/badge.svg)](https://codecov.io/gh/SiyuQian/devpilot)
[![GitHub Downloads](https://img.shields.io/github/downloads/SiyuQian/devpilot/total)](https://github.com/SiyuQian/devpilot/releases)

**A skill catalog for [Claude Code](https://claude.ai/code), plus a small set of Go-native helpers for Gmail, Slack, and Trello.** Install curated skills into any project with one command; use the CLI when a typed OAuth client beats a skill (Gmail digests, Slack posting, Trello credential storage).

## What DevPilot Gives You

- **A skill catalog.** `devpilot skill add <name>` pulls a curated Claude Code skill into `.claude/skills/`. `devpilot skill list` shows what is available and what is installed. `devpilot init` picks sensible defaults for a new project based on detected stack.
- **Gmail digest.** `devpilot gmail summary` reads unread mail via OAuth, summarises it with Claude Code, and optionally posts the digest to Slack.
- **Slack send.** `devpilot slack send` posts a message to a channel or DM with the credentials stored by `devpilot login slack`.
- **Trello helpers.** `devpilot login trello` stores credentials; `devpilot push` creates a Trello card from a markdown plan file. Skills such as `devpilot-trello` read the same credential store.

## Getting Started

### Prerequisites

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated
- Git
- *(Optional)* Trello account with API credentials for `login trello`
- *(Optional)* Google OAuth for `gmail *`
- *(Optional)* Slack OAuth for `slack send` and `gmail summary --channel`

### Installation

**From release:**
```bash
curl -sSL https://raw.githubusercontent.com/SiyuQian/devpilot/main/install.sh | sh
```

Optionally specify a version or directory:
```bash
curl -sSL https://raw.githubusercontent.com/SiyuQian/devpilot/main/install.sh | sh -s -- --version v1.0.0 --dir ~/.local/bin
```

**From source (Go 1.25+):**
```bash
git clone https://github.com/SiyuQian/devpilot.git
cd devpilot
make build
sudo mv bin/devpilot /usr/local/bin/
```

Verify: `devpilot --version`

### Quick Start

```bash
# Initialise a project — detects stack, installs a sensible skill set
devpilot init

# Browse and install additional skills
devpilot skill list
devpilot skill add devpilot-pr-review

# Summarise unread Gmail, send to Slack
devpilot login gmail
devpilot login slack
devpilot gmail summary --channel daily-digest
```

## CLI Reference

### Core Commands

| Command | Description |
|---------|-------------|
| `devpilot init` | Project setup wizard (detects stack, generates config, installs starter skills) |
| `devpilot init -y` | Accept all defaults without prompting |

### Skill Commands

| Command | Description |
|---------|-------------|
| `devpilot skill add <name[@version]>` | Install a skill from the devpilot catalog |
| `devpilot skill add --all` | Install every skill in the catalog |
| `devpilot skill list` | List available + installed skills |

### Service Commands

| Command | Description |
|---------|-------------|
| `devpilot login <service>` | Authenticate (`trello`, `gmail`, `slack`) |
| `devpilot logout <service>` | Remove stored credentials |
| `devpilot status` | Show auth status for all services |

### Gmail Commands

| Command | Description |
|---------|-------------|
| `devpilot gmail list` | List emails with optional filters |
| `devpilot gmail read <id>` | Display full email |
| `devpilot gmail mark-read <id...>` | Mark as read |
| `devpilot gmail bulk-mark-read` | Bulk mark by query |
| `devpilot gmail summary` | AI digest of unread emails (optionally to Slack) |

### Trello Commands

| Command | Description |
|---------|-------------|
| `devpilot push <plan.md> --board "Board"` | Create a Trello card from a markdown plan |

### Slack Commands

| Command | Description |
|---------|-------------|
| `devpilot slack send --channel "#channel"` | Send a Slack message |

### Generation Commands

| Command | Description |
|---------|-------------|
| `devpilot commit` | Generate a conventional commit message from staged changes |

### `devpilot commit` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-m, --message` | — | Additional context for AI |
| `--model` | *(from config)* | Override Claude model |
| `--dry-run` | `false` | Generate message without committing |

### `devpilot gmail list` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--unread` | `false` | Show only unread messages |
| `--after` | — | Show messages after date (YYYY-MM-DD) |
| `--limit` | `20` | Maximum messages to return |

### `devpilot gmail summary` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--channel` | — | Send summary to Slack channel |
| `--dm` | — | Send summary as DM to Slack user ID |
| `--no-mark-read` | `false` | Preview mode (don't mark emails as read) |

## Configuration

DevPilot stores project config in `.devpilot.yaml`. Initialize with `devpilot init`:

```yaml
# Example
source: github           # Informational only; used by init/future helpers
```

## Architecture

See [`ARCHITECTURE.md`](ARCHITECTURE.md).

## Development

```bash
make build      # Build to bin/devpilot
make test       # Run all tests
make lint       # Run linter (required before commit)
make lint-fix   # Auto-fix lint issues
make clean      # Remove build artifacts
```

Tests and lint must pass before committing. CI enforces this.

### Testing a Single Package

```bash
go test ./internal/skillmgr/ -run TestInstallSkill   # Single test by name
go test ./internal/gmail/ -v                          # Single package, verbose
```

## Tech Stack

- **Language:** Go 1.25.6
- **CLI:** [Cobra](https://github.com/spf13/cobra)
- **Selector UI:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) (skill picker in `devpilot init`)
- **Skills:** [Claude Code](https://claude.ai/code) skill system
- **External APIs:** [Trello API](https://developer.atlassian.com/cloud/trello/), [Gmail API](https://developers.google.com/gmail/api), [Slack API](https://api.slack.com/)

## License

MIT
