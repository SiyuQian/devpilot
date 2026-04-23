---
name: devpilot-pr-review
description: Use when the user asks to review a pull request, merge request, or a diff — "review this PR", "review PR #123", "look over these changes", "check my diff before I merge", "/review", or when they share a PR URL and ask for thoughts. Do NOT use for pure style/lint review, formatting-only changes, or language-specific idiom review (defer to style skills like devpilot-google-go-style).
---

# PR Review (Behavior-First, Unknown-Unknowns)

## Overview

Most PR review fails by staying inside an already-narrow option set: naming, formatting, "LGTM". This skill pushes the review onto the **behavior** the PR introduces into the system, including behavior neither the author nor the reviewer has noticed yet.

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

1. **Load the PR.** `gh pr view <url> --json title,body,files,baseRefName,author` + `gh pr diff <url>`; or `git diff <base>...HEAD` for a local branch; or read a pasted patch directly. A PR with no stated intent is itself a finding.
2. **Answer the five blind-spot questions, then trace behavior.** See `references/unknown-unknowns.md` (also covers the paired Behavior Trace step).
3. **Draft the review.** Render the skeleton in `references/template.md` (includes the severity rubric and version-resolution rules). Apply tone, stance, and language rules from `references/style.md`. Calibrate against `references/example-review.md` on first use.
4. **Post it.** Commands, post-mode mapping, and skip conditions in `references/posting.md`.

Before posting, walk the self-check in `references/rationalizations.md`.

## Cross-References

- Code quality at the naming / function / class level → `devpilot-clean-code-principles`.
- Go-specific idiom review → `devpilot-google-go-style`.
- Defer to those skills rather than duplicating their content.

## Reference Index

| File | What's in it |
|---|---|
| `references/unknown-unknowns.md` | The five blind-spot questions, per-change-class pitfall table, paired Behavior Trace step, output format. |
| `references/template.md` | The review skeleton, per-field rules, version resolution, severity rubric. |
| `references/style.md` | Language / tone / stance rules for the posted review. |
| `references/posting.md` | `gh` / `glab` commands, severity-to-mode mapping, skip conditions, inline-comment API. |
| `references/example-review.md` | Fully-filled worked example for calibration. |
| `references/rationalizations.md` | Common shortcuts with rebuttals, plus the pre-post self-check list. |
