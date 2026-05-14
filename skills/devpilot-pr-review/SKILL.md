---
name: devpilot-pr-review
description: Use when the user asks to review a pull request, merge request, or a diff — "review this PR", "review PR #123", "look over these changes", "check my diff before I merge", "/review", or when they share a PR URL and ask for thoughts. Findings are posted as inline comments anchored to specific lines so the author can act on each one in place. Do NOT use for pure style/lint review, formatting-only changes, or language-specific idiom review (defer to style skills like devpilot-google-go-style).
---

# PR Review (Eligibility-Gated, Parallel-Fanout, Inline-First)

## Overview

A PR review the author can act on: every concrete finding is posted as an **inline comment anchored to the line it talks about**, drawn from a **parallel multi-angle scan** and filtered through an explicit **confidence rubric** so the author sees signal, not noise. The body holds a short verdict, strengths, the blind-spot sweep, and counts.

Three structural ideas drive this skill:

1. **Eligibility gate** — decide the PR is worth a full review before spending tokens on it. Dependabot, drafts, generated-file PRs, "already reviewed" all stop here.
2. **Parallel fanout** — five subagents look at the change from five independent angles in parallel. Coverage comes from diversity of angle, not depth of a single pass. The main session dispatches and merges; subagents read code.
3. **Confidence filtering** — every finding carries `Confidence: 0–100`. Findings below 70 are dropped by default. Coverage at collection, filtering at posting.

## When NOT to Use

- Pure formatting / lint / rename PRs — defer to the relevant style skill.
- Generated-file or dependency-bump PRs with no behavior change — the eligibility gate stops here.
- No PR, diff, or branch given — ask the user for one.

## Three rules that govern every finding

<coverage_in_collection_filtering_at_posting>
Subagents report every finding they reach, including uncertain ones — that is the only way to get coverage. The main session filters by Confidence and Severity at posting time. A finding silently dropped by a subagent because it "felt minor" is a defect in the review; a finding scored 40 and filtered out at the gate is fine.
</coverage_in_collection_filtering_at_posting>

<investigate_before_asserting>
State how the code behaves only after opening and reading the relevant files. When a finding depends on a caller or test the subagent did not locate, it MUST score Confidence ≤ 50 and record the gap. No speculation passed off as fact.
</investigate_before_asserting>

<inline_by_default>
Every finding tied to a specific line goes in as an inline review comment, never in the body. The body holds only the Verdict, TL;DR, Strengths, the sweep summary, finding counts, and Open Questions. If a finding has no obvious anchor (cross-cutting concern, missing-but-not-present code), anchor it to the most representative line and say so in the comment — do not promote it to the body.
</inline_by_default>

## Workflow

```
0. Eligibility gate         → references/eligibility.md
1. Load PR                  → gh / git / pasted patch
2. Parallel fanout          → references/fanout.md (5 subagents, parallel)
3. Filter + merge           → references/confidence.md (threshold 70, dedupe)
4. Draft review             → references/template.md + style.md
5. Post one combined POST   → references/posting.md
Self-check before post      → references/rationalizations.md
```

### 0. Eligibility gate

Before anything else, run the gate in `references/eligibility.md`. If the PR is closed, draft, automation-only, generated-only, or already reviewed by you **at the current head SHA**, stop and tell the user. Cheap; saves an entire fanout.

The gate also produces two outputs the later steps consume:

- **Review mode** — `full` (no prior devpilot review) or `incremental` (prior review exists but head has moved; fanout runs against `last_reviewed_sha..head_sha`, not the full PR diff).
- **Existing review comments** — `/tmp/existing_review_comments.json`, every prior inline comment on this PR from any reviewer. Used by step 3 to drop findings that duplicate an existing comment.

### 1. Load the PR

```bash
gh pr view <url> --json title,body,files,baseRefName,headRefOid,author
gh pr diff <url>
```

Or `git diff <base>...HEAD` for a local branch, or read a pasted patch directly. Capture the **head SHA** — link rendering depends on it (see `references/posting.md`). A PR with no stated intent is itself a finding.

### 2. Parallel fanout (5 subagents)

Dispatch all five in a single message so they run in parallel. Each receives the PR metadata, the diff, and one focused brief. Each returns findings with `Confidence: 0–100` and `Severity`. See `references/fanout.md` for the prompts.

In **incremental mode**, the diff passed to subagents is the range diff (`last_reviewed_sha..head_sha`), not the full PR diff — agents should look at the new commits only. Agent A still grounds its blast-radius checks in the full repo, but findings must be anchored to lines changed in the new commits.

| Agent | Angle |
|---|---|
| A | Behavior sweep (5 blind-spot questions + behavior trace) |
| B | Shallow bug scan on the diff |
| C | CLAUDE.md / AGENTS.md compliance |
| D | Git blame & history + comments on prior PRs touching these files |
| E | Code comments & in-file conventions in modified files |

The main session does NOT also do these passes itself. Subagent context savings are the point.

### 3. Filter, dedupe, merge

Apply the rubric in `references/confidence.md`:

- Drop findings with `Confidence < 70`.
- Drop findings that match the false-positive list in `references/eligibility.md` (pre-existing issues, lines the PR did not modify, linter/typechecker-catchable, ignored-by-comment, **already raised by an existing review comment at the same anchor**, etc.). The existing-comments file from step 0 is the source of truth for the duplicate check.
- Dedupe across agents (same line, same defect → one inline comment, take the higher confidence).
- Assign each surviving finding an inline anchor `(path, line)`. Cross-cutting findings anchor to the most representative line.

### 4. Draft the review

One inline comment per anchored finding using `references/template.md` → "Inline comment template". One body using `references/template.md` → "Review body template": Verdict + TL;DR + Strengths + Unknown-Unknowns Sweep summary (from Agent A) + counts + Open Questions. Apply tone/stance/language from `references/style.md`. Calibrate against `references/example-review.md` on first use.

### 5. Post

Single combined POST: body + inline comments + event. Links in body use full-SHA format. See `references/posting.md`.

Before posting, walk `references/rationalizations.md` self-check.

## Cross-References

- Code quality at the naming / function / class level → `devpilot-clean-code-principles`.
- Go-specific idiom review → `devpilot-google-go-style`.
- Defer to those skills rather than duplicating their content.

## Reference Index

| File | What's in it |
|---|---|
| `references/eligibility.md` | Gate rules + false-positive list (when to skip review entirely, what to never flag). |
| `references/fanout.md` | Five subagent prompts (Behavior, Bug scan, CLAUDE.md, Git history, In-file comments). |
| `references/confidence.md` | 0–100 rubric, threshold 70, severity vs. confidence axes, dedupe rules. |
| `references/unknown-unknowns.md` | Behavior sweep details — Agent A's playbook. |
| `references/checklist.md` | Quality dimensions referenced by Agent B's bug scan and Agent A's checklist tail. |
| `references/template.md` | Inline comment template + review body template (Verdict, Strengths, sweep, counts). |
| `references/style.md` | Tone, stance, and language rules for both body and inline comments. |
| `references/posting.md` | One combined POST (`gh api`), event mapping, full-SHA link format, GitLab equivalent. |
| `references/example-review.md` | Worked example: body + inline comments. |
| `references/rationalizations.md` | Common shortcuts + pre-post self-check. |
