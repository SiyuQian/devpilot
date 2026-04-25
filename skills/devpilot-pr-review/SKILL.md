---
name: devpilot-pr-review
description: Use when the user asks to review a pull request, merge request, or a diff — "review this PR", "review PR #123", "look over these changes", "check my diff before I merge", "/review", or when they share a PR URL and ask for thoughts. Findings are posted as inline comments anchored to specific lines so the author can act on each one in place. Do NOT use for pure style/lint review, formatting-only changes, or language-specific idiom review (defer to style skills like devpilot-google-go-style).
---

# PR Review (Inline-First, Behavior-Aware)

## Overview

A PR review the author can act on: every concrete finding is posted as an **inline comment anchored to the line it talks about**, so the author can see exactly which code is in question and decide finding-by-finding whether to fix. The summary in the review body stays short — TL;DR, the blind-spot sweep, the verdict.

Findings come from two complementary passes, both required:

1. **Behavior sweep** — the five blind-spot questions plus a behavior trace. Catches the parts of the change a narrow diff view misses (`references/unknown-unknowns.md`).
2. **Quality checklist** — code quality, architecture, testing, requirements, production readiness. Catches the parts a behavior pass would skip (`references/checklist.md`).

The behavior sweep is what makes this review more than a style pass. The quality checklist is what makes it more than a behavior trace. Run both.

## When NOT to Use

- Pure formatting / lint / rename PRs — defer to the relevant style skill.
- Generated-file or dependency-bump PRs with no behavior change — quick sanity check, skip the sweep.
- No PR, diff, or branch given — ask the user for one.

## Three rules that govern every finding

<coverage_first_findings>
Report every finding you reach after tracing the code, including ones you are uncertain about or judge low-severity. Your job at this stage is **coverage**, not filtering. Each finding carries its own `Confidence` and `Severity` so the author and any downstream reviewer can rank and filter. A finding that later gets filtered out is fine; a finding that was silently dropped because it felt minor is not.
</coverage_first_findings>

<investigate_before_asserting>
State how the code behaves only after opening and reading the relevant files. When a finding depends on a caller or test you have not located, mark it `Confidence: low` and record the gap in `Open Questions` rather than speculating.
</investigate_before_asserting>

<inline_by_default>
Every finding tied to a specific line goes in as an inline review comment, never in the body. The body holds only the TL;DR, the sweep summary, the inline-finding counts, what's working well, and Open Questions. If a finding has no obvious anchor (cross-cutting concern, missing-but-not-present code), anchor it to the most representative line and say so in the comment — do not promote it to the body.
</inline_by_default>

## Workflow

1. **Load the PR.** `gh pr view <url> --json title,body,files,baseRefName,author` + `gh pr diff <url>`; or `git diff <base>...HEAD` for a local branch; or read a pasted patch directly. A PR with no stated intent is itself a finding.
2. **Run the behavior sweep.** Five blind-spot questions + behavior trace. See `references/unknown-unknowns.md`.
3. **Run the quality checklist.** Code quality, architecture, testing, requirements, production readiness. See `references/checklist.md`.
4. **Draft the review.** One inline comment per anchored finding (`references/template.md` → "Inline comment template"); one summary body for the review (`references/template.md` → "Review body template"). Apply tone, stance, and language rules from `references/style.md`. Calibrate against `references/example-review.md` on first use.
5. **Post it.** Single combined POST: body + inline comments + event. See `references/posting.md`.

Before posting, walk the self-check in `references/rationalizations.md`.

## Cross-References

- Code quality at the naming / function / class level → `devpilot-clean-code-principles`.
- Go-specific idiom review → `devpilot-google-go-style`.
- Defer to those skills rather than duplicating their content.

## Reference Index

| File | What's in it |
|---|---|
| `references/unknown-unknowns.md` | Behavior sweep: five blind-spot questions, change-class pitfalls, behavior trace. |
| `references/checklist.md` | Quality categories: code, architecture, testing, requirements, production readiness. |
| `references/template.md` | Two templates — inline comment per finding, summary body for the review. Severity rubric, version resolution. |
| `references/style.md` | Tone, stance, and language rules for both body and inline comments. |
| `references/posting.md` | `gh api` invocation for one combined POST with inline comments + body; GitLab equivalent. |
| `references/example-review.md` | Worked example: body summary + multiple inline comments. |
| `references/rationalizations.md` | Common shortcuts with rebuttals, plus the pre-post self-check list. |
