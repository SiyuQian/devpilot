# Plans

Active work, newest first. Each entry: status, one-line goal, link.

- 2026-04-24 — Remove the runner and sibling AI-wrapping commands; reposition DevPilot as a skill catalog + narrow Go helpers. ([design](docs/exec-plans/active/2026-04-24-remove-runner-design.md), [plan](docs/exec-plans/active/2026-04-24-remove-runner-plan.md))

## Recently completed

_Historical design+plan pairs from before this index existed live under `docs/plans/` (dated YYYY-MM-DD, paired `*-design.md` / `*-plan.md`). They are not migrated and not maintained — treat as archive._

## Tech debt tracker

See `docs/exec-plans/tech-debt-tracker.md`.

## Format

- New plans go in `docs/exec-plans/active/YYYY-MM-DD-<slug>.md`; add one line here.
- On completion, move the file to `docs/exec-plans/completed/` and move the line to "Recently completed".
- Per-plan file shape: see `.claude/skills/devpilot-harness-engineering/references/plans.md`.
- OpenSpec vs plain exec-plan: OpenSpec when the change touches a user-facing spec; plain exec-plan for infra/tooling/refactors.
