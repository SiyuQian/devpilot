# Confidence Rubric, Filtering, and Merge

The five fanout agents return findings with `confidence: 0–100`. This file is the rubric they score against and the procedure the main session uses to filter, dedupe, and merge their output.

## Confidence rubric (0–100)

Confidence measures **how sure you are the finding is real**, independent of how bad it would be (that's Severity).

| Score | Meaning |
|---|---|
| 100 | Literal-string evidence. The defect is visible as a single phrase in the diff. |
| 85–95 | Traced through code on this branch. The reviewer opened the relevant files and reproduced the path. |
| 70–84 | Strong pattern match. The defect is clear from the code, but the reviewer didn't trace every caller / test. |
| 50–69 | Plausible but unverified. A caller or test would confirm, but the reviewer couldn't locate it. |
| 25–49 | Speculation grounded in general knowledge, not in this codebase. |
| 0–24 | Not really a finding. Should not have been surfaced. |

## Default threshold: 70

Drop every finding with `confidence < 70` before drafting.

**Why 70:**
- Below 70 the reviewer has not opened the code that would confirm. Posting it lowers signal.
- 70 is below "traced on this branch" (85+) but above pure pattern-match speculation. It keeps high-pattern-match findings worth a sentence.

**When to override:**
- The user explicitly asked for "everything, including maybes" → threshold = 50.
- The user asked for "only certain" → threshold = 85.
- Otherwise: 70. Don't bargain with yourself.

## Severity vs. Confidence are orthogonal

A high-severity bug you are moderately sure about is `Severity: Blocking, Confidence: 75`. Low confidence never automatically demotes severity. Severity describes *impact if true*; confidence describes *probability of being true*.

| Severity | Description |
|---|---|
| Blocking | Would cause data loss, security regression, outage, or silently wrong behavior in production. |
| Should-fix | Real bug on a reachable code path, missing test for a risky path, unhandled pitfall. |
| Consider | Design or maintainability feedback worth the author's attention. |
| Nit | Style, naming, wording. |

## Inline-comment confidence label

The inline comment template (`template.md`) shows `Confidence: high | medium | low`, not the raw 0–100 score. Mapping:

| 0–100 | Label |
|---|---|
| 85–100 | high |
| 70–84 | medium |
| < 70 | (dropped — does not render) |

The numeric score is internal to the fanout pipeline. The label is what the PR author sees.

## Graph reconciliation (before filtering)

When `references/graph.md` produced a preflight payload (mode=`built`), each finding is reconciled against it before the threshold filter runs:

- **Corroborated** — the finding cites a symbol whose `changed_symbols[].callers` / `risk_factors` match the defect (e.g. "this exported function's caller in package X doesn't update for the new contract" and the named caller is in `callers.sample`). Confidence floor raised to 85, cap stays at 95 unless the diff itself shows literal-string evidence.
- **Contradicted** — the finding asserts a caller relationship or hub status the graph denies (named caller absent from `callers.sample`; "this is a hub" with `in_hub:false`; "no tests" with `tests.has_tests:true`). Confidence capped at 50.
- **Unsupported** — finding sits outside graph coverage (no symbol match, or `mode != "built"`). Original score stands; do not boost or penalize.

A finding both corroborated on one dimension and contradicted on another takes the more conservative outcome (cap at 50).

## Merge procedure (after the fanout returns)

1. **Collect** all findings from agents A–E into one list.
2. **Reconcile** against `GRAPH_PREFLIGHT` per the section above (skip if graph fell back).
3. **Filter:**
   - Drop `confidence < 70` (or the user-overridden threshold).
   - Drop anything matching the false-positive list in `eligibility.md`.
4. **Dedupe:**
   - Same `(path, line)` and same defect class from multiple agents → one finding. Keep the highest confidence; merge the fixes if they differ.
   - **Same defect across multiple lines or files** (e.g. four files all log the same secret) → **one consolidated inline comment** anchored to the worst/most-representative occurrence. List the other `path:line` locations inside the comment body as a short bullet list (`Same defect also at: a.go:42, b.go:91, c.go:30`). Also note the recurrence count in the body sweep summary under "Blast radius". Do NOT post one near-identical comment per file — that is the noise inline-first is meant to avoid.
5. **Anchor:**
   - Every surviving finding needs `(path, line, side)`. Cross-cutting findings (e.g. "this PR has no tests") anchor to the most representative line — the new function's signature, the first new line of the changed file. The comment body MUST say so.
6. **Count** by severity. The counts go in the body.
7. **Derive the review event** (`REQUEST_CHANGES` / `COMMENT` / `APPROVE`) from the highest-severity surviving finding. See `posting.md` → "Event mapping".

## What to do when the fanout returns nothing

- Verify the gate in `eligibility.md` didn't already explain why (generated-only, automation, etc.).
- Re-read the diff yourself for a one-pass sanity check. A clean fanout on a non-trivial PR is a yellow flag — either the PR is genuinely clean, or the briefs missed something.
- If still clean: post `event: APPROVE` with the body's "Strengths" filled in, sweep summary present, finding counts all zero.
