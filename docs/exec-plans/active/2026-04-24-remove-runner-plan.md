# Remove the Runner — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete the runner (`devpilot run`) and every sibling AI-wrapping command (`sync`, `review`, `commit`, `readme`), plus the Go packages that back them, and reposition DevPilot as a skill catalog + narrow Go helpers.

**Architecture:** Outside-in deletion across a single branch (`chore/remove-runner`). Unregister commands first, delete packages from leaves to root of the import graph, delete skills, delete the top-level `openspec/` archive directory, rewrite the project's doc surface, and finally tidy Go modules. Every commit leaves `main` in a green state.

**Tech Stack:** Go 1.25.6 · Cobra · the existing `Makefile` targets (`build`, `test`, `lint`, `check-skills-sync`).

**Spec:** [`docs/exec-plans/active/2026-04-24-remove-runner-design.md`](2026-04-24-remove-runner-design.md)

**Branch:** `chore/remove-runner` (already created; design doc already committed).

---

## File map

**Deleted (Go):**
- `internal/taskrunner/` (entire directory)
- `internal/executor/` (entire directory)
- `internal/review/` (entire directory)
- `internal/generate/` (entire directory, including `prompts/`)
- `internal/openspec/` (entire directory)

**Deleted (skills):**
- `skills/devpilot-task-executor/`
- `skills/devpilot-task-refiner/`
- `.claude/skills/devpilot-task-executor/`
- `.claude/skills/devpilot-task-refiner/`

**Deleted (other):**
- `openspec/` (top-level directory)

**Modified:**
- `cmd/devpilot/main.go` — drop four registrations + imports
- `skills/index.json` — drop two entries
- `README.md` — rewrite
- `CLAUDE.md` — rewrite identity sentence + repo map
- `AGENTS.md` — rewrite identity sentence + repo map
- `ARCHITECTURE.md` — rewrite
- `GOLDEN_PRINCIPLES.md` — remove Runner & Event System section, fix principle #5 example
- `docs/cli-reference.md` — rewrite
- `PLANS.md` — scan-only (no runner refs expected; confirm)
- `.claude/settings.local.json` — drop `Bash(bin/devpilot run:*)` entry
- `go.mod` / `go.sum` — `go mod tidy` drops `github.com/charmbracelet/bubbles`

---

## Task 1: Unregister deleted commands

**Files:**
- Modify: `cmd/devpilot/main.go`

After this task, `run`, `sync`, `review`, `commit`, `readme` are not exposed by the CLI, but the `internal/` packages still exist unreferenced.

- [ ] **Step 1: Edit `cmd/devpilot/main.go` to remove four imports and four registrations**

Replace the entire file with:

```go
package main

import (
	"fmt"
	"os"

	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/siyuqian/devpilot/internal/gmail"
	"github.com/siyuqian/devpilot/internal/initcmd"
	"github.com/siyuqian/devpilot/internal/skillmgr"
	"github.com/siyuqian/devpilot/internal/slack"
	"github.com/siyuqian/devpilot/internal/trello"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "devpilot",
		Short: "Developer toolkit for managing service integrations",
		Long:  "devpilot manages authentication and integrations for external services like Trello, GitHub, and more.",
	}

	rootCmd.Version = version

	auth.RegisterCommands(rootCmd)
	initcmd.RegisterCommands(rootCmd)
	skillmgr.RegisterCommands(rootCmd)
	trello.RegisterCommands(rootCmd)
	gmail.RegisterCommands(rootCmd)
	slack.RegisterCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify the build**

Run: `go build ./cmd/devpilot`
Expected: Exit 0, no output.

- [ ] **Step 3: Verify tests still pass**

Run: `go test ./...`
Expected: All packages PASS (the `internal/taskrunner`, `internal/review`, `internal/generate`, `internal/openspec`, `internal/executor` packages still build and test themselves — they just have no caller from `cmd/`).

- [ ] **Step 4: Verify lint**

Run: `make lint`
Expected: No errors.

- [ ] **Step 5: Verify `--help` no longer shows deleted commands**

Run: `make build && bin/devpilot --help`
Expected: Output lists available commands. `run`, `sync`, `review`, `commit`, `readme` are **not** in the list. `init`, `skill`, `login`, `logout`, `status`, `gmail`, `slack`, `trello`, `completion`, `help` are present.

- [ ] **Step 6: Commit**

```bash
git add cmd/devpilot/main.go
git commit -m "$(cat <<'EOF'
chore(cmd): unregister run/sync/review/commit/readme

First step toward removing runner and sibling AI-wrapping commands.
Packages still exist; only the CLI surface is trimmed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Delete `internal/taskrunner/`

**Files:**
- Delete: `internal/taskrunner/` (all files in the directory)

After Task 1, nothing in `cmd/` imports `taskrunner`, but `internal/taskrunner/reviewer.go` still imports `internal/review` and `internal/executor`. Once taskrunner is gone, review's and executor's inbound edges drop further.

- [ ] **Step 1: Delete the directory**

Run: `rm -rf internal/taskrunner`

- [ ] **Step 2: Verify the build**

Run: `go build ./...`
Expected: Exit 0. (If a residual import exists somewhere, Go will report it here; fix by removing the import — none expected based on the pre-plan import audit.)

- [ ] **Step 3: Verify tests**

Run: `go test ./...`
Expected: All remaining packages PASS.

- [ ] **Step 4: Verify lint**

Run: `make lint`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore: delete internal/taskrunner

Runner loop, TUI, Trello/GitHub task sources, event bridge, priority
scheduler, per-task git helpers. ~3800 LOC across 28 files. The
`devpilot run` command was unregistered in the prior commit; removing
the code completes the deletion.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Delete `internal/review/`, `internal/generate/`, `internal/openspec/`

**Files:**
- Delete: `internal/review/`
- Delete: `internal/generate/` (includes `internal/generate/prompts/` embedded templates)
- Delete: `internal/openspec/`

These three packages all only had callers in `cmd/devpilot/main.go` (unregistered in Task 1) and in `internal/taskrunner/` (deleted in Task 2). After this task, `internal/executor/` still exists but is unreferenced.

- [ ] **Step 1: Delete the three directories**

Run: `rm -rf internal/review internal/generate internal/openspec`

- [ ] **Step 2: Verify the build**

Run: `go build ./...`
Expected: Exit 0.

- [ ] **Step 3: Verify tests**

Run: `go test ./...`
Expected: All remaining packages PASS.

- [ ] **Step 4: Verify lint**

Run: `make lint`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore: delete internal/review, internal/generate, internal/openspec

The `devpilot review`, `devpilot commit`, `devpilot readme`, and
`devpilot sync` commands were unregistered; their backing packages
(all thin `claude -p` wrappers or the OpenSpec → Trello/GitHub bridge)
are removed. Replaced by the devpilot-pr-review, devpilot-pr-creator,
and devpilot-auto-feature skills.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Delete `internal/executor/`

**Files:**
- Delete: `internal/executor/`

After Task 3, `internal/executor/` has zero inbound imports. Safe to remove.

- [ ] **Step 1: Confirm executor has no remaining callers**

Run: `grep -rn "siyuqian/devpilot/internal/executor" cmd/ internal/ 2>/dev/null | grep -v "_test.go"`
Expected: No output.

- [ ] **Step 2: Delete the directory**

Run: `rm -rf internal/executor`

- [ ] **Step 3: Verify the build**

Run: `go build ./...`
Expected: Exit 0.

- [ ] **Step 4: Verify tests**

Run: `go test ./...`
Expected: All remaining packages PASS.

- [ ] **Step 5: Verify lint**

Run: `make lint`
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore: delete internal/executor

The `claude -p --output-format stream-json` Go wrapper is no longer
referenced after taskrunner and review were removed. Claude Code
skills invoke Claude directly; the Go wrapper is dead code.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Delete runner-specific skills

**Files:**
- Delete: `skills/devpilot-task-executor/`
- Delete: `skills/devpilot-task-refiner/`
- Delete: `.claude/skills/devpilot-task-executor/`
- Delete: `.claude/skills/devpilot-task-refiner/`
- Modify: `skills/index.json`

Both skills describe themselves as "used by the devpilot runner" — with the runner gone they have no caller.

- [ ] **Step 1: Delete skill source directories**

Run: `rm -rf skills/devpilot-task-executor skills/devpilot-task-refiner`

- [ ] **Step 2: Delete installed skill copies**

Run: `rm -rf .claude/skills/devpilot-task-executor .claude/skills/devpilot-task-refiner`

- [ ] **Step 3: Remove the two entries from `skills/index.json`**

Open `skills/index.json`. Remove the `devpilot-task-executor` object (lines matching `"name": "devpilot-task-executor"`) and the `devpilot-task-refiner` object (lines matching `"name": "devpilot-task-refiner"`). After the edit the `"skills"` array should contain 15 entries (down from 17): `devpilot-auto-feature`, `devpilot-clean-code-principles`, `devpilot-confluence-reviewer`, `devpilot-content-creator`, `devpilot-google-go-style`, `devpilot-harness-engineering`, `devpilot-learn`, `devpilot-news-digest`, `devpilot-pm`, `devpilot-pr-creator`, `devpilot-pr-review`, `devpilot-product-research`, `devpilot-prompt-review`, `devpilot-scanning-repos`, `devpilot-trello`.

- [ ] **Step 4: Verify `skills/index.json` is valid JSON**

Run: `python3 -c "import json; d=json.load(open('skills/index.json')); print(len(d['skills']))"`
Expected: `15`

- [ ] **Step 5: Verify no drift between `skills/` and `.claude/skills/`**

Run: `make check-skills-sync`
Expected: No output (drift check passes).

- [ ] **Step 6: Verify `devpilot skill list` no longer shows the deleted skills**

Run: `make build && bin/devpilot skill list`
Expected: Output lists 15 skills. `devpilot-task-executor` and `devpilot-task-refiner` do **not** appear.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore(skills): delete devpilot-task-executor and devpilot-task-refiner

Both skills' descriptions say "used by the devpilot runner". With the
runner gone they have no caller. Removed from skills/, .claude/skills/,
and skills/index.json.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Delete top-level `openspec/` directory

**Files:**
- Delete: `openspec/` (top-level; not `internal/openspec/` which went in Task 3)

The directory holds 21 OpenSpec specs and 3 in-flight changes. Most of them describe already-deleted features (`review-*`, `commit-*`, `email-assistant-skill`). Accept the loss of OpenSpec's change-log trail; git history is the only record going forward.

- [ ] **Step 1: Confirm this is the top-level directory, not the Go package**

Run: `ls openspec/`
Expected: Output shows `changes/`, `config.yaml`, `specs/` (not Go files).

- [ ] **Step 2: Delete the directory**

Run: `rm -rf openspec`

- [ ] **Step 3: Verify the build and tests (sanity — no Go code changed)**

Run: `go build ./... && go test ./...`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "$(cat <<'EOF'
chore: delete top-level openspec/ directory

Most specs here describe already-removed features (review-*, commit-*,
email-assistant-skill). The surviving ones (skill-*, gmail-*, slack-*)
don't justify keeping the directory alive alone, and `devpilot sync`
— the only tool that consumed this tree — was removed in Task 3.
Change history preserved in git.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Rewrite project documents

This task is split into sub-steps, one per document, with full new content. Each sub-step is its own Edit/Write; commit at the end of the task, not between sub-steps.

### 7.1 — Rewrite `README.md`

- [ ] **Step 1: Replace the file with the new content**

Overwrite `README.md` with:

````markdown
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
````

### 7.2 — Rewrite `CLAUDE.md`

- [ ] **Step 2: Replace the file with the new content**

Overwrite `CLAUDE.md` with:

```markdown
# DevPilot

Skill catalog for Claude Code plus a small set of Go-native helpers (Gmail OAuth digests, Slack sending, Trello credential storage) that complement the catalog where a typed OAuth client beats a skill.

## Repo map
- `cmd/devpilot/` — entry point
- `internal/<domain>/` — self-contained domains (`auth`, `trello`, `gmail`, `slack`, `initcmd`, `skillmgr`, `project`); each owns its Cobra commands in `commands.go`
- `skills/` — distributable skill catalog (register each in `skills/index.json`)
- `.claude/skills/` — installed skills for this project
- `.github/workflows/` — CI (test + release)
- `docs/` — on-demand references

## Build / test / lint
```
make build    # → bin/devpilot
make test     # go test ./...
make lint     # golangci-lint; must pass before commit
```

## Conventions the agent keeps getting wrong
- Cobra commands live with their domain in `commands.go` — no central `cli/` router.
- Constructors with optional params use functional options (`WithXxx()`), never positional bool flags.
- Wrap errors at layer boundaries: `fmt.Errorf("doing X: %w", err)`.
- Tests are table-driven with named subtests; don't mock our own packages.
- When adding/removing a skill under `skills/`, update `skills/index.json` in the same PR.
- Skill helper scripts use Python 3.

## Pointers (read on demand; do not inline)
- System shape + invariants: `ARCHITECTURE.md`
- Taste calls: `GOLDEN_PRINCIPLES.md`
- Active plans: `PLANS.md`
- CLI surface: `docs/cli-reference.md`

## Safety rules the harness can't enforce
- Never commit without an explicit user ask.
- Never push to `main`; always work on a branch.
```

### 7.3 — Rewrite `AGENTS.md`

- [ ] **Step 3: Replace the file with the new content**

Overwrite `AGENTS.md` with the same content as `CLAUDE.md` (they are intentionally paired and hand-maintained in lockstep):

```markdown
# DevPilot

Skill catalog for Claude Code plus a small set of Go-native helpers (Gmail OAuth digests, Slack sending, Trello credential storage) that complement the catalog where a typed OAuth client beats a skill.

## Repo map
- `cmd/devpilot/` — entry point
- `internal/<domain>/` — self-contained domains (`auth`, `trello`, `gmail`, `slack`, `initcmd`, `skillmgr`, `project`); each owns its Cobra commands in `commands.go`
- `skills/` — distributable skill catalog (register each in `skills/index.json`)
- `.claude/skills/` — installed skills for this project
- `.github/workflows/` — CI (test + release)
- `docs/` — on-demand references

## Build / test / lint
```
make build    # → bin/devpilot
make test     # go test ./...
make lint     # golangci-lint; must pass before commit
```

## Conventions the agent keeps getting wrong
- Cobra commands live with their domain in `commands.go` — no central `cli/` router.
- Constructors with optional params use functional options (`WithXxx()`), never positional bool flags.
- Wrap errors at layer boundaries: `fmt.Errorf("doing X: %w", err)`.
- Tests are table-driven with named subtests; don't mock our own packages.
- When adding/removing a skill under `skills/`, update `skills/index.json` in the same PR.
- Skill helper scripts use Python 3.

## Pointers (read on demand; do not inline)
- System shape + invariants: `ARCHITECTURE.md`
- Taste calls: `GOLDEN_PRINCIPLES.md`
- Active plans: `PLANS.md`
- CLI surface: `docs/cli-reference.md`

## Safety rules the harness can't enforce
- Never commit without an explicit user ask.
- Never push to `main`; always work on a branch.
```

### 7.4 — Rewrite `ARCHITECTURE.md`

- [ ] **Step 4: Replace the file with the new content**

Overwrite `ARCHITECTURE.md` with:

````markdown
# Architecture

Read this before touching an unfamiliar part of the codebase. Describes the shape, the invariants, and where new code of type X belongs.

## System shape

```
┌─────────────────────────────────────────────────────────────┐
│  devpilot (Go CLI, Cobra)                                   │
│                                                             │
│  cmd/devpilot ──► internal/<domain>/commands.go ──► domain  │
│                                                     logic   │
│                                                             │
│  domains: auth · trello · gmail · slack · initcmd ·         │
│           skillmgr · project                                │
└────────────────┬────────────────────────────────────────────┘
                 │
      ┌──────────┴───────────┐
      ▼                      ▼
 skill catalog          external services
 (devpilot skill)       (Trello / Gmail / Slack)
      │                      │
      ▼                      ▼
 installs into          OAuth via internal/auth
 .claude/skills/        creds at ~/.config/devpilot/credentials.json
```

DevPilot has two jobs: (1) distribute Claude Code skills (the `skills/` catalog, `devpilot skill add`, `devpilot init`), and (2) host thin Go CLIs where an OAuth flow or typed client is a better fit than a Claude skill (Gmail digest, Slack post, Trello credential store).

## Components

Each `internal/<domain>/` is self-contained. The important ones:

- **`cmd/devpilot/`** — wires root-level commands from every domain. Owns nothing else.
- **`internal/auth/`** — credential storage for external services. The only package that reads credentials from disk. Services implement the `Service` interface and register in `service.go`.
- **`internal/trello/`** — Trello API client + `devpilot push` (create card from plan); `devpilot login trello` flow.
- **`internal/gmail/`** — Gmail OAuth client + `devpilot gmail list | read | mark-read | bulk-mark-read | summary`. The `summary` subcommand is the only place that still shells out to `claude` for AI work.
- **`internal/slack/`** — Slack OAuth client + `devpilot slack send`.
- **`internal/initcmd/`** — `devpilot init` scaffolding: detects project stack, writes `.devpilot.yaml`, installs starter skills.
- **`internal/skillmgr/`** — `devpilot skill add | list`. Downloads from the catalog, syncs `skills/` ↔ `.claude/skills/`. Uses Bubble Tea for an interactive skill picker.
- **`internal/project/`** — cross-cutting repo config (`.devpilot.yaml` shape, stack detection).
- **`skills/`** (top-level, not `internal/`) — distributable skill catalog. `.claude/skills/` is the *installed* copy.

## Invariants

Violations break the system non-obviously. Pair each with a sensor where possible.

1. **`internal/auth/` is the only package that reads credentials from disk.** Other domains receive a `Service` and call it.
2. **Cobra commands live with their domain.** There is no central `cli/` router. `cmd/devpilot/main.go` only wires.
3. **External service clients live in the same package as their domain logic.** Don't create `internal/httpclient/` or similar shared client packages.
4. **One error-wrap style:** `fmt.Errorf("doing X: %w", err)` at layer boundaries.
5. **`skills/` and `.claude/skills/` must stay in sync**, and every `skills/<name>/` directory must have an entry in `skills/index.json`.
6. **Always-loaded context files (`CLAUDE.md` / `AGENTS.md`) stay under ~60 lines.** Every line costs tokens on every turn.

Sensors today: `gofmt`, `goimports`, `golangci-lint`, `go test ./...`, CI, `make check-skills-sync`. Unimplemented fitness functions (gaps): import-boundary test for (1)/(3), line-count check for (6).

## Extension points

| I want to… | It goes here |
|---|---|
| Add a new OAuth-backed service (Trello-like) | New package under `internal/<domain>/`; implement `Service` in `internal/auth/`; add Cobra commands in `commands.go` |
| Add a CLI subcommand | `commands.go` in the owning domain package — not `cmd/devpilot/` |
| Teach the agent a project-wide convention | This file's Invariants (if repo-wide + non-obvious) or `CLAUDE.md` (if always-on). Otherwise a skill. |
| Teach the agent a task-specific workflow | New skill under `skills/devpilot-<name>/`; register in `skills/index.json` |
| Encode a taste call a linter can't catch | `GOLDEN_PRINCIPLES.md` |
| Record why a feature exists or was rejected | `docs/plans/*-design.md` (historical) or `docs/rejected/*` |
| Prevent a recurring mechanical mistake | `golangci` rule or a `make lint` step — **not** prose |
| Give agents a new external capability | Skill first; CLI subcommand only if OAuth or a typed client is required; MCP server only if both are inadequate |

## Skills

A skill is a `SKILL.md` (YAML frontmatter + markdown body) with optional `references/` and `scripts/`. Progressive disclosure: frontmatter is always in context, body loads on invocation, references load on demand.

- `skills/` — catalog source; what `devpilot skill add` fetches.
- `.claude/skills/` — per-project installation directory.
- `skills/index.json` — manifest mapping name → files; required for the installer.
- `make sync-skills` / `make check-skills-sync` — drift guard.

## Out of scope

- Deployment, release engineering → `.github/workflows/` + `docs/cli-reference.md`.
- Per-feature design rationale → `docs/plans/*-design.md`.
- Active work queue → `PLANS.md`.
- Style rules → `golangci` config + `GOLDEN_PRINCIPLES.md` for taste.

## Today's harness gaps

Ranked by blast radius. Close only when a failure is observed.

1. **No pre-commit hook.** `make lint` relies on memory. `lefthook` or `.githooks/pre-commit` would shift left.
2. **No import-boundary enforcement.** Invariants 1–3 are prose only; a `depguard` rule or import-graph test would close this.
3. **No line-count check for `CLAUDE.md` / `AGENTS.md`.** Invariant 6 is unenforced.
4. **MCP tool bloat.** Gmail, Calendar, Drive, Notion, and similar servers load on every session; most are unused. Scope per task.
````

### 7.5 — Edit `GOLDEN_PRINCIPLES.md`

- [ ] **Step 5: Update principle #5's example and "why" line**

Edit `GOLDEN_PRINCIPLES.md`. Find:
```
### 5. Functional options for any constructor with more than one optional parameter
New clients, executors, runners expose `NewXxx(required, ...Option)` with `WithYyy(v) Option` helpers.
```
Replace with:
```
### 5. Functional options for any constructor with more than one optional parameter
New clients expose `NewXxx(required, ...Option)` with `WithYyy(v) Option` helpers.
```

Find:
```
**Why:** testability, forward-compatibility, and our existing `Executor` + `trello.Client` already use this pattern — every new constructor joining them keeps the codebase consistent.
```
Replace with:
```
**Why:** testability, forward-compatibility, and `trello.Client` already uses this pattern — every new constructor joining it keeps the codebase consistent.
```

- [ ] **Step 6: Delete the entire "Runner & Event System" section**

Edit `GOLDEN_PRINCIPLES.md`. Delete the lines from:
```
## Runner & Event System

### 17. Runner events are additive, not mutated
```
…through and including:
```
### 19. Per-card logs to `~/.config/devpilot/logs/{card-id}.log`
Don't invent parallel log locations. One path, one format.
```

(Principles 17, 18, 19 and their "Runner & Event System" heading are removed. The next section after this deletion should be `## Anti-Principles (Do Not Do)`.)

### 7.6 — Rewrite `docs/cli-reference.md`

- [ ] **Step 7: Replace the file with the new content**

Overwrite `docs/cli-reference.md` with:

````markdown
# DevPilot CLI Reference

Full command surface. CLAUDE.md links here; this file is read on demand.

## Build & Development

```bash
make build                         # Build binary to bin/devpilot
make test                          # Run all tests (go test ./...)
make lint                          # Run golangci-lint (must pass before commit)
make lint-fix                      # Auto-fix lint issues where possible
make run ARGS="--help"             # Build and run with arguments
make clean                         # Remove bin/
```

Run a single test:
```bash
go test ./internal/skillmgr/ -run TestInstallSkill   # Single test by name
go test ./internal/skillmgr/ -v                       # Single package, verbose
```

## Authentication & Status

```bash
devpilot login trello                # Authenticate with Trello (API key + token)
devpilot login gmail                 # Authenticate with Gmail (OAuth)
devpilot login slack                 # Authenticate with Slack (OAuth)
devpilot logout <service>            # Remove stored credentials
devpilot status                      # Show authentication status for all services
```

## Project Setup

```bash
devpilot init                        # Interactive project setup wizard
devpilot init -y                     # Accept all defaults
```

## Skills

```bash
devpilot skill add <name>                       # Install a skill (prompts for project/user level)
devpilot skill add <name>@<ref>                 # Install at specific git ref
devpilot skill add <name> --level user          # Install at user level non-interactively
devpilot skill add --all                        # Install every skill in the catalog
devpilot skill add --all --level project        # Bulk install at project level, no prompt
devpilot skill list                             # List available skills with install status
devpilot skill list --installed                 # List only installed skills
```

## Trello

```bash
devpilot push <plan.md> --board "Board Name"                 # Create Trello card from plan file
devpilot push <plan.md> --board "Board Name" --list "Ready"  # Specify target list (default: Ready)
```

## Gmail

```bash
devpilot gmail list                            # List recent emails
devpilot gmail list --unread --limit 10        # Filter
devpilot gmail read <id>                       # Display full email
devpilot gmail mark-read <id...>               # Mark as read
devpilot gmail bulk-mark-read --query "..."    # Bulk mark by Gmail query
devpilot gmail summary                         # Dry run: summarize unread emails (won't mark as read)
devpilot gmail summary --channel daily-digest  # Send summary to a Slack channel (marks as read)
devpilot gmail summary --dm U0123ABCDE         # Send summary as a DM (marks as read)
```

## Slack

```bash
devpilot slack send --channel "#general" --text "hi"   # Send a Slack message
```

## Skill Helper Scripts (Python 3)

```bash
python3 .claude/skills/skill-creator/scripts/init_skill.py       # Scaffold a new skill
python3 .claude/skills/skill-creator/scripts/package_skill.py    # Package a skill for distribution
python3 .claude/skills/skill-creator/scripts/quick_validate.py   # Validate skill structure
```
````

### 7.7 — Scan `PLANS.md`

- [ ] **Step 8: Scan for runner references in `PLANS.md`**

Run: `grep -in "runner\|taskrunner\|internal/executor\|internal/review\|internal/generate\|internal/openspec\|devpilot run\|devpilot sync\|devpilot review\|devpilot commit\|devpilot readme" PLANS.md`

Expected: Either no output (ideal) or only the one line added in the design-commit referencing this removal work (e.g., `"Remove the runner and sibling AI-wrapping commands"`). That line is the active plan entry and stays. If any other runner-specific line appears, delete it.

### 7.8 — Drop the stale permission from `.claude/settings.local.json`

- [ ] **Step 9: Remove `Bash(bin/devpilot run:*)` from the allow list**

Edit `.claude/settings.local.json`. Find the line:
```
      "Bash(bin/devpilot run:*)",
```
Delete it (including the trailing comma on that line; the preceding and following entries keep their own commas).

- [ ] **Step 10: Verify JSON validity**

Run: `python3 -c "import json; json.load(open('.claude/settings.local.json')); print('ok')"`
Expected: `ok`

### 7.9 — Verify all docs compile and commit

- [ ] **Step 11: Verify the build, tests, and lint still pass (no Go changes, but full check before commit)**

Run: `go build ./... && go test ./... && make lint`
Expected: All green.

- [ ] **Step 12: Verify `CLAUDE.md` and `AGENTS.md` are under ~60 lines each (invariant 6)**

Run: `wc -l CLAUDE.md AGENTS.md`
Expected: Each file ≤ 60 lines.

- [ ] **Step 13: Commit**

```bash
git add README.md CLAUDE.md AGENTS.md ARCHITECTURE.md GOLDEN_PRINCIPLES.md docs/cli-reference.md PLANS.md .claude/settings.local.json
git commit -m "$(cat <<'EOF'
docs: reposition devpilot as a skill catalog + OAuth helpers

Rewrite README, CLAUDE.md, AGENTS.md, ARCHITECTURE.md, and the CLI
reference to describe the surface that remains after the runner and
sibling AI-wrapping commands were removed. Drop the Runner & Event
System section from GOLDEN_PRINCIPLES.md and fix principle #5's
example. Scan PLANS.md for stale entries. Drop the stale
`bin/devpilot run` permission from .claude/settings.local.json.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: `go mod tidy` + final verification

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Run `go mod tidy`**

Run: `go mod tidy`
Expected: `github.com/charmbracelet/bubbles` drops from `go.mod` (it was only used by `internal/generate/`'s spinner). `bubbletea` and `lipgloss` **remain** because `internal/skillmgr/select.go` still imports them. Minor indirect deps may also drop.

- [ ] **Step 2: Verify the build and tests are still green**

Run: `go build ./... && go test ./...`
Expected: All PASS.

- [ ] **Step 3: Verify `make lint` passes**

Run: `make lint`
Expected: No errors.

- [ ] **Step 4: Final sanity — `devpilot --help` surface**

Run: `bin/devpilot --help`
Expected available commands (order may vary): `completion`, `gmail`, `help`, `init`, `login`, `logout`, `push`, `skill`, `slack`, `status`, `trello`. **Not** present: `run`, `sync`, `review`, `commit`, `readme`.

- [ ] **Step 5: Final grep sweep — no stray references to deleted packages/skills anywhere except historical archives**

Run:
```bash
grep -rn "taskrunner\|internal/executor\|internal/review\|internal/generate\|internal/openspec\|task-executor\|task-refiner" . \
  --exclude-dir=.git \
  --exclude-dir=node_modules \
  --exclude-dir=bin
```
Expected: Matches only in:
- `docs/plans/` (historical archive — untouched per design)
- `docs/rejected/` (historical archive — untouched per design)
- `docs/exec-plans/active/2026-04-24-remove-runner-design.md` (this work's design doc)
- `docs/exec-plans/active/2026-04-24-remove-runner-plan.md` (this plan)

Any other match is a leak to fix before committing.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum
git commit -m "$(cat <<'EOF'
chore(deps): go mod tidy after runner removal

Drops github.com/charmbracelet/bubbles (only used by the deleted
generate spinner). bubbletea and lipgloss remain — skillmgr's
interactive picker still uses them.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 7: Push the branch and open a PR**

Push:
```bash
git push -u origin chore/remove-runner
```

Open a PR with this description (the PR itself is user-authorised to open — confirm before running `gh pr create`):

```
## Summary

Removes the autonomous runner (`devpilot run`) and every sibling command that was
a thin `claude -p` wrapper (`sync`, `review`, `commit`, `readme`). Repositions
DevPilot as a skill catalog plus a small set of OAuth-backed Go helpers.

## What's deleted

- `internal/taskrunner/` (~3800 LOC)
- `internal/executor/` (~1200 LOC)
- `internal/review/`, `internal/generate/`, `internal/openspec/`
- Skills: `devpilot-task-executor`, `devpilot-task-refiner`
- Top-level `openspec/` directory

## Migration notes (breaking)

| Removed | Replacement |
|---|---|
| `devpilot run` | `devpilot-auto-feature`, `devpilot-task-executor` (Claude Code skills) |
| `devpilot sync` | Author OpenSpec changes directly; invoke the skill workflow |
| `devpilot review` | `devpilot-pr-review` skill |
| `devpilot commit` | `/commit` slash command / Claude Code skill of choice |
| `devpilot readme` | Claude Code skill of choice |

`.devpilot.yaml` fields `board:`, `source:`, `models:` become dead fields; no
migration.

## Test plan
- [ ] `make build && make test && make lint` green
- [ ] `bin/devpilot --help` lists only `init / skill / login / logout / status / gmail / slack / trello / push / completion / help`
- [ ] `bin/devpilot skill list` lists 15 skills (no task-executor/task-refiner)
- [ ] `make check-skills-sync` green
- [ ] CI green
```

---

## Self-review

(This section is internal to plan authoring; an executing agent can skip it.)

**Spec coverage:** Every item in the design's Scope table and Execution ladder is covered by a numbered task. Design's "Migration and residue" is captured in the PR description (Task 8 Step 7). Design's "Risks" don't need separate tasks — they are covered by release-note content in the PR description and by the verification gate at the end of Task 8.

**Placeholder scan:** No TBDs; every `Expected:` is concrete; every rewritten doc has full inline content. No "fill in details".

**Type consistency:** No new types introduced. The only cross-task identifiers are filenames and package paths, and those are consistent.

---
