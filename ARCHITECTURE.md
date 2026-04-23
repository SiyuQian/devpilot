# Architecture

Read this before touching an unfamiliar part of the codebase. Describes the shape, the invariants, and where new code of type X belongs.

## System shape

```
┌───────────────────────────────────────────────────────────────┐
│  devpilot (Go CLI, Cobra)                                     │
│                                                               │
│  cmd/devpilot ──► internal/<domain>/commands.go ──► domain    │
│                                                     logic     │
│                                                               │
│  domains: auth · trello · github · taskrunner · project ·     │
│           skill · review · pm · …                             │
└────────────────┬──────────────────────────────────────────────┘
                 │
   ┌─────────────┴──────────────┐
   ▼                            ▼
task source                 execution
(Trello lists /             claude -p  ──► stream-json
 GitHub labels)                       │
   │                                  ▼
   │                           EventBridge ──► TUI (Bubble Tea)
   │                                  │
   ▼                                  ▼
 state machine                 branch → PR → auto-merge
 Ready → In Progress              (via gh)
       → Done / Failed
```

## Components

Each `internal/<domain>/` is self-contained. The important ones:

- **`cmd/devpilot/`** — wires root-level commands from every domain. Owns nothing else.
- **`internal/auth/`** — credential storage for external services. The only package that reads credentials from disk. Services implement the `Service` interface and register in `service.go`.
- **`internal/trello/`** — Trello API client + card/list state machine. Owns its Cobra commands.
- **`internal/github/`** — GitHub Issues task source; label-based state machine mirroring Trello lists.
- **`internal/taskrunner/`** — the enforcement loop. Polls a task source, branches from `main`, executes `claude -p`, opens a PR, optionally reviews, auto-merges. See `eventbridge.go` for stream-json → runner-event translation.
- **`internal/project/`** — cross-cutting repo config (`.devpilot.yaml`, skill installation, init wizard).
- **`internal/tui/`** — Bubble Tea dashboard (header · active · tools+files · claude output · footer). Consumes events via a buffered channel. Falls back to plain text when not a TTY or `--no-tui` is set.
- **`internal/review/`** — `devpilot review <pr-url>` second-pass review via `claude -p`.
- **`skills/`** (top-level, not `internal/`) — distributable skill catalog. `.claude/skills/` is the *installed* copy.

## Invariants

Violations break the system non-obviously. Pair each with a sensor where possible.

1. **`internal/auth/` is the only package that reads credentials from disk.** Other domains receive a `Service` and call it.
2. **Cobra commands live with their domain.** There is no central `cli/` router. `cmd/devpilot/main.go` only wires.
3. **External service clients live in the same package as their domain logic.** Don't create `internal/httpclient/` or similar shared client packages.
4. **One error-wrap style:** `fmt.Errorf("doing X: %w", err)` at layer boundaries.
5. **One task = one branch = one PR = one context window.** The runner's depth-first block. Cards that don't fit this shape are the bug, not the runner.
6. **`skills/` and `.claude/skills/` must stay in sync**, and every `skills/<name>/` directory must have an entry in `skills/index.json`.
7. **Always-loaded context files (`CLAUDE.md` / `AGENTS.md`) stay under ~60 lines.** Every line costs tokens on every turn.

Sensors today: `gofmt`, `goimports`, `golangci-lint`, `go test ./...`, CI, optional `devpilot review` pass. Unimplemented fitness functions (gaps): import-boundary test for (1)/(3), `skills/index.json` drift check for (6), line-count check for (7).

## Extension points

| I want to… | It goes here |
|---|---|
| Add a new service (Trello-like) | New package under `internal/<domain>/`; implement `Service` in `internal/auth/`; add Cobra commands in `commands.go` |
| Add a CLI subcommand | `commands.go` in the owning domain package — not `cmd/devpilot/` |
| Teach the agent a project-wide convention | This file's Invariants (if repo-wide + non-obvious) or `CLAUDE.md` (if always-on). Otherwise a skill. |
| Teach the agent a task-specific workflow | New skill under `skills/devpilot-<name>/`; register in `skills/index.json` |
| Encode a taste call a linter can't catch | `GOLDEN_PRINCIPLES.md` |
| Record why a feature exists or was rejected | `docs/plans/*-design.md` (historical) or `docs/rejected/*` |
| Prevent a recurring mechanical mistake | `golangci` rule or a `make lint` step — **not** prose |
| Run something after every PR | New event type + handler in `internal/taskrunner/` |
| Give agents a new external capability | CLI subcommand in the owning domain — **not** a new MCP server unless the CLI is inadequate |

## Task runner detail

Task state machine (Trello lists; GitHub uses equivalent labels):

**Ready → In Progress → Done / Failed**

Loop per card:
1. Poll "Ready" list. Sort by priority labels (P0 > P1 > P2; default P2).
2. Move card to "In Progress".
3. Create branch `task/{cardID}-{slug}` from `main`.
4. Execute plan via `claude -p` with `stream-json` output.
5. Push branch; create PR via `gh`.
6. Optionally run automated review via a second `claude -p` (disable with `--review-timeout 0`).
7. Auto-merge (`gh pr merge --squash --auto`). CI is the sensor gate.
8. Move card to "Done" (with PR link) or "Failed" (with error log path).

Per-card logs: `~/.config/devpilot/logs/{card-id}.log`.

## OpenSpec integration

When OpenSpec is installed and `openspec/changes/` exists:
- `devpilot sync` scans changes and creates/updates Trello cards or GitHub Issues.
- Card title = change directory name (used as `opsx:apply` argument).
- Card description = full content of `proposal.md` + `tasks.md`.
- Runner auto-detects OpenSpec and uses `/opsx:apply <change-name>` instead of raw plan text.
- Interrupted tasks resume from the last unchecked task.

## Skills

A skill is a `SKILL.md` (YAML frontmatter + markdown body) with optional `references/` and `scripts/`. Progressive disclosure: frontmatter is always in context, body loads on invocation, references load on demand.

- `skills/` — catalog source; what `devpilot skill add` fetches.
- `.claude/skills/` — per-project installation directory.

## Out of scope

- Deployment, release engineering → `.github/workflows/` + `docs/cli-reference.md`.
- Per-feature design rationale → `docs/plans/*-design.md`.
- Active work queue → `PLANS.md`.
- Style rules → `golangci` config + `GOLDEN_PRINCIPLES.md` for taste.

## Today's harness gaps

Ranked by blast radius. Close only when a failure is observed.

1. **No background GC refactor loop.** `GOLDEN_PRINCIPLES.md` exists; nothing sweeps for deviations. Candidate: nightly `devpilot run` consuming a "golden-principles-sweep" card.
2. **No pre-commit hook.** `make lint` relies on memory. `lefthook` or `.githooks/pre-commit` would shift left.
3. **Thin behavior harness.** No end-to-end runner test (fake board → fake `claude -p` → assert PR).
4. **No import-boundary enforcement.** Invariants 1–3 are prose only; a `depguard` rule or import-graph test would close this.
5. **No `skills/index.json` drift check.** Adding under `skills/` requires manual index updates.
6. **MCP tool bloat.** Gmail, Calendar, Drive, Notion, Pencil servers load on every session; most are unused. Scope per task.
