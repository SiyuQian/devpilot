---
name: devpilot-scanning-repos
description: Use when the user asks to scan, audit, or sweep an entire GitHub repository for issues and file them as tickets — "scan this repo", "audit the codebase", "find bugs/security holes/missing tests", "/repo-scan", "open issues for all the problems you find". Scans security, edge cases, and testing coverage without assuming business logic. Do NOT use for reviewing a single PR (use devpilot-pr-review) or language-specific style review (use devpilot-google-go-style).
---

# Repo Scan (Security / Edge Cases / Coverage → GitHub Issues)

## Files in this skill

| File | When to load |
|---|---|
| `agents/security-scanner.md` | Step 3 — sub-agent prompt for the security scanner. |
| `agents/edge-case-hunter.md` | Step 3 — sub-agent prompt for edge-case hunting (no business logic). |
| `agents/coverage-auditor.md` | Step 3 — sub-agent prompt for test-coverage gap detection. |
| `references/scoring.md` | Step 4 — full 0/25/50/75/100 rubric + false-positive classes. |
| `references/issue-template.md` | Step 7 — exact `gh issue create` body and label contract. |
| `references/labels.md` | Step 2 — one-shot `gh label create` commands. |
| `scripts/check-findings.py` | Step 3.5 — validates each scanner's JSON output against the schema. |
| `evals/evals.json` | Test scenarios for skill behavior (not loaded at runtime). |

## Overview

A whole-repo sweep that dispatches **three parallel specialist sub-agents**, scores every finding 0–100 for confidence, filters below threshold, then files each surviving finding as a labeled GitHub issue. Business logic is out of scope — scanners only catch mistakes a reasonable reader could flag without domain knowledge.

**Core principle:** coverage during scan, filtering during scoring, noise-free issues at the end. The sub-agents are told to surface everything they notice; a separate scoring pass kills the noise so the human only sees load-bearing issues.

## When NOT to Use

- Single PR / diff review → `devpilot-pr-review`.
- Pure style / lint / formatting → the relevant style skill (`devpilot-google-go-style`, etc.) + the project's linter.
- Business-logic correctness ("does this function compute the right tax rate?") → a human with domain context.
- Repo without a `.github`-style issue tracker, or user doesn't want issues created → ask first; print findings to terminal instead.

## Workflow

1. **Resolve target.** Accept `owner/repo`, a clone URL, or "this repo" (use `gh repo view --json nameWithOwner`). Verify with `gh repo view`.
2. **Ensure labels exist.** Create (idempotently) the labels this skill uses: `repo-scan`, `scan:security`, `scan:edge-case`, `scan:coverage`, `severity:high`, `severity:medium`, `severity:low`. See `references/labels.md` for the `gh label create` incantations.
3. **Dispatch scanners in parallel.** In ONE message, launch three sub-agents using the prompts in `agents/`:
   - `agents/security-scanner.md`
   - `agents/edge-case-hunter.md`
   - `agents/coverage-auditor.md`
   Each returns a list of `Finding` objects (see format below). Scanners are told to emit everything they notice — including low-severity — because filtering happens in step 4, not in the scanner.
3.5. **Validate scanner output.** Pipe each scanner's JSON array through `python3 scripts/check-findings.py`. It exits non-zero and prints the offending object if any finding is missing a required field, uses an invalid `category`/`severity` enum, or has an empty `evidence` block. Fix (or ask the scanner to re-emit) before scoring.
4. **Score every finding.** For each finding, dispatch a lightweight scoring sub-agent using the rubric in `references/scoring.md`. Scores are 0, 25, 50, 75, or 100 — the same scale used by the official `/code-review:code-review` command, adapted for repo-wide scans.
5. **Filter.** Drop every finding with score `< 75`. If zero survive, stop — report "no high-confidence issues found" to the user and do not create issues.
6. **Deduplicate against existing issues.** Before filing, run `gh issue list --label repo-scan --state all --limit 200 --json title,body,number` and skip findings whose normalized title already matches an open or recently-closed scan issue.
7. **File issues.** One `gh issue create` per surviving finding, using the template in `references/issue-template.md`. Labels: always `repo-scan` + one `scan:*` category + one `severity:*`.
8. **Summarize.** Print a compact table to the user: `[category] [severity] title → #issue-number`.

## Finding format

Every scanner returns a JSON array of objects with exactly these fields:

```json
{
  "category": "security | edge-case | coverage",
  "title": "<≤80 chars, imperative — e.g. 'Sanitize shell input in cmd/devpilot/run.go'>",
  "severity": "high | medium | low",
  "file": "<path relative to repo root>",
  "line_range": "L42-L58",
  "evidence": "<2–5 lines quoted from the file, with line numbers>",
  "why_it_matters": "<1–3 sentences, no business-logic claims>",
  "suggested_fix": "<1–3 sentences; null if scanner can't confidently propose one>"
}
```

`evidence` is mandatory. A finding without quotable code is speculation — drop it at the scanner level.

## Scoring rubric (summary — full rubric in `references/scoring.md`)

| Score | Meaning | Action |
|---|---|---|
| 0 | False positive / pre-existing / would be caught by CI | drop |
| 25 | Might be real; scanner couldn't verify | drop |
| 50 | Real but minor nit | drop |
| 75 | Real, meaningful, verified in code | **file** |
| 100 | Certain, load-bearing, reproducible | **file** |

Threshold is 75, matching the confidence bar of a senior reviewer who doesn't cry wolf.

## What scanners MUST NOT flag

(These are pre-filtered at the scanner level — do not let findings like this through.)

- Anything a linter, formatter, type-checker, or compiler catches.
- Business-logic correctness — "this discount calculation is wrong" requires domain context the scanner doesn't have.
- Style / naming / readability nits — defer to style skills.
- "Missing tests" for files that are pure types, generated code, or obviously trivial (constants, one-line accessors).
- Dependency CVEs — that's Dependabot's job.
- Pre-existing issues on lines the scanner can see were untouched in recent history (scanner should `git log` to check if desired).

See each agent prompt for category-specific false-positive classes.

## Quick reference

| I want to… | Do this |
|---|---|
| Scan current repo | `gh repo view --json nameWithOwner` → pass to step 1 |
| Scan a specific path only | Tell each scanner: "scope = `internal/auth/` only" |
| Preview without filing issues | Skip step 2 and 7; print the table |
| Re-scan and not duplicate issues | Step 6 handles this; don't delete existing `repo-scan` issues |
| Change threshold | Edit step 5 — keep the 0/25/50/75/100 scale, move only the cutoff |

## Acceptance criteria (the "test" this skill is written against)

A correct run of this skill produces:

1. Exactly three scanner sub-agent dispatches, in parallel.
2. Every filed issue has `repo-scan` + one `scan:*` + one `severity:*` label.
3. No issue is filed whose scoring-agent score is below 75.
4. No duplicate of an existing open `repo-scan` issue.
5. No finding cites business-logic correctness as the sole reason.
6. Every filed issue body quotes ≥2 lines of actual code with a `<file>#L<start>-L<end>` link.
7. If zero findings survive scoring, no issues are filed and the user is told so explicitly.

If any of these is violated, the skill failed — stop and correct before continuing.

## Common mistakes

- **Letting scanners filter their own output.** They should over-report. The scoring pass does the filtering. Merging the two loses calibration.
- **Using the scanner agent to also create the issue.** Don't — the scanner has too much context. File issues from the main agent after scoring.
- **Dropping the evidence block in the issue body.** Without it the human has to re-derive the finding. File a crap issue once and nobody trusts the skill.
- **Creating labels inside the issue-creation loop.** Race-y and noisy. Do it once upfront (step 2).
- **Asking scanners to rank severity *and* confidence.** Confidence is the scoring pass's job; scanners assign severity only.
- **Forgetting the dedupe step.** Re-running the skill must be idempotent or the user will stop running it.

## Evaluation

Test scenarios for this skill live in `evals/evals.json`. Each eval gives a prompt, expected output shape, and machine-checkable assertions (e.g. *`exactly_three_scanner_dispatches`*, *`no_business_logic_findings_filed`*, *`all_issues_have_three_labels`*). Run before shipping any change to scanner prompts or the scoring rubric.
