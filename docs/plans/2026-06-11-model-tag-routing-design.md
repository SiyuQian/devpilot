# Model-tag routing for scan issues and issue resolution

**Date:** 2026-06-11
**Status:** Approved design, pending implementation
**Skills touched:** `skills/devpilot-scanning-repos`, `skills/devpilot-resolve-issues`

## Problem

`devpilot-resolve-issues` dispatches one implementer subagent per task but never
passes a `model` to the Agent dispatch, so every fix — from a one-line doc-drift
correction to a cross-cutting concurrency fix — runs on the main session's model.
There is no signal on the issue itself saying how capable a model the fix needs,
even though the scanner that filed it had the best context to judge.

## Decision summary

| Decision | Choice |
|---|---|
| Where the tag is assigned | Both: scanning-repos tags at filing time; resolve-issues backfills missing tags at triage |
| Tag semantics | Literal model family alias: `model:haiku` / `model:sonnet` / `model:opus` — passed verbatim as the Agent tool's `model` param, zero mapping layer |
| Assignment rubric | Estimated **fix complexity**, decoupled from severity |
| Scope of effect | Implementer subagents only; reviewers, triage, and final verify keep the main session model |

## Rubric (canonical copy lives in `devpilot-scanning-repos/references/labels.md`)

- `model:haiku` — mechanical, single-file, low-judgment change: doc drift, typo,
  adding a nil check, comment fix.
- `model:sonnet` — default tier: a normal code fix plus tests, single concern.
- `model:opus` — multi-file change, or a fix requiring careful reasoning about
  concurrency, security, or architecture.

Rules: judge the **cost of the fix, not the severity of the problem** (a critical
security hole can be a one-line haiku fix). When unsure, pick the higher tier —
wasted tokens are cheaper than a re-dispatch. No `model:fable`: opus is the top
of the taxonomy; beyond that is `need:human`.

`devpilot-resolve-issues` carries an inline three-line copy of this rubric marked
"keep in sync with devpilot-scanning-repos/references/labels.md" — the skills
install independently, so the duplication is deliberate.

## Changes to `devpilot-scanning-repos`

1. **`references/labels.md`** — add the `model:*` group (three labels, colors +
   descriptions + `gh label create` one-liners) and the rubric above. The group
   joins the step-2 reconcile scope (exact match / semantic rename / create).
2. **Scanner agent prompts** (`agents/*.md`) — each finding gains a required
   `model` field (`haiku` | `sonnet` | `opus`), assigned by the scanner per the
   rubric. The scanner just read the code; it has the most context to estimate
   fix cost.
3. **`SKILL.md` step 7 + success criteria** — every filed issue now carries
   **exactly six labels**: `scan:<category>`, subcategory, `severity:<level>`,
   `confidence:<score>`, `area:<dir>`, `model:<tier>`. `model:*` is reconciled
   upfront in step 2 (fixed three-value enum, not lazy like `area:*`).
4. **`references/issue-template.md`** — label contract updated from five to six;
   `model:*` comes from the finding's `model` field via the step-2 mapping table.
5. **`scripts/check-findings.py`** — add `model` to `REQUIRED_FIELDS` and
   validate the value against `{haiku, sonnet, opus}`.
6. **`evals/evals.json`** — update label-count assertions
   (`three_labels_per_issue` and friends) to the new six-label contract and the
   new finding field.

## Changes to `devpilot-resolve-issues`

1. **Triage backfill (new sub-step after a REAL verdict, before worktree
   creation)** — inspect the issue's labels:
   - No `model:*` label → judge the tier with the inline rubric and
     `gh issue edit <num> --add-label "model:<tier>"` (create the label first if
     the repo lacks it, same one-liner pattern as `need:human`).
   - Multiple `model:*` labels → keep the highest tier, remove the rest.
   - A pre-existing tag (scanner-filed or human-applied) is used as-is — manual
     tagging is the supported override path.
2. **Step 6 dispatch** — the implementer Agent dispatch passes
   `model: <tier from the tag>`. Per-task reviewers
   (`superpowers:requesting-code-review`), triage, and step-7 final verify are
   unchanged and inherit the session model.
3. **BLOCKED escalation made deterministic** (`SKILL.md:228`,
   `references/subagent-spec.md:120`) — replace "dispatch a more capable model"
   with: escalate exactly one tier (haiku→sonnet, sonnet→opus) and re-dispatch,
   **updating the issue's `model:*` label to match**; a BLOCKED return at opus
   goes straight to `NEEDS-HUMAN`. The existing rule stands: never re-dispatch
   the same model with the same spec.

## Out of scope (YAGNI)

- No `model:fable` tier.
- No model routing for reviewer/scanner subagents themselves.
- No tier→model indirection table; the label value *is* the Agent param value.

## Sync notes

- `skills/` is the distributable source of truth. `devpilot-scanning-repos`
  also exists under `.claude/skills/` — copy the updated files there in the same
  PR. `devpilot-resolve-issues` is not installed under `.claude/skills/`; only
  `skills/` changes.
- `skills/index.json` does not change (no skill added or removed).

## Testing

- Run `scripts/check-findings.py` against a sample finding with/without the
  `model` field to confirm the new validation.
- Re-read both SKILL.md files end-to-end after editing: no remaining "five
  labels" / "exactly five" phrasing in scanning-repos, no remaining vague "more
  capable model" phrasing in resolve-issues.
- Eval assertions in `evals/evals.json` updated in the same PR (per repo
  convention: label contract and evals move together).
