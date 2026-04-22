# DevPilot's Agent Harness

A map of how this repository is set up to let coding agents (Claude Code, Codex, DevPilot's own runner) ship reliably. Read this when:

- You're about to add a new guardrail, skill, linter, or hook and want to know where it belongs.
- Agent output quality has regressed and you need to find the weak link.
- You want to understand what `devpilot run` is actually enforcing when it executes a card.

For the underlying theory, see the `devpilot-harness-engineering` skill. This doc is the concrete instantiation for *this* repo.

---

## The Layers

```
┌─────────────────────────────────────────────────────────────┐
│  GUIDES (feedforward — read before acting)                  │
│  CLAUDE.md  →  skills/  →  docs/plans/  →  GOLDEN_PRINCIPLES │
├─────────────────────────────────────────────────────────────┤
│  SENSORS (feedback — catch drift after acting)              │
│  gofmt  →  golangci-lint  →  go test  →  CI  →  devpilot review│
├─────────────────────────────────────────────────────────────┤
│  EXECUTION                                                  │
│  devpilot run → claude -p → branch → PR → auto-merge        │
└─────────────────────────────────────────────────────────────┘
```

## Guides (what the agent reads)

| Artifact | Scope | Loaded when | Size today |
|---|---|---|---|
| `CLAUDE.md` | Repo-wide always-on context | Every prompt | 172 lines (over target; see gaps) |
| `docs/cli-reference.md` | CLI surface detail | On demand | split from CLAUDE.md |
| `.claude/skills/*` | On-demand workflows | Skill description triggers | varies |
| `skills/*` | Distributable skill catalog | Installed via `devpilot skill add` | varies |
| `GOLDEN_PRINCIPLES.md` | Opinionated taste calls | Read before architecture-touching PRs | new |
| `docs/plans/*-design.md` | Per-feature intent | Referenced from a task | varies |
| `docs/rejected/*` | Deferred ideas | Read by `devpilot-pm` skill | varies |

## Sensors (what catches drift)

| Sensor | Category | Execution | Where it runs |
|---|---|---|---|
| `gofmt` / `goimports` | Maintainability | Computational | pre-commit (manual), CI |
| `golangci-lint` | Maintainability + some architecture | Computational | `make lint`, CI (`test.yml`) |
| `go test ./...` | Behavior | Computational | `make test`, CI |
| `skills/index.json` drift | Maintainability | Computational (manual today) | — gap — |
| `devpilot review <pr-url>` | All three | Inferential | Manually or auto-invoked by runner |
| Runner's auto code-review pass | All three | Inferential | `devpilot run` (can disable with `--review-timeout 0`) |
| Import-boundary enforcement | Architecture fitness | — | gap — |
| Behavior harness beyond unit tests | Behavior | — | gap — |

## Execution (what runs the work)

The **task runner** (`devpilot run`) is itself the enforcement loop:

1. Polls Trello/GitHub for a Ready card.
2. Branches from `main`, hands the card to `claude -p`.
3. Streams events into the TUI (see `internal/taskrunner/eventbridge.go`).
4. Pushes a PR, optionally runs the review pass (`devpilot review` under the hood).
5. Auto-merges on green CI — CI is the sensor gate.
6. Moves the card to Done / Failed.

This pipeline is a **depth-first block**: one card = one context window = one PR. The harness-engineering skill's block-sizing rule applies directly — cards that don't fit this shape are the problem, not the runner.

---

## Today's Gaps

Ranked by blast radius.

1. **No background GC refactor loop.** `GOLDEN_PRINCIPLES.md` now exists, but nothing sweeps the repo for deviations. Candidate: a scheduled `devpilot run` that consumes a "golden-principles-sweep" card nightly. *Add when:* drift between audits is observable.
2. **No pre-commit hook.** Every contributor relies on memory to run `make lint`. `lefthook` or a one-file `.githooks/pre-commit` would shift the linter left. *Add when:* an agent PR ships with obvious lint failures that CI catches.
3. **Behavior harness is thin.** Unit tests cover packages well, but there's no integration test for the runner end-to-end (polls a fake board, executes a fake `claude -p`, asserts a PR is opened). *Add when:* a runner regression lands that unit tests couldn't catch.
4. **No import-boundary enforcement.** Principles 1–3 in `GOLDEN_PRINCIPLES.md` are guides only; nothing prevents `internal/trello` from importing `internal/taskrunner`. Candidate: a `go test` that parses import graphs, or a `depguard` golangci rule.
5. **No `skills/index.json` drift check.** Adding a file under `skills/` today requires manual index updates. A simple test that cross-checks directory contents vs `index.json` would close this.
6. **CLAUDE.md still ~172 lines after split.** Revisit: candidates for further extraction are the "Architecture" subsections (TUI Dashboard, Event System), which are background reference rather than every-prompt context.
7. **MCP tool audit.** This project has Gmail, Google Calendar, Google Drive, Notion, Pencil MCP servers loaded. Most are unused by day-to-day development. Scoping them per task would reclaim context budget.

---

## Where Things Go

Use this table to place a new guardrail:

| I want to… | Put it in |
|---|---|
| Teach the agent a project-wide convention | `CLAUDE.md` (only if it's short and load-bearing on every prompt) |
| Teach the agent a task-specific workflow | A new skill under `skills/devpilot-<name>/` + register in `index.json` |
| Prevent a recurring mechanical mistake | `golangci.yml` custom rule or `make lint` step |
| Encode an opinionated taste call | `GOLDEN_PRINCIPLES.md` |
| Record why a feature exists or was rejected | `docs/plans/…-design.md` or `docs/rejected/…` |
| Make the runner do something after every PR | `internal/taskrunner/` hook + event type |
| Give agents a new external capability | CLI command under `internal/<domain>/commands.go`, **not** a new MCP server unless the CLI is inadequate |

---

## Maintenance

Every time a code review comment is repeated across three or more PRs, ask:

- **Mechanical?** Promote to a linter rule or test.
- **Judgement call?** Promote to `GOLDEN_PRINCIPLES.md`.
- **Domain-specific workflow?** Write a new skill.
- **Always-on context?** Consider `CLAUDE.md`, but only if it will *never* be stale-ignored.

When a gap from the list above starts actually biting, close it. Don't close them speculatively — that's exactly the anti-pattern the harness-engineering skill warns against.
