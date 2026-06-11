# Grading rubric — harness-engineering output quality

Read this alongside the skill-creator grader contract (`agents/grader.md`). That
file defines the mechanics — read the transcript, cite evidence, write
`grading.json` with `text` / `passed` / `evidence`, no partial credit. This file
adds the rules specific to grading **harness advice**, where the failure mode is
not "wrong file" but "generic advice that looks right."

## What these evals measure

Whether the advice the skill produces is *better than baseline Claude's default
knowledge* about agent harnesses. Every scenario is run twice — once with the
skill loaded, once without (`baseline`) — against the same prompt. The signal is
the **pass-rate delta**, not the absolute pass rate. An assertion both arms pass
tells us nothing about the skill.

## The non-discrimination rule (most important)

Baseline Claude already knows the obvious moves: "write an AGENTS.md", "add a
linter", "keep PRs small". Do **not** pass an assertion just because the output
states the generic version of the right idea. Pass it only when the output
demonstrates the skill's *distinctive* framing for that scenario:

- **guide↔sensor pairing** — not "document the rule better" but "add a mechanical
  check, and its output must tell the *agent* how to fix the violation."
- **anti-speculation (Hashimoto)** — not "set up good docs" but "only AGENTS.md +
  ARCHITECTURE.md now; add the rest *when you observe the agent getting that
  wrong*."
- **GC / compounding drift** — not "review more carefully" but "no single PR is
  the culprit; add a background refactor pass / golden-principles sweep."
- **depth-first sizing** — not "ask for smaller PRs" but "one block = one context
  window = one reviewable PR, tracked in a plan index."
- **progressive disclosure** — not "trim the docs" but "AGENTS.md is injected on
  every turn; move the conditional detail into on-demand skills."
- **tool/context bloat** — not "use fewer tools" but "tool descriptions burn
  context every turn; scope per task, prefer CLIs the model already knows."

When in doubt, ask: *would the baseline arm, with no skill, plausibly produce
this same sentence?* If yes, the assertion is non-discriminating — fail it for
the with-skill arm too unless the output goes beyond the generic version, and
flag it under `eval_feedback`.

## Negative assertions

Some assertions are phrased as "does NOT do X" (e.g. "does not create all six
root docs upfront"; "does not just reword the doc"; "does not say add more
tools"). For these:

- **PASS** = the output genuinely avoids the bad move *and* the avoidance looks
  deliberate (it recommends the better alternative), not merely an omission
  because the output was thin.
- **FAIL** = the output does the bad thing, or hedges toward it ("you could
  create those scaffolds now and fill them later" fails the anti-speculation
  negative).

Negative assertions are where baseline most often diverges from the skill — weight
them when reading the delta.

## Scoring guidance

- No partial credit per assertion (grader contract). But a near-miss that states
  the distinctive idea weakly should fail, with the gap quoted in `evidence`.
- Reward *specificity tied to the scenario's planted defect* over breadth. An
  answer that correctly diagnoses this repo's one real problem beats a generic
  checklist that happens to contain the right item.
- For the two fixture scenarios, the output must show it actually inspected the
  files (cites the line count, names the violating files) — advice that would be
  identical without reading the repo is a weaker pass at best.

## Critiquing the evals

Per the grader contract's Step 6, flag non-discriminating assertions you notice
(both arms pass) and any distinctive idea the scenario *should* test but no
assertion covers. These evals are new; that feedback is the point.
