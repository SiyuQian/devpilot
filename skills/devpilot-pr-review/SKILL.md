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

## Step 3 — Write the Review and Post It

Match the PR's language (Chinese PR → Chinese review).

**Default behavior: post the review to the PR.** Do not stop at printing it in chat.

**Tone rules — this review is posted publicly on the PR:**
- No emoji anywhere. Not in headings, not in findings, not in the sign-off.
- No exclamation marks, no colloquialisms, no hedging flourishes ("just a thought", "maybe").
- Open with a short, professional greeting addressed to the PR author by their GitHub/GitLab handle (resolved from `gh pr view --json author -q .author.login`, rendered as `@handle`). If the handle cannot be resolved, use "Hi there,".
- Close with the HTML comment metadata block exactly as shown below. The block is required on every posted review so readers can tell which skill version produced it.

### Skill Version

This skill is currently at **v0.1.0**. Bump this version (and the value in the template below) whenever the template structure, tone rules, or Step 1 questions change in a user-visible way. Treat the version as a stable identifier readers can grep for.

### Template

```
<!-- devpilot-pr-review v0.1.0 -->
Hi @<author-handle>,

Thanks for the change. Review below.

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

---
<!--
Generated by devpilot-pr-review skill v0.1.0
Model: <model id, e.g. claude-opus-4-7>
Review mode: <request-changes | comment | approve>
-->
```

The trailing HTML comment block renders as nothing in the GitHub/GitLab UI but is preserved in the raw markdown. Fill in all three fields; do not leave placeholders.

## Step 4 — Post the Review to the PR (Default)

Show the user the drafted review first, then post it. Default flow:

- **GitHub** — post as a PR review (not a loose comment) so severity is explicit:
  - Findings include ≥1 **Blocking** → `gh pr review <url> --request-changes --body-file -`
  - Only Should-fix / Consider / Nits → `gh pr review <url> --comment --body-file -`
  - No findings at all → `gh pr review <url> --approve --body-file -`
  - Pipe the rendered markdown via stdin (`printf '%s' "$review" | gh pr review ...`) to avoid shell-quoting issues.
- **GitLab** — `glab mr note <id> --message "$review"` (GitLab has no "request changes" state; severity lives inside the body).

**Do NOT post when:**
- User explicitly said "don't post" / "just draft" / "local only" / "dry run".
- Review is on a patch the user pasted into chat (no real PR exists).
- PR is already merged or closed — tell the user and skip.

If posting is skipped for any reason, say so explicitly so the user knows the review is local-only.

**Inline comments (advanced, opt-in only):** For line-level feedback, use the GitHub review API:
`gh api -X POST /repos/{owner}/{repo}/pulls/{num}/reviews -f event=... -f body=... -F 'comments[]=...'`
with each item as `{path, line, side, body}`. Only do this if the user asks for inline comments; otherwise stay with the summary review above.

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
| "I'll just print the review in chat, user can paste it." | Default is to post. Skip posting only if the user said so, or no real PR exists. |
| "Blocking finding, but I'll post as `--comment` to be polite." | If it's Blocking, use `--request-changes`. Severity must match the posted review state. |
| "A small emoji softens the tone." | No emoji. The review is a professional artifact on the PR record. |
| "Greeting feels redundant, skipping it." | The greeting is required. It addresses the author by handle and sets tone. |
| "The version comment is noise, I'll drop it." | The `<!-- devpilot-pr-review vX.Y.Z -->` block is required so readers can attribute and triage the review. |

## Red Flags — Stop and Restart

- Started writing line comments before answering the five questions.
- Every finding is naming / formatting / "could be cleaner".
- Comparing two options the author already listed, instead of asking what options they didn't consider.
- Said "LGTM" without tracing one input through the change.
- Only looked at files in the diff, never at callers or tests.

All of these mean: back up to Step 1 and restart.
