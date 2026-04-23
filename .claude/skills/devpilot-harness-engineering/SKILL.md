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

1. **Architectural constraints** — decide module boundaries, error model, dependency surface before the agent adds a tenth variant of each. See `references/architecture.md`.
2. **Agent context** — write a tight AGENTS.md and identify which specialized knowledge belongs in skills vs sub-agents vs tool descriptions. See `references/agents.md`.
3. **Guides + sensors** — pair each important rule with a mechanical check (linter, test, fitness function). See `references/guides-and-sensors.md`.
4. **Depth-first decomposition + plan index** — size each block so one = one context window = one reviewable PR, and track what's in flight in a plans index. See `references/depth-first-decomposition.md` for sizing, `references/plans.md` for `PLANS.md` + `docs/exec-plans/` format and the OpenSpec-vs-exec-plan decision.
5. **Golden principles + GC loop** — encode taste and run a background refactor agent to counter compounding drift. See `references/golden-principles.md`.

## Recommended Repo Layout

A harness-ready repo typically settles into this shape. Root-level `.md` files are **always-on or frequently-loaded** context; `docs/` subtrees are **on-demand** reference the agent reads when a task pulls it in.

```
AGENTS.md              # always-on preamble (see references/agents.md)
ARCHITECTURE.md        # on-demand map + invariants (see references/architecture.md)
PLANS.md               # index of active exec plans (see references/plans.md)
DESIGN.md              # how we design — taste for non-obvious calls
FRONTEND.md            # UI conventions (only if the repo has a frontend)
PRODUCT_SENSE.md       # product bar, user-visible quality rules
QUALITY_SCORE.md       # how we grade PRs / output
RELIABILITY.md         # error budgets, on-call, failure-mode posture
SECURITY.md            # threat model + non-negotiables

docs/
├── design-docs/       # durable design rationale; paired with exec-plans
│   ├── index.md
│   └── <topic>.md
├── exec-plans/
│   ├── active/        # one file per in-flight plan
│   ├── completed/     # archived when done; preserves decision trail
│   └── tech-debt-tracker.md
├── generated/         # machine-generated refs (db-schema, API shape) — never hand-edit
├── product-specs/
│   ├── index.md
│   └── <feature>.md
└── references/        # third-party *-llms.txt dumps (design-system, uv, nixpacks, …)
```

**What each root file answers:**

| File | Question the agent needs answered |
|---|---|
| `AGENTS.md` | What should I remember on *every* task in this repo? |
| `ARCHITECTURE.md` | What's the shape, and what invariants must I not break? |
| `DESIGN.md` | When the code could go two ways, which does this team prefer and why? |
| `FRONTEND.md` | What UI primitives / patterns exist; what's banned? |
| `PLANS.md` | What are we currently trying to ship, and what's next? |
| `PRODUCT_SENSE.md` | What makes a change user-worthy here? |
| `QUALITY_SCORE.md` | How will this PR be judged? |
| `RELIABILITY.md` | What happens when this breaks in prod; what's my budget? |
| `SECURITY.md` | What must I never do, regardless of how convenient? |

**Per-file authoring guides (in `references/`):**
Available now: `agents.md`, `architecture.md`, `plans.md`.

Not yet written: `design.md`, `frontend.md`, `product-sense.md`, `quality-score.md`, `reliability.md`, `security.md` — **add when you've observed agents getting that doc wrong**. Following Hashimoto's rule: the harness grows from observed failures, not speculation.

Not every repo needs every file. `FRONTEND.md` is pointless without a frontend; `RELIABILITY.md` is noise for a CLI tool. But `AGENTS.md` and `ARCHITECTURE.md` are mandatory for any repo with more than trivial agent activity.

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
