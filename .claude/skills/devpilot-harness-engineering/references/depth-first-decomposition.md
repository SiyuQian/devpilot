# Depth-First Decomposition

## Principle

When most code is agent-authored, the human's most valuable act is **breaking the goal down into blocks an agent can finish cleanly**. OpenAI describes their approach as depth-first: larger goals are decomposed into small building blocks (design, code, review, test), and each finished block becomes the scaffolding that unlocks the next.

The unit of agent work should be a block the agent can:

1. Load enough context to do well
2. Finish in one session without hitting context limits
3. Verify via sensors (tests / build / lint) before handoff
4. Hand off with a clear artifact (PR, function, schema)

## What Counts as an "Agent-Sized" Block

| Too small | Right-sized | Too large |
|---|---|---|
| "Rename this var" | "Add endpoint X with handler, tests, and migration" | "Build the billing system" |
| "Fix one lint error" | "Refactor module Y to use the new error wrapper" | "Rearchitect the data layer" |

Too small: overhead dominates; humans should just do it.
Too large: agent loses the plot, context fills with noise, PR is unreviewable.

## Depth-First, Not Breadth-First

Breadth-first (bad): sketch 20 TODOs across the whole system, launch 20 agents, get 20 half-baked PRs.

Depth-first (good): fully finish one vertical slice — design doc → code → tests → review → merge — and use the merged slice as scaffolding the next slice can lean on.

Why depth-first wins with agents:
- Each finished slice becomes **context** the next agent can read
- Integration problems surface immediately, not at the end
- Humans review one coherent thing, not 20 overlapping stubs
- Sensors you build for slice 1 catch regressions in slice 2

## The Block Shape

A well-shaped block has:

- **Design note** (human or agent, reviewed by human): scope, interfaces, non-goals
- **Task statement** with acceptance criteria
- **Links to the relevant skills / references** (not copies)
- **Definition of done** that a sensor can check (tests pass, lint clean, fitness test green)

## Parallelism, Carefully

Parallel agents are safe when blocks are **genuinely independent**: no shared files, no shared state, no sequential dependency. They are unsafe when blocks touch the same module — merge conflicts, inconsistent patterns, duplicated helpers.

Default: depth-first serial. Parallelize only where independence is structural.

## In DevPilot Terms

This maps directly onto the DevPilot task-runner model:

- A Trello card / GitHub Issue = one block
- P0 / P1 / P2 labels = depth-first ordering
- Branch + PR per block = clean handoff artifact
- Auto-merge on green CI = sensor-gated completion
- `openspec` change = pre-approved block shape with design + tasks

If a card feels too big to hand to the runner, it's not a runner problem — the block isn't sized right yet.

## Anti-Patterns

- **"Just have the agent figure it out."** Decomposition is the human's leverage point; don't outsource it.
- **Breadth-first tickets.** 20 stubs is 20 problems.
- **Blocks without a definition of done.** The agent and the sensors disagree on "finished."
- **Blocks that require reading 30 files to start.** Either the context isn't organized or the block is too big.
