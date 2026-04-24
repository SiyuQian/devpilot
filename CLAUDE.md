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
