# Evals — devpilot-harness-engineering

An **output-quality** eval suite: it measures whether the advice this skill
produces beats baseline Claude's default knowledge about agent harnesses. It does
**not** measure triggering accuracy.

## What's here

- `evals.json` — 6 scenarios, each anchored to a *distinctive* idea of the skill
  (guide↔sensor pairing, anti-speculation, GC/drift loop, depth-first sizing,
  AGENTS.md progressive disclosure, tool/context bloat). Each carries 3–5
  `expectations`, including ≥1 negative assertion that catches baseline
  over-eagerness.
- `rubric.md` — grading guidance layered on top of the skill-creator grader
  contract; its core is the **non-discrimination rule** (don't credit advice the
  baseline arm would also give).
- `fixtures/bloated-agents-md/` — a repo whose AGENTS.md is ~160 lines of
  if-X-then-Y rules (scenario 5).
- `fixtures/rule-without-sensor/` — a repo with a documented ban and no
  mechanical check enforcing it; two source files violate it (scenario 1).

`evals.json` follows the skill-creator schema (`references/schemas.md` in the
skill-creator skill).

## How to run it (later session)

This suite is **authored, not run**. To run it, drive the `skill-creator` skill's
eval loop ("Running and evaluating test cases"). In short:

1. **Stage fixtures.** Scenarios 1 and 5 reference a fixture *directory* in their
   `files`. Copy that directory into the executor's working area so the agent can
   inspect it, and phrase the prompt so the agent knows where the repo is.
2. **Spawn paired runs in the same turn.** For every scenario, launch two
   subagents at once — one **with** the skill loaded, one **baseline** (no skill)
   — on the same prompt. Don't run all with-skill first.
3. **Grade** each run with a grader subagent that reads `rubric.md` plus the
   skill-creator grader contract (`agents/grader.md`); write `grading.json`
   (`text` / `passed` / `evidence`) per run.
4. **Aggregate** into `benchmark.json` via the skill-creator aggregation script;
   the signal is the **pass-rate delta** (with_skill − without_skill), not the
   absolute rate.
5. **Analyst pass + viewer.** Flag non-discriminating assertions (both arms pass)
   and open the eval viewer for the qualitative read.

Run outputs belong in a sibling workspace
(`devpilot-harness-engineering-workspace/`, organized by `iteration-N/eval-<id>/`),
**not** committed here.

## Reading the results

Because baseline Claude already knows generic harness advice, expect some
assertions to be near-ties. The scenarios were chosen so the *distinctive* moves
discriminate — pay closest attention to the negative assertions and to the two
fixture scenarios, where the skill must actually inspect files to answer well.
