---
name: devpilot-pr-review
description: Use when the user asks to review a pull request, merge request, or a diff — "review this PR", "review PR #123", "look over these changes", "check my diff before I merge", "/review", or when they share a PR URL and ask for thoughts. Do NOT use for pure style/lint review, formatting-only changes, or language-specific idiom review (defer to style skills like devpilot-google-go-style).
---

# PR Review (Behavior-First, Unknown-Unknowns)

## Overview

Most PR review fails by staying inside an already-narrow option set: naming, formatting, "LGTM". This skill pushes the review onto the **behavior** the PR introduces into the system, including behavior neither the author nor the reviewer has noticed yet.

**Core principle:** Answer the five blind-spot questions before writing findings. Those questions exist to surface what the diff alone cannot show.

## When NOT to Use

- Pure formatting / lint / rename PRs — defer to the relevant style skill.
- Generated-file or dependency-bump PRs with no behavior change — quick sanity check, skip the sweep.
- No PR, diff, or branch given — ask the user for one.

## Two rules that govern every finding

<coverage_first_findings>
Report every finding you reach after tracing the code, including ones you are uncertain about or judge low-severity. Your job at this stage is **coverage**, not filtering. Each finding carries its own `Confidence` and `Severity` so the author and any downstream reviewer can rank and filter. A finding that later gets filtered out is fine; a finding that was silently dropped because it felt minor is not.
</coverage_first_findings>

<investigate_before_asserting>
State how the code behaves only after opening and reading the relevant files. When a finding depends on a caller or test you have not located, mark it `Confidence: low` and record the gap in `Open Questions` rather than speculating.
</investigate_before_asserting>

## Workflow

### Step 0 — Load the PR

- PR URL → `gh pr view <url> --json title,body,files,baseRefName,author` + `gh pr diff <url>`
- Local branch → `git diff <base>...HEAD` + `git log <base>..HEAD --oneline`
- Pasted patch → read directly

Read the PR description and any linked issue. A PR with no stated intent is itself a finding worth surfacing.

### Step 1 — Unknown-Unknowns Sweep

Answer the five blind-spot questions in writing before writing findings. `references/unknown-unknowns.md` has the full questions, the pitfall table per change class, and the output format.

### Step 2 — Behavior Trace

For each meaningful change, trace at least one golden-path input and one edge-case input through the code. Record:

- The observable behavior delta (inputs → outputs, side effects, state).
- Behavior changes not mentioned in the PR description (new log lines, new DB writes, changed defaults, changed ordering, new error paths).
- How we would detect a break in production (logs, metrics, errors).

A review that reaches "LGTM" without tracing at least one input through at least one change has not completed Step 2.

### Step 3 — Write the Review

Render the template from `references/template.md`. A fully-filled reference is in `references/example-review.md` — read it the first time you use this skill, and whenever you are calibrating tone or depth.

Write every section (TL;DR, findings, disclaimer, Open Questions, metadata) in the PR's language. Chinese PR → Chinese review, end to end.

**Tone.** Professional prose. Skip emoji, exclamation marks, and softeners like "just a thought" or "maybe". Greet the PR author by their resolved `@handle`.

**Stance.**
- State system behavior as claims, not questions. Traced claims belong in Behavior Findings; the corresponding question belongs in `Open Questions` only when the code could not answer it.
- When you see a concrete alternative, name it. Give one sentence on why it is better, and ask the author to confirm the direction.
- Every finding carries `Confidence: high | medium | low`. Confidence and severity are independent axes.

### Step 4 — Post the Review

Show the user the drafted review, then post. Default flow:

- **GitHub** — `printf '%s' "$review" | gh pr review <url> <mode> --body-file -`
  - `--request-changes` when any finding is Blocking
  - `--comment` when findings are only Should-fix / Consider / Nit
  - `--approve` when there are no findings
- **GitLab** — `glab mr note <id> --message "$review"`
- **Inline comments (opt-in)** — use `gh api -X POST /repos/{owner}/{repo}/pulls/{num}/reviews` with `comments[]` entries of `{path, line, side, body}`. Only when the user asks.

Skip posting when the user opts out ("don't post", "dry run", "local only"), the review is on a pasted patch with no real PR, or the PR is already merged or closed. Say so explicitly when skipping.

## Severity rubric

Severity describes *impact if the finding is real*, independent of confidence.

- **Blocking** — would cause data loss, security regression, outage, or silently wrong behavior in production.
- **Should-fix** — real bug on a reachable code path, missing test for a risky path, or an unhandled pitfall from the sweep.
- **Consider** — design or maintainability feedback worth the author's attention.
- **Nit** — style, naming, wording. Bottom of the review.

Report findings at every severity; downstream readers filter.

## Version

The skill itself is unversioned. Every posted review carries the **devpilot binary version** instead. Resolve it at post time via `devpilot --version` (the binary prints e.g. `devpilot version v0.12.2`; take the `vX.Y.Z` token). Fall back to `unknown` when unavailable.

## Cross-References

- Code quality at the naming / function / class level → `devpilot-clean-code-principles`.
- Go-specific idiom review → `devpilot-google-go-style`.
- Defer to those skills rather than duplicating their content.

## Reference Index

- `references/unknown-unknowns.md` — the five blind-spot questions, pitfall table by change class, output format.
- `references/template.md` — the review skeleton and per-field rules.
- `references/example-review.md` — a fully-filled worked example for calibration.
- `references/rationalizations.md` — common shortcuts with rebuttals, plus the pre-post self-check list.
