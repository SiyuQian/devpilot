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
