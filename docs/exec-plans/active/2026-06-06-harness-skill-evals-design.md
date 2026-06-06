# Output-quality eval suite for `devpilot-harness-engineering`

**Status:** active
**Owner:** agent (Siyu)
**Started:** 2026-06-06

## Goal

The `devpilot-harness-engineering` skill currently has no way to tell whether the
advice it produces is actually better than what baseline Claude would say. This
plan delivers a committed, reviewable **eval suite** that measures the *quality of
the harness advice the skill produces*, structured so the skill's lift over a
no-skill baseline is visible.

"Done" = the suite (scenarios, fixtures, assertions, grader rubric) lives under
`skills/devpilot-harness-engineering/evals/`, is committed, and a later session can
run it with the skill-creator harness without further authoring. This plan
**authors** the suite; it does **not** run it.

## Non-goals

- **Running** the evals (spawning with-skill/baseline subagents, grading,
  aggregating the benchmark, opening the viewer). Left to a follow-up session.
- **Triggering-accuracy** evals (does the description fire on the right prompts).
  We deliberately chose output/advice quality as the target instead.
- Fixing the `index.json` filename drift for this skill — it lists
  `references/agent-context.md` / `architectural-constraints.md`, which do not
  exist (real files: `agents.md`, `architecture.md`, …). Real adjacent bug,
  tracked here but out of scope for this plan.
- Fixing the stale "Available now: `agents.md`, `architecture.md`, `plans.md`" line
  in `SKILL.md` (omits `depth-first-decomposition.md`, `golden-principles.md`,
  `guides-and-sensors.md`, which exist). Adjacent, out of scope.
- Registering the eval files in `skills/index.json` — evals are dev-time
  artifacts, not shipped skill content.

## Design

### Input strategy

Mix of prompt-only scenarios (breadth, cheap to author) plus a small number of
fixture repos for the cases where file inspection is the entire point.

### Grading strategy

Per-scenario **assertions** (3–5 per scenario, LLM-graded via a shared
`rubric.md`, scripts where a claim is mechanically checkable) **plus baseline
lift**: each scenario is run both with the skill and without it (the skill-creator
harness), so we both diagnose specific quality and prove the skill adds value over
Claude's default knowledge.

The central risk for a guidance skill is **non-discriminating assertions** —
claims that both the with-skill and baseline arms pass, because baseline Claude
already knows generic harness advice (write an AGENTS.md, add a linter). The
analyzer pass flags these, but we pre-empt the problem by:

1. Anchoring every scenario to a *distinctive* idea of the skill — the
   guide↔sensor pairing, the anti-speculation (Hashimoto) rule, the GC/drift
   loop, depth-first one-block-one-PR sizing, AGENTS.md progressive disclosure,
   tool/context bloat — where baseline Claude tends to give generic-but-wrong
   advice.
2. Including at least one **negative** assertion per scenario that catches
   baseline over-eagerness (e.g. "does NOT create all six root docs upfront").
3. A `rubric.md` that instructs the grader to reward skill-specific framework
   moves and *not* credit generic advice both arms would give.

### Layout

```
skills/devpilot-harness-engineering/
  evals/
    evals.json            # 6 scenarios: prompt, input-files, assertions
    rubric.md             # shared grading guidance for the grader subagent
    fixtures/
      bloated-agents-md/    # ~200-line AGENTS.md of if-X-then-Y rules
      rule-without-sensor/  # CLAUDE.md bans a pattern; nothing enforces it
```

`evals.json` follows the skill-creator schema (an `evals` array; each entry has a
prompt, optional input files, and an assertions list). Run outputs go in a
sibling `*-workspace/` at run time (gitignored, not created by this plan).

### The 6 scenarios

Each anchored to a distinctive idea so lift shows; each carries 3–5 assertions
including ≥1 negative assertion.

| # | Type | Scenario | Distinctive idea | Sample discriminating assertions |
|---|------|----------|------------------|----------------------------------|
| 1 | fixture (`rule-without-sensor/`) | A CLAUDE.md rule bans a pattern but agents still do it occasionally | guide↔**sensor** pairing | recommends a *mechanical* check (linter/test/hook); says sensor output must carry agent-actionable correction text; ¬ "just reword the doc" |
| 2 | prompt | "Set me up for agents: create AGENTS.md, SECURITY.md, RELIABILITY.md, FRONTEND.md, QUALITY_SCORE.md now" | **anti-speculation** (Hashimoto) | only AGENTS.md + ARCHITECTURE.md mandatory; defers the rest to observed failures; **¬ creates all six upfront** |
| 3 | prompt | "Every PR passes review but the repo feels worse month over month; no single culprit" | **GC loop** / compounding drift | names golden-principles + background refactor agent; frames it as drift not a per-PR sensor gap; ¬ "just tighten per-PR review" |
| 4 | prompt | "Agent PRs are huge — reviewing takes longer than the agent took to write them" | **depth-first** one-block = one-PR sizing | recommends decomposition + plan index; ties block size to one context window; ¬ generic "ask for smaller PRs" only |
| 5 | fixture (`bloated-agents-md/`) | A ~200-line AGENTS.md full of conditional rules | AGENTS.md bloat / progressive disclosure | flags that it's injected on every prompt; move detail into skills; cites the ~60/100-line bar |
| 6 | prompt | "I added 12 MCP servers 'just in case' and the agent seems dumber / context blown" | tool & context bloat | prune/scope tool descriptions; prefer CLIs the model already knows; ¬ "add more tools" |

### Spec location rationale

Written as a plain exec-plan (`docs/exec-plans/active/`) per the repo's PLANS.md
convention, not the generic superpowers `docs/superpowers/specs/` path: this is
tooling that touches no user-facing spec surface, which the OpenSpec-vs-plain-plan
rule (`references/plans.md`) routes to a plain exec-plan.

## Tasks

- [ ] 1. Create `evals/fixtures/bloated-agents-md/` — a minimal repo skeleton with
      a ~200-line `AGENTS.md` of if-X-then-Y conditional rules and a couple of
      stub source dirs, enough that the right advice is "split into skills."
      Verify: `AGENTS.md` line count > 150 and contains conditional ("if … then")
      phrasing.
- [ ] 2. Create `evals/fixtures/rule-without-sensor/` — a skeleton with a
      `CLAUDE.md` that states a banned pattern in prose, and source files that
      *violate* it, with no linter/test/hook config present. Verify: the banned
      pattern appears in `CLAUDE.md` and at least one source file violates it, and
      there is no lint/test config enforcing it.
- [ ] 3. Write `evals/rubric.md` — grading guidance for the grader subagent:
      how to score each assertion, the explicit instruction to NOT credit generic
      advice both arms give, and how negative assertions are scored. Verify: file
      states the non-discrimination rule and covers negative-assertion handling.
- [ ] 4. Write `evals/evals.json` — the 6 scenarios above, each with prompt,
      input-files (fixtures for #1 and #5; none otherwise), and 3–5 named
      assertions including ≥1 negative assertion each. Verify: valid JSON, 6
      entries, every entry has ≥3 assertions and ≥1 assertion whose name marks it
      negative; assertion names are descriptive (readable in the viewer).
- [ ] 5. Add a short `README.md` (or header comment) in `evals/` pointing at the
      skill-creator run procedure (spawn with+baseline, grade via `rubric.md`,
      aggregate, view) so a later session can run without re-deriving it. Verify:
      file names the run steps and the expected `*-workspace/` output location.
- [ ] 6. Update `PLANS.md` index line and confirm nothing was added to
      `skills/index.json`. Verify: `PLANS.md` has one line linking this plan;
      `git diff skills/index.json` is empty.

## Decisions log

- 2026-06-06 — Target output/advice quality over triggering accuracy (user choice).
- 2026-06-06 — Mix prompt-only + a few fixtures for inputs (user choice).
- 2026-06-06 — Grade with per-scenario assertions + baseline lift (user choice).
- 2026-06-06 — Anchor scenarios to distinctive skill ideas so lift is visible,
  ~6 cases for iteration 1 (user choice).
- 2026-06-06 — Author-only deliverable; running the suite is a follow-up (user choice).
- 2026-06-06 — Plain exec-plan, not superpowers spec path, per repo PLANS.md
  convention (tooling, no spec surface).

## Open questions

- None blocking. The `index.json` filename drift and the stale SKILL.md
  "Available now" line are real but explicitly out of scope; raise as a separate
  cleanup if desired.
