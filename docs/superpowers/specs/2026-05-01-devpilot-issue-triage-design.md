# devpilot-issue-triage — Design

## Purpose

Bridge the gap between `devpilot-scanning-repos` (creates issues) and `devpilot-resolve-issues` (executes fixes) by classifying an existing GitHub issue backlog into actionable buckets and drafting — but **not posting** — the follow-up actions.

## Triggering

Use when the user wants to triage, classify, sort, or sweep open GitHub issues to decide what's worth working on. Triggers: "triage these issues", "what's worth fixing", "sort the backlog", "/triage", "分诊 issues", "整理 backlog".

Do **not** use for: filing new issues (→ `devpilot-scanning-repos`), executing fixes (→ `devpilot-resolve-issues`), or reviewing a single PR (→ `devpilot-pr-review`).

## Workflow

1. **Collect**: `gh issue list` (open, optional filter: label, author, age) → produce ordered list.
2. **Batch classify** — for each issue, assign exactly one bucket:
   - `ready-to-fix` — clear scope, reproducible, small, no design questions; eligible for `devpilot-resolve-issues`.
   - `needs-info` — missing repro / env / expected behavior; reporter must answer questions.
   - `needs-design` — real problem, but solution requires discussion or trade-off decisions.
   - `duplicate` — same as another open or closed issue (link required).
   - `stale` / `won't-fix` — outdated, abandoned, or out-of-direction.
   - `out-of-scope` — legitimate request, wrong repo.
   Bucket choice MUST come with one-line evidence ("no repro steps", "duplicates #88").
3. **Deep dive** — for every `ready-to-fix` issue only:
   - Read cited files, scan related code paths.
   - Note suspected root cause (1–2 sentences).
   - Estimate fix size: XS (< 30 min) / S (< 2h) / M (half-day) / L (split first).
   - If deep-dive reveals the issue is actually `needs-design` or `needs-info`, demote it.
4. **Draft actions** — write but do not post:
   - For `needs-info`: drafted question comment.
   - For `duplicate`: drafted close-comment with link.
   - For `stale`: drafted close-comment with rationale.
   - For `ready-to-fix`: 1-line handoff hint for `devpilot-resolve-issues`.
   - Suggested labels per issue.
5. **Output**: a single markdown report at `./issue-triage-<owner>-<repo>-<YYYY-MM-DD>.md` containing:
   - Summary counts per bucket.
   - Per-issue section: number + title, bucket, evidence, drafted action, suggested labels, deep-dive (if applicable).
   - Footer: copy-pasteable command to feed the `ready-to-fix` list into `devpilot-resolve-issues`.

## Hard rules (the "B" boundary)

- **Read-only against GitHub.** Never call `gh issue comment`, `gh issue close`, `gh issue edit`, `gh label`, etc.
- All proposed mutations live in the report as drafts.
- Closing the loop (actually posting) is the user's job, or they can hand the report to a different skill.

## Non-goals

- No automatic re-triage on a schedule.
- No cross-repo dedup beyond the current repo.
- No prioritization scoring beyond bucket + size — leave priority to humans.

## Files

```
skills/devpilot-issue-triage/
  SKILL.md
  references/
    bucket-rubric.md     # decision tree for bucket assignment + examples
    report-template.md   # exact markdown skeleton for the output file
    draft-comments.md    # canonical wording for needs-info / duplicate / stale comments
```

## Success criteria

- A subagent given a 10-issue sample classifies every issue into exactly one bucket with cited evidence.
- The report is immediately actionable: every drafted comment is paste-ready, every `ready-to-fix` entry has a handoff hint.
- No GitHub state was modified during the run.
