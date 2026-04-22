---
name: devpilot-harness-engineering
description: Use when setting up a repository for autonomous coding agents, adding guardrails, context files, or automation so agents ship reliably without constant review. Triggers on "make this repo agent-friendly", "agents keep drifting", "set up AGENTS.md / skills / sub-agents", "harness engineering", architectural drift with agent-authored code, or retrofitting guardrails after output quality decayed.
---

# Harness Engineering

## Overview

An **agent harness** is everything around a coding agent *except the model itself*: the guides that steer it before it acts, the sensors that catch drift after it acts, the context it reads, the sub-agents it delegates to, and the automation that maintains code quality in the background.

**Core principle:** When agents write most of the code, the engineering team's primary job shifts from *writing code* to *building the harness that lets agents write good code*. Constraints you'd normally postpone until hundreds of engineers are onboard become day-one prerequisites.

**Operating definition (Mitchell Hashimoto):** every time the agent makes a mistake, engineer the harness so it cannot make that mistake again. The harness is never "done" — it grows from observed failures, not speculation.

**Synthesized from** OpenAI's "Harness engineering: leveraging Codex in an agent-first world" (a ~5-month experiment by a team growing from 3 to 7 engineers, producing roughly 1M lines of code and ~1,500 merged PRs with no manually-written code), Martin Fowler's harness-engineering article, and HumanLayer's "Skill Issue" writeup.

## When to Use

**Use when:**
- Starting a repo where agents will author most code
- Agent output quality is decaying (style drift, inconsistent patterns, duplicated utilities)
- You're spending more time reviewing agent PRs than the agents spent writing them
- Deciding where to invest: AGENTS.md, skills, sub-agents, linters, fitness functions, background refactor jobs
- Retrofitting guardrails onto a codebase that now has agents loose in it

**Don't use when:**
- You only need style rules for a specific language → use `devpilot-google-go-style` or similar
- You need PR review mechanics → use `devpilot-pr-review`
- The task is a one-off script, not an ongoing codebase

## Core Model

A harness has **two control types** applied across **three regulation categories**:

| Control | Direction | Examples |
|---------|-----------|----------|
| **Guide** | Feedforward — prevents bad output before it happens | AGENTS.md, skills, architecture docs, example-driven style guides |
| **Sensor** | Feedback — detects bad output after it happens | Linters, type checkers, tests, fitness functions, review agents |

| Category | What it regulates | Maturity |
|----------|-------------------|----------|
| **Maintainability** | Style, structure, consistency | Most mature — reuse existing tooling |
| **Architecture fitness** | Performance, module boundaries, dependencies | Medium — needs fitness functions |
| **Behavior** | Functional correctness | Least mature — tests + manual |

Controls are either **computational** (deterministic, ms–s, cheap) or **inferential** (LLM-based, slower, richer feedback). Shift cheap ones left (pre-commit), expensive ones right (CI, background jobs).

## Quick Reference — Where to Invest First

| Symptom | Add this |
|---------|----------|
| Agent doesn't know project conventions | `AGENTS.md` / `CLAUDE.md` at repo root (keep it short — HumanLayer recommends under ~60 lines; treat ~100 as a hard split point) |
| Agent repeats the same task awkwardly | A skill with SKILL.md + references/ |
| Agent floods its own context with noise | Delegate to sub-agents; let them cite, not dump |
| Style keeps drifting | Custom linter rules + pre-commit hook |
| Module boundaries get violated | ArchUnit-style structural tests |
| Quality decays PR-over-PR | Encode "golden principles" (taste/structure, *not* style rules — those belong in linters) + background GC refactor agent |
| Agent "finishes" but build is broken | Post-tool hooks that run build/typecheck and feed errors back |
| Too many MCP tools; context blown | Prune tool descriptions; scope tools per task. Prefer CLIs the model already knows from training over custom MCP wrappers (HumanLayer) |

## Day-One Order of Operations

Work top-to-bottom; each step makes the next cheaper.

1. **Architectural constraints** — decide module boundaries, error model, dependency surface before the agent adds a tenth variant of each. See `references/architectural-constraints.md`.
2. **Agent context** — write a tight AGENTS.md and identify which specialized knowledge belongs in skills vs sub-agents vs tool descriptions. See `references/agent-context.md`.
3. **Guides + sensors** — pair each important rule with a mechanical check (linter, test, fitness function). See `references/guides-and-sensors.md`.
4. **Depth-first decomposition** — size the unit of agent work so one block = one context window = one reviewable PR. See `references/depth-first-decomposition.md`.
5. **Golden principles + GC loop** — encode taste and run a background refactor agent to counter compounding drift. See `references/golden-principles.md`.

## Red Flags

- Agent PRs are large and require deep human review → decomposition missing, sensors too weak
- Same mistake appears across unrelated PRs → guide missing, promote the lesson into AGENTS.md / skill
- AGENTS.md has grown past ~100 lines and is full of "if X then Y" → move into skills with progressive disclosure
- You keep adding MCP tools "just in case" → tool descriptions are burning context on every turn
- Background quality is dropping but no single PR caused it → add a drift-detection / golden-principles sweep

## Common Mistakes

- **Treating AGENTS.md as a dumping ground.** It's injected into every prompt. Keep it tight; push detail into on-demand skills.
- **Only feedforward, no feedback.** Docs alone don't catch the agent's 1% failures. Pair every important rule with a sensor.
- **Only feedback, no feedforward.** Letting the agent fail and retry burns tokens and time. Prevent what you can cheaply.
- **Human-readable sensor output.** Linter messages should include correction instructions the *agent* can act on, not just human-targeted prose.
- **Ignoring compounding drift.** Without a GC loop, each PR adds a little mess; in a month the repo is unrecognizable.
- **Speculative configuration.** Installing skills, MCP servers, or hooks "just in case" before observing a real failure bloats context and hides real signal. Build the harness *from observed mistakes* (Hashimoto), not from imagined ones.
- **Ignoring context rot.** Long sessions degrade even before hitting hard limits. Compaction, sub-agent delegation, and aggressive pruning are maintenance, not optimization.

## The Bottom Line

Speed at agent scale comes from constraints, not freedom. A good harness makes the boring right thing easy and the drifty wrong thing impossible — so humans spend their attention on the decisions only humans can make.
