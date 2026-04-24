# Remove the Runner — Design

Date: 2026-04-24
Status: Active (design approved; plan to follow)

## Motivation

DevPilot's autonomous runner (`devpilot run`) and its sibling AI-wrapping
commands (`devpilot sync`, `devpilot review`, `devpilot commit`, `devpilot
readme`) were built before Claude Code skills existed. Every one of them
ultimately shells out to `claude -p` with a prompt and some scaffolding. The
same job is now better done by invoking skills directly from Claude Code:

- `devpilot-auto-feature`, `devpilot-task-executor` → replace the runner's
  "execute a plan autonomously" loop
- `devpilot-pr-review` → replaces `devpilot review`
- `devpilot-pr-creator` → replaces branch/PR creation inside the runner
- `devpilot-trello` → replaces Trello-specific Go code for day-to-day task
  management
- `devpilot-harness-engineering` → replaces most of what `devpilot init`
  generates for agent context

Keeping a Go re-implementation of Claude Code's own workflow is pure
maintenance cost: ~6k lines of Go, a TUI layer, and several overlapping
skills that say "used by the runner" in their description. This change
removes all AI-wrapping Go code and repositions DevPilot as a skill catalog
plus a few narrow helpers that are genuinely better in Go (OAuth flows,
Gmail client, Slack client).

## Scope

One PR, aggressive deletion, no deprecation window. The repo is small and
primarily single-user; a deprecation window protects no meaningful user
base.

### Deleted — Go packages

| Package | Approx LOC | Reason |
|---|---|---|
| `internal/taskrunner/` | ~3800 | The runner itself: polling, prioritisation, TUI, task sources, event bridge |
| `internal/executor/` | ~1200 | `claude -p --output-format stream-json` wrapper; only callers are `taskrunner` and `review`, both deleted |
| `internal/review/` | ~400 | `devpilot review` — replaced by `devpilot-pr-review` skill |
| `internal/generate/` | ~600 | `devpilot commit` + `devpilot readme` — thin `claude -p` wrappers |
| `internal/openspec/` | ~400 | `devpilot sync` — only consumer of OpenSpec→Trello/GitHub bridge |

### Deleted — Skills

- `skills/devpilot-task-executor/` (and the mirror under `.claude/skills/`)
- `skills/devpilot-task-refiner/` (and the mirror)
- Both entries in `skills/index.json`

### Deleted — CLI commands

`run`, `sync`, `review`, `fix` (lives inside the review package), `commit`,
`readme`.

### Deleted — Top-level directories

- `openspec/` — the OpenSpec metadata directory. Specs for already-deleted
  features (`review-*`, `commit-*`, `email-assistant-skill`) dominate this
  tree and describing still-living features from this archive is more
  confusing than useful. Accept that git history is the only record going
  forward.

### Kept — Go packages

- `cmd/devpilot/main.go` (trimmed to register only remaining commands)
- `internal/auth/` — OAuth + credential store
- `internal/gmail/` — Gmail client + AI digest
- `internal/slack/` — Slack client
- `internal/initcmd/` — `devpilot init` project scaffolding
- `internal/skillmgr/` — `devpilot skill add | list`
- `internal/project/` — shared stack detection
- `internal/trello/` — required by `devpilot login trello` and `initcmd`

### Kept — CLI commands

`init`, `skill add | list`, `login | logout | status`, `gmail *`, `slack
send`, plus Cobra's `completion` / `help`.

### Rewritten — documents

- `README.md` — the opening pitch ("Autonomous development workflow
  automation …") describes only the deleted runner. Rewrite the project
  identity plus the command/config tables.
- `CLAUDE.md` — strip `taskrunner` and related entries from the repo map
  and conventions list.
- `AGENTS.md` — line 3 carries the same identity sentence as CLAUDE.md;
  keep them in sync.
- `ARCHITECTURE.md` — the "Runner → TaskSource + Executor → EventBridge →
  TUI/Logger" diagram is the entire document; replace with the new
  architecture.
- `GOLDEN_PRINCIPLES.md` — delete the "Runner & Event System" section
  (from line 108); fix the line 43 example that cites `Executor` as the
  canonical functional-options user.
- `docs/cli-reference.md` — entire sections ("Running the Autonomous
  Runner", "OpenSpec Sync", "Code Review", `commit`/`readme`) go; rewrite
  around the remaining command surface.
- `PLANS.md` — scan for runner references (none expected; confirm).
- `.claude/settings.local.json` — drop the `Bash(bin/devpilot run:*)`
  permission entry.
- `skills/index.json` — remove the two deleted skill entries.

### Migration and residue

- `.devpilot.yaml` fields `board:`, `source:`, `models:` become dead
  fields. **No automatic migration.** Release notes must call this out.
- `~/.config/devpilot/logs/{task-id}.log` — existing runner logs are left
  in place; no cleanup code, user can delete manually.
- User CI that invokes deleted commands will fail. Release notes must
  list every deleted command and its skill replacement.

### Left alone (historical archive)

- `docs/plans/` — ~30 paired design+plan files from the era we were
  building the runner. `PLANS.md` already flags this tree as "archive, not
  maintained". No edits.
- `docs/rejected/` — rejected runner feature ideas. No edits.

## Execution — one PR, eight commits

Each commit must leave `main` in a state where `go build ./cmd/devpilot`,
`go test ./...`, and `make lint` all pass.

1. **Unregister commands.** Remove the four `RegisterCommands` calls and
   their imports from `cmd/devpilot/main.go`. The `internal/` packages
   still exist but have no callers.
2. **Delete `internal/taskrunner/`.** No other package imports it after
   Step 1.
3. **Delete `internal/review/`, `internal/generate/`, `internal/openspec/`.**
   `internal/executor/` remains but is unreferenced.
4. **Delete `internal/executor/`.**
5. **Delete runner-specific skills.** Remove
   `skills/devpilot-task-executor/`, `skills/devpilot-task-refiner/`, and
   their mirrors under `.claude/skills/`. Update `skills/index.json`. Run
   `make check-skills-sync`.
6. **Delete `openspec/` top-level directory.**
7. **Rewrite documents.** `README.md`, `CLAUDE.md`, `AGENTS.md`,
   `ARCHITECTURE.md`, `GOLDEN_PRINCIPLES.md`, `docs/cli-reference.md`,
   `PLANS.md` scan, `.claude/settings.local.json`.
8. **`go mod tidy`.** `github.com/charmbracelet/bubbles` should drop (only
   `generate`'s spinner used it); `bubbletea` and `lipgloss` stay because
   `internal/skillmgr/select.go` still uses them.

Ordering rationale: work outside-in so every intermediate state compiles;
isolate Step 1 so bisect can distinguish "command unregistered" from
"package deleted"; leave docs for last when the code delta is settled;
`go mod tidy` absolute last to avoid IDE/git churn mid-sequence.

## Verification

Per-commit gate:

- `go build ./cmd/devpilot` passes
- `go test ./...` passes
- `make lint` passes

Pre-merge gate:

- `bin/devpilot --help` lists exactly: `init`, `skill`, `login`, `logout`,
  `status`, `gmail`, `slack`, plus Cobra's `completion` / `help`
- `bin/devpilot skill list` runs
- `bin/devpilot init --help`, `bin/devpilot gmail --help`, `bin/devpilot
  login trello --help` run
- CI green on `.github/workflows/test.yml`
- `make check-skills-sync` green
- Final grep turns up no stray references:
  ```
  grep -rn "taskrunner\|internal/executor\|internal/review\|internal/generate\|internal/openspec\|task-executor\|task-refiner" .
  ```
  Only matches allowed: `docs/plans/` and `docs/rejected/` (archive).

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| User CI invokes a deleted command | Pipeline breaks on upgrade | Release notes enumerate every removed command + skill replacement |
| `.devpilot.yaml` dead fields look active | Silent no-op — users confused | Release notes call this out; no migration logic |
| `internal/trello/` becomes thinly-used | Large package, narrow caller surface | Out of scope here; revisit separately |
| Deleting `openspec/` loses spec history | OpenSpec change-log trail gone | Accepted trade-off (scope choice 7a); git history preserves it |
| `skillmgr/select.go` still uses bubbletea | `bubbletea` stays as a dep | Out of scope; acceptable |

## New project identity (direction only — drafted during writing-plans)

Old: *"Autonomous development workflow automation. Write a plan in
markdown, track it in Trello or GitHub Issues, and let DevPilot execute
it."*

New direction: DevPilot is (1) a **skill catalog** distributed via
`devpilot skill add` and installed via `devpilot init`, plus (2) a small
set of **Go-native helpers** that complement the skill ecosystem where Go
is genuinely the better fit — Gmail OAuth + AI digest, Slack sending,
Trello/Gmail/Slack credential storage.

Final wording for README / CLAUDE.md / AGENTS.md is drafted in the
implementation plan, not here.

## Out of scope

- Refactoring `internal/trello/` now that its only callers are `login` and
  `initcmd`
- Removing `bubbletea` / `lipgloss` from `internal/skillmgr/select.go`
- Version-bump strategy beyond "this is a breaking change; release note
  accordingly"
- Migration tooling for `.devpilot.yaml`
