# Depth-First Decomposition

Use this when sizing a task before handing it to an agent, or when reviewing a plan's task list for "too wide, too shallow."

## Block vs plan — the hierarchy

- **Block** = one atomic task = one agent session = one PR. This document is about blocks.
- **Exec plan** = the envelope holding multiple blocks for one feature; has Goal / Non-goals / Design / Tasks / Decisions log / Open questions. Each task inside an exec plan should itself pass the block self-check below.
- **Plans index** (`PLANS.md`) = one line per active exec plan.

So: block shape (Scope/Interfaces/Acceptance/Links/DoD, below) is **per task inside an exec plan**, not an alternative to the exec-plan template.

## Block size self-check

Run this on the candidate block. Any "no" means resize before starting.

- [ ] Fits in one agent session without hitting context limits
- [ ] Produces exactly one reviewable artifact (PR, migration, doc)
- [ ] Has a definition of done a sensor can verify (tests pass, lint clean, fitness green)
- [ ] Does not require reading more than ~10 files to start
- [ ] Independent of any block currently in flight (no shared files, no shared state)
- [ ] Leaves a *merged, functional* slice — not a stub other blocks depend on

## Rule-of-thumb ceilings

Heuristics for "does this still fit in one session." Crossing one is a yellow flag; crossing two is a red flag — split.

| Signal | Ceiling |
|---|---|
| Diff size | ~400 lines added + changed |
| Files touched | ~8 files |
| Agent wall time to finish | ~30 minutes |
| New concepts the block introduces | 1 (maybe 2) |
| External systems a block talks to | 1 |

These are not hard rules; a block that exceeds them may still be right-sized if it's mostly generated code or test fixtures. But "exceeds and feels fine" should be justified in the block's Scope, not assumed.

## Block size by example

| Too small (just do it manually) | Right-sized | Too large (split) |
|---|---|---|
| Rename one variable | Add endpoint X with handler, tests, migration | Build the billing system |
| Fix one lint error | Refactor module Y to use new error wrapper | Rearchitect the data layer |
| Update one doc link | Add a new skill with one example + TOC | "Harden the platform" |

## Depth-first vs breadth-first — which to pick

Default to **depth-first**. Pick breadth-first only when the structural-independence test below passes.

**Depth-first (default):** finish one vertical slice end-to-end (design → code → tests → merge) before starting the next.

**Breadth-first is only safe when all three hold:**
- [ ] Blocks touch disjoint files and directories
- [ ] Blocks share no in-memory or on-disk state
- [ ] Block N does not depend on block N-1's output

If any fails → serialize.

## Required block shape

A block handed to an agent must include:

1. **Scope** — one paragraph, with explicit non-goals.
2. **Interfaces** — function signatures / schema / endpoint shape, not prose description.
3. **Acceptance criteria** — concrete, sensor-checkable (e.g., "endpoint returns 200 with `{id,status}` for valid input; 400 otherwise; test coverage >80% on new code").
4. **Links** (not copies) — to the relevant skills, architecture invariants, and example prior PRs.
5. **Definition of done** — the exact commands that must pass (`make test && make lint && <fitness-test>`).

If any of 1–5 is missing, the block is not ready to hand over; refine it first.

## Red flags in a task list

Scan `PLANS.md` or an exec plan for these — any hit means resize.

- Tasks phrased as verbs without objects ("refactor", "cleanup", "investigate")
- Tasks that say "and" more than once (usually two tasks glued together)
- Tasks whose definition of done is "looks good" or "it works"
- A wide list where every task touches the same module (conflict risk → serialize)
- Tasks that finish by leaving the codebase non-compilable (stub-and-follow-up pattern)

## DevPilot mapping

In this repo:
- Trello card / GitHub Issue = one block
- P0 / P1 / P2 labels = depth-first ordering (finish P0 before starting P1)
- Branch + PR per block = clean handoff artifact
- Auto-merge on green CI = sensor-gated completion
- `openspec` change = pre-approved block with design + tasks

If a card feels too big for the runner, it's the block that's wrong, not the runner.
