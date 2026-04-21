---
name: devpilot-pr-review
description: Use when the user asks to review a pull request, merge request, or a diff — "review this PR", "review PR #123", "look over these changes", "check my diff before I merge", "/review", or when they share a PR URL and ask for thoughts. Do NOT use for pure style/lint review, formatting-only changes, or language-specific idiom review (defer to style skills like devpilot-google-go-style).
---

# PR Review (Behavior-First, Unknown-Unknowns)

## Overview

Most PR review fails by staying inside an already-narrow option set: naming, formatting, "LGTM". The job of this skill is to force the review *out* of that narrow set and onto the **behavior** the PR introduces into the system — including behavior neither you nor the author has noticed yet.

**Core principle:** Before any line-level comment, answer the five blind-spot questions. Skipping them is the failure mode this skill exists to prevent.

## When NOT to Use

- Pure formatting / lint / rename PRs → defer to the relevant style skill.
- Generated-file or dependency-bump PRs with no behavior change → quick sanity check, skip the five-question sweep.
- You haven't been given a PR, diff, or branch to review → ask for one, don't guess.

## Step 0 — Load the PR

- PR URL → `gh pr view <url> --json title,body,files,baseRefName` + `gh pr diff <url>`
- Local branch → `git diff <base>...HEAD` + `git log <base>..HEAD --oneline`
- Pasted patch → read directly

Read the PR description / linked issue. A PR with no stated intent is itself a finding — note it and ask the user before continuing.

## Step 1 — Five Blind-Spot Questions (MANDATORY, answer in writing)

Write these answers out explicitly in your notes. "N/A, because X" is a valid answer. Silently skipping any of them = restart.

1. **Local pattern fit.** How is this kind of change done elsewhere in *this* codebase? Is the PR matching convention, diverging on purpose, or diverging by accident?
2. **Blast radius.** Beyond the diff: who calls the changed code? What tests, configs, flags, migrations, docs, downstream services depend on this behavior?
3. **Known pitfalls for this change class.** Name the class first (auth / concurrency / migration / query / prompt / retry / cache / etc.), then check each known pitfall against the diff. Include **security, data integrity, and reversibility** here — new input boundaries, secret leaks, non-idempotent writes, irreversible migrations.
4. **Stale-training check.** Anything I'm about to assert as "the right way" that my training might be 6–18 months stale on? Verify against `go.mod` / lockfiles / recent sources before asserting.
5. **Hand-rolled vs. off-the-shelf.** What in this PR is being hand-rolled (retry, rate-limit, cache, diff parsing, date math, auth flow, slugify, …) that has a mature option already in the repo or its deps?

## Step 2 — Behavior Trace

For each meaningful change, trace at least one golden-path input and one edge-case input through the code. Ask:

- What is the observable behavior delta (inputs → outputs, side effects, state)?
- What *else* changed that isn't in the PR description? (new log lines, new DB writes, changed defaults, changed ordering, new error paths)
- If this breaks in prod, how would we know? (logs, metrics, errors)

"LGTM" without tracing one input = restart.

## Step 3 — Write the Review

Match the PR's language (Chinese PR → Chinese review).

```
## PR Review: <title / #number>

### TL;DR
<1–2 sentences: safe to merge? single most important thing to address?>

### Unknown-Unknowns Sweep
1. Local pattern fit: <finding or "matches convention in X">
2. Blast radius: <finding>
3. Known pitfalls (incl. security/data/reversibility): <finding>
4. Stale-training check: <finding or "N/A">
5. Hand-rolled vs. off-the-shelf: <finding or "N/A">

### Behavior Findings
#### [Blocking | Should-fix | Consider] <short title>
- **Where:** `path/file.go:42`
- **Behavior today on this branch:** <what the code actually does>
- **Why that's a problem:** <impact on users / data / operability — not style>
- **Suggested change:** <concrete direction>

### What's working well
<2–3 bullets worth preserving>

### Nits (optional)
<style / naming / wording, one line each — never blocking>
```

## Severity Rubric

- **Blocking** — data loss, security regression, outage, silently wrong behavior. Must fix before merge.
- **Should-fix** — real bug, missing test for a risky path, unhandled pitfall from Step 1.3.
- **Consider** — design feedback worth attention, not blocking.
- **Nit** — style/naming/wording, bottom of the review, never blocking.

Prefer few high-signal findings over many mixed-severity ones.

## Cross-References

- Code quality at the naming / function / class level → `devpilot-clean-code-principles`
- Go-specific idiom review → `devpilot-google-go-style`
- Don't duplicate those here; cite them.

## Rationalization Table

| Excuse | Reality |
|---|---|
| "This PR is small, skip the five questions." | Small PRs change defaults, delete branches, flip flags. 2 minutes to answer. |
| "The author explained intent in the description." | Intent ≠ actual behavior. Still trace one input through the code. |
| "Diff looks clean, no need to look at callers." | Diff shows *what changed*, not the blast radius. Always check Step 1.2. |
| "I recognize the pattern, I can assert the right way." | Your training is 6–18 months stale. Do Step 1.4 before asserting. |
| "Writing a retry loop / cache / parser here is fine." | Almost always has a mature off-the-shelf option. Do Step 1.5. |
| "LGTM, nothing jumps out." | If nothing jumps out, you haven't traced an input yet. Do Step 2. |
| "Findings are mostly naming — let me file them as Should-fix." | Style findings go in Nits. Never Blocking or Should-fix. |

## Red Flags — Stop and Restart

- Started writing line comments before answering the five questions.
- Every finding is naming / formatting / "could be cleaner".
- Comparing two options the author already listed, instead of asking what options they didn't consider.
- Said "LGTM" without tracing one input through the change.
- Only looked at files in the diff, never at callers or tests.

All of these mean: back up to Step 1 and restart.
