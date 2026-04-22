# DevPilot Architecture

Subsystem-level detail. CLAUDE.md links here; this file is read on demand.

## CLI

Go CLI using Cobra for subcommand routing. Entry point in `cmd/devpilot/main.go` wires root-level commands from domain packages under `internal/`. Adding a new service: implement the `Service` interface in `internal/auth/`, register in `service.go`.

## Project Init (`devpilot init`)

Interactive wizard that detects project state and generates missing pieces:
- Detects: `.devpilot.yaml`, Trello credentials, skills, git repo
- Generates: board config, gitignore entries, skill installation
- Task source configuration (Trello/GitHub) can be skipped

## Task Runner (`devpilot run`)

Tasks move through a state machine. Trello uses lists (**Ready** → **In Progress** → **Done** / **Failed**); GitHub Issues use labels (`devpilot` → `in-progress` → closed or `failed`).

1. Polls "Ready" list for cards
2. Sorts cards by priority (P0 > P1 > P2 labels; default P2)
3. Moves card to "In Progress"
4. Creates branch `task/{cardID}-{slug}` from main
5. Executes plan via `claude -p` with `stream-json` output
6. Pushes branch, creates PR via `gh`
7. Optionally runs automated code review via a second `claude -p` invocation
8. Auto-merges PR (`gh pr merge --squash --auto`)
9. Moves card to "Done" (with PR link) or "Failed" (with error log path)

Logs per-card output to `~/.config/devpilot/logs/{card-id}.log`.

## OpenSpec Integration

When OpenSpec is installed and `openspec/changes/` exists:
- `devpilot sync` scans changes and creates/updates Trello cards or GitHub Issues
- Card title = change directory name (used as `opsx:apply` argument)
- Card description = full content of proposal.md + tasks.md
- Runner auto-detects OpenSpec and uses `/opsx:apply <change-name>` instead of raw plan text
- Supports resumability: interrupted tasks pick up from last unchecked task

## TUI Dashboard

When `devpilot run` launches in a TTY, it displays a real-time Bubble Tea dashboard:
- **Header**: Board name, runner phase, token stats
- **Status & Active**: Trello list states + current card info
- **Tools & Files**: Tool call history with durations + file access tracking
- **Claude Output**: Scrollable text output
- **Footer**: Completed task history + errors

Keyboard: `q`/`Ctrl-C` quit, `Tab` switch pane, `j/k/↑/↓` scroll, `g/G` top/bottom.

Falls back to plain text mode when not a TTY or `--no-tui` is set.

## Event System

The runner uses an event-driven architecture:
- **Runner** emits lifecycle events (`CardStarted`, `CardDone`, `ToolStart`, `TextOutput`, etc.)
- **EventBridge** parses `claude -p` stream-json output and translates to runner events
- **TUI** receives events via buffered channel (size 100) and updates the Bubble Tea model

## Skills

Skills are defined by a `SKILL.md` file (YAML frontmatter + markdown body) with optional `references/` and `scripts/` directories. They use progressive disclosure: frontmatter metadata is always in context, body loads on invocation, references load on demand.

**Directory distinction:**
- `skills/` at the project root — Catalog source code for distributable skills (prefixed `devpilot-<name>/`). This is what gets fetched by `devpilot skill add`.
- `.claude/skills/` — Where skills are installed in a project (so Claude Code discovers them). Also where project-local OpenSpec workflow skills live.
