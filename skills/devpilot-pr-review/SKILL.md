---
name: devpilot-pr-review
description: Use when the user asks to review a pull request, merge request, or a diff — "review this PR", "review PR #123", "look over these changes", "check my diff before I merge", "/review", or when they share a PR URL and ask for thoughts. Findings are posted as inline comments anchored to specific lines so the author can act on each one in place. Do NOT use for pure style/lint review, formatting-only changes, or language-specific idiom review (defer to style skills like devpilot-google-go-style).
---

# PR Review (Eligibility-Gated, Parallel-Fanout, Inline-First)

## Overview

A PR review the author can act on: every concrete finding is posted as an **inline comment anchored to the line it talks about**, drawn from a **parallel multi-angle scan** and filtered through an explicit **confidence rubric** so the author sees signal, not noise. The body holds a short verdict, strengths, the blind-spot sweep, and counts.

Three structural ideas drive this skill:

1. **Eligibility gate** — decide the PR is worth a full review before spending tokens on it. Dependabot, drafts, generated-file PRs, "already reviewed" all stop here.
2. **Parallel fanout** — five core subagents (A–E) plus an optional sixth (F) look at the change from independent angles in parallel. Coverage comes from diversity of angle, not depth of a single pass. The main session dispatches and merges; subagents read code. Agent F (Dependency Reality Check) is dispatched only when the dispatcher's pre-extracted dependency manifest is non-empty — it verifies imports/packages resolve on their public registry, catching hallucinated names that pass every text-based agent.
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
1.5 Graph enrichment        → references/graph.md (preflight once; fallback OK)
2. Parallel fanout          → references/fanout.md (5 subagents, parallel)
3. Filter + merge + reconcile against graph → references/confidence.md
4. Draft review             → references/template.md
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

### 1.5. Graph enrichment

Run `devpilot graph preflight --base <base-sha> --head <head-sha>` once. Cache the JSON to `/tmp/pr_review_graph.json` and inject it into the shared header that every fanout brief sees. The payload tells subagents — before they read any code — which symbols changed, who calls each, which are hubs, which lack tests, and which cross-community edges this PR adds. Agent A's blast-radius answer comes from this payload, not from grep.

If the graph cache is missing, the language is unsupported, or preflight fails, **fall back** to the grep-only path and note `Behavior trace: grep-only (graph unavailable: <reason>)` in the body's sweep summary. Do not auto-run `devpilot graph build`. See `references/graph.md` for the full payload schema, fallback triggers, and confidence-weighting rules.

### 2. Parallel fanout (5 core + F conditional)

Dispatch all in a single message so they run in parallel. Each receives the PR metadata, the diff, and one focused brief. Each returns findings with `Confidence: 0–100` and `Severity`. See `references/fanout.md` for the prompts. Agent F is conditional: the dispatcher pre-extracts the dependency manifest per `references/import-verifier.md` → "What the dispatcher pre-extracts"; F is dispatched only when that manifest is non-empty.

In **incremental mode**, the diff passed to subagents is the range diff (`last_reviewed_sha..head_sha`), not the full PR diff — agents should look at the new commits only. Agent A still grounds its blast-radius checks in the full repo, but findings must be anchored to lines changed in the new commits.

| Agent | Angle |
|---|---|
| A | Behavior sweep (5 blind-spot questions + behavior trace) |
| B | Shallow bug scan on the diff + Security/Performance [REQUIRED CHECKS] coverage |
| C | CLAUDE.md / AGENTS.md compliance |
| D | Git blame & history + comments on prior PRs touching these files |
| E | Code comments & in-file conventions in modified files |
| F | Dependency reality check — verifies added imports/packages resolve on public registry (conditional: only dispatched when the diff adds dependencies) |

The main session does NOT also do these passes itself. Subagent context savings are the point.

### 3. Filter, dedupe, merge

Graph-reconcile each finding (corroborated → floor 85; contradicted → cap 50) → drop `Confidence < 70` → drop matches against `eligibility.md` false-positive list, including duplicates of existing inline comments at the same anchor (from step 0) → dedupe across agents; same defect across multiple files → one consolidated comment listing the other `path:line`s → anchor each survivor to `(path, line)`. Full procedure incl. graph-injected missing-test findings: `references/confidence.md`.

### 4. Draft the review

One inline comment per anchored finding: severity-tagged title + Behavior today / Why that's a problem / Suggested change / Confidence. One body: Verdict + TL;DR + Strengths + Unknown-Unknowns Sweep summary (from Agent A) + Security/Performance coverage line + inline-finding counts + Open Questions. Templates, field rules, and tone/stance/language are in `references/template.md`. Calibrate against `references/example-review.md` on first use.

### 5. Post

Single combined POST to `repos/:owner/:repo/pulls/:num/reviews` carrying `{event, body, comments[]}` in one call — never split into multiple reviews and never post inline findings via `gh pr comment`. Event derived from highest severity (`confidence.md` → "Severity rubric"). Links in the body use full-SHA `blob` URLs so GitHub renders the snippet preview.

Payload shape:

```bash
jq -n --arg event "$event" --arg body "$body" --argjson comments "$comments_json" \
  '{event:$event, body:$body, comments:$comments}' \
| gh api -X POST "repos/$owner/$repo/pulls/$num/reviews" --input -
# each entry in $comments_json: {path, line, side:"RIGHT"|"LEFT", body}
```

See `references/posting.md` for the full `jq` build, anchor field rules (multi-line / LEFT side / `start_line`), GitLab equivalent, and the local-only "skip posting" mode. Before posting, walk `references/rationalizations.md` self-check.

## Cross-References

- Code quality at the naming / function / class level → `devpilot-clean-code-principles`.
- Go-specific idiom review → `devpilot-google-go-style`.
- Defer to those skills rather than duplicating their content.

## Reference Index

| File | What's in it |
|---|---|
| `references/eligibility.md` | Gate rules + false-positive list (when to skip review entirely, what to never flag). |
| `references/graph.md` | `devpilot graph preflight` payload schema, fallback triggers, confidence-weighting rules consumed by step 3. |
| `references/fanout.md` | Six subagent prompts (Behavior, Bug scan + sec/perf coverage, CLAUDE.md, Git history, In-file comments, Dependency reality) — A–E receive the graph payload; F receives the pre-extracted dependency manifest. |
| `references/import-verifier.md` | Agent F spec: per-ecosystem registry-check commands (Go / npm / Python / Rust), finding shape, typosquat heuristic, fallback rules. |
| `references/confidence.md` | 0–100 rubric, threshold 70, severity vs. confidence axes, dedupe rules, graph reconciliation. |
| `references/unknown-unknowns.md` | Behavior sweep details — Agent A's playbook. |
| `references/checklist.md` | Quality dimensions referenced by Agent B's bug scan and Agent A's checklist tail. |
| `references/template.md` | Inline comment template + review body template (Verdict, Strengths, sweep, counts) + tone/stance/language rules. |
| `references/posting.md` | One combined POST (`gh api`), full-SHA link format, GitLab equivalent. |
| `references/example-review.md` | Worked example: body + inline comments. |
| `references/rationalizations.md` | Common shortcuts + pre-post self-check. |
