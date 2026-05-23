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

0. **Coverage assertion (Agent B).** Verify Agent B returned a `coverage` block covering every [REQUIRED CHECK] item in `references/checklist.md` §Security AND §Performance, with allowed values (`checked, no_evidence` | `finding_raised` | `not_applicable (<reason>)`). Missing keys, blank reasons, or items returned with unrecognized values trigger a single re-dispatch of Agent B with the exact missing items spelled out in the brief. If the second return is still incomplete, record `Security/Perf scan: partial (n/m items covered, missing: <list>)` in the body's Unknown-Unknowns Sweep block and downgrade the review event to at most `COMMENT` (never `APPROVE`) — an incomplete safety scan is not an approval.
1. **Collect** all findings from agents A–E into one list.
1.5. **Inject graph-derived findings** for missing tests on changed public surface. See "Graph-injected findings" below. Skip if graph fell back.
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

## Graph-injected findings (step 1.5)

The fanout relies on Agent A's judgment to surface missing tests on changed public surface. In practice agents rationalize this away (`"too defensive to need a test"`, `"author justified skipping it"`, `"would be a nag"`). The graph already knows which changed public symbols lack a direct test; the main session injects findings unconditionally from the payload so judgment can only *upgrade* the finding, not silence it.

**Inject one finding for every `changed_symbols[]` entry that satisfies ALL of:**

- `is_exported == true`
- `change_type` ∈ {`"modified"`, `"added"`}
- `tests.has_tests == false`
- `kind` ∈ {`"function"`, `"method"`}
- The symbol's diff is **not trivial** (see "Trivial diff" below).

**Default shape:**

```
title: Missing test for changed public surface — <symbol_id>
severity: Should-fix
confidence: 85           # graph corroborated; floor in graph.md
anchor: <path>:<signature line of the symbol at head SHA>
agent: "graph"
behavior: "<symbol> is exported, modified by this PR, and has no direct test symbol in the graph."
why: "Untested public surface changes are how silent regressions ship. Author may have justified skipping a test in the PR body — surface the gap explicitly so the decision is reviewable."
fix: "Add a direct test for <symbol> covering the changed behavior, or reply with the reason a test is impractical (then resolve)."
```

**Severity escalation:**

- `risk_factors` contains `hub` → `Severity: Blocking` (a hub semantic change without a test pages everyone downstream).
- `risk_factors` contains `interface_change` → `Severity: Blocking`.

**Trivial diff** (skip injection — these would be pure noise):

- The symbol's hunk is ≤ 3 lines AND every line is a comment or whitespace.
- The symbol's hunk is a pure rename / signature relocation (name change, no body change).
- The symbol's body becomes a single-line forward (`return Other(args...)` and nothing else).

**Not trivial — DO inject even though it feels minor:**

- Defensive nil-guard / error-guard mirroring an existing pattern (5-line `if x == nil { continue }` blocks). The author asserting "this is too defensive to test" is exactly the case the injection is designed to overcome. Author can resolve the inline with "untestable, see PR body" — that's healthy. Silently dropping the finding is not.
- Refactors that change control flow even by one branch.
- Anything the author flagged in the PR description as "no test added because ___" — the justification belongs in the resolution thread, not in the reviewer's head.

**Sibling-test transitivity is not coverage.** "The mirror branch / sibling function has a test, so this one is covered by analogy" is a rationalization, not a fact. Each symbol's `tests.has_tests` is independent. If two branches share a contract, the right answer is usually to factor out a helper and test the helper — that refactor *is* the fix.

**Dedupe with Agent A:** if Agent A returned a finding for the same `(symbol, "missing test")` defect, drop the graph-injected one and keep Agent A's (Agent A's `fix` is concrete; the injected default is generic).

**Anchor:** the symbol's declaration line at head SHA. For Go methods, the line of `func (recv T) Name(...)`. The main session resolves the line by reading the file at head SHA — do not guess.

## What to do when the fanout returns nothing

- Verify the gate in `eligibility.md` didn't already explain why (generated-only, automation, etc.).
- Re-read the diff yourself for a one-pass sanity check. A clean fanout on a non-trivial PR is a yellow flag — either the PR is genuinely clean, or the briefs missed something.
- If still clean: post `event: APPROVE` with the body's "Strengths" filled in, sweep summary present, finding counts all zero.
