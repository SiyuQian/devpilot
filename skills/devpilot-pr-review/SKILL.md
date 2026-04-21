---
name: devpilot-pr-review
description: Use when the user asks to review a pull request, merge request, or a diff — "review this PR", "review PR #123", "look over these changes", "check my diff before I merge", "/review", or when they share a PR URL and ask for thoughts. Do NOT use for pure style/lint review, formatting-only changes, or language-specific idiom review (defer to style skills like devpilot-google-go-style).
---

# PR Review (Behavior-First, Unknown-Unknowns)

## Overview

Most PR review fails by staying inside an already-narrow option set: naming, formatting, "LGTM". This skill pushes the review onto the **behavior** the PR introduces into the system, including behavior neither the author nor the reviewer has noticed yet.

**Core principle:** Answer the five blind-spot questions before writing line-level comments. Those questions exist to surface what the diff alone cannot show.

## When NOT to Use

- Pure formatting / lint / rename PRs — defer to the relevant style skill.
- Generated-file or dependency-bump PRs with no behavior change — quick sanity check, skip the five-question sweep.
- No PR, diff, or branch given — ask the user for one.

## Coverage, not filtering

<coverage_first_findings>
Report every finding you reach after tracing the code, including ones you are uncertain about or judge low-severity. Your job at this stage is **coverage**, not filtering. Each finding carries its own `Confidence` and `Severity` so the author and any downstream reviewer can rank and filter. A finding that later gets filtered out is fine; a finding that was silently dropped because it felt minor is not.

This rule overrides any instinct to "keep the review tidy" or "only surface what matters". Tidiness is a filtering concern, handled downstream.
</coverage_first_findings>

## Investigate before asserting

<investigate_before_asserting>
State how the code behaves only after opening and reading the relevant files. When a finding depends on a caller or test you have not located, mark it `Confidence: low` and record the gap in `Open Questions` rather than speculating. "I think this might..." is a signal to either open one more file or lower the confidence label — not to post the guess as-is.
</investigate_before_asserting>

## Step 0 — Load the PR

- PR URL → `gh pr view <url> --json title,body,files,baseRefName,author` + `gh pr diff <url>`
- Local branch → `git diff <base>...HEAD` + `git log <base>..HEAD --oneline`
- Pasted patch → read directly

Read the PR description and any linked issue. A PR with no stated intent is itself a finding worth surfacing.

## Step 1 — Five Blind-Spot Questions

Write these answers out explicitly in your notes before writing findings. "N/A, because X" is a valid answer; the important part is that each question gets a deliberate answer rather than being skipped.

1. **Local pattern fit.** How is this kind of change done elsewhere in *this* codebase? Is the PR matching convention, diverging on purpose, or diverging by accident?
2. **Blast radius.** Beyond the diff: who calls the changed code? What tests, configs, flags, migrations, docs, downstream services depend on this behavior?
3. **Known pitfalls for this change class.** Name the class first (auth / concurrency / migration / query / prompt / retry / cache / etc.), then check each known pitfall against the diff. Include **security, data integrity, and reversibility** — new input boundaries, secret leaks, non-idempotent writes, irreversible migrations.
4. **Stale-training check.** Anything you are about to assert as "the right way" that your training might be 6–18 months stale on? Verify against `go.mod` / lockfiles / recent sources before asserting.
5. **Hand-rolled vs. off-the-shelf.** What in this PR is being hand-rolled (retry, rate-limit, cache, diff parsing, date math, auth flow, slugify, …) that has a mature option already in the repo or its deps?

## Step 2 — Behavior Trace

For each meaningful change, trace at least one golden-path input and one edge-case input through the code. For each change, record:

- The observable behavior delta (inputs → outputs, side effects, state).
- Behavior changes not mentioned in the PR description (new log lines, new DB writes, changed defaults, changed ordering, new error paths).
- How we would detect a break in production (logs, metrics, errors).

A review that reaches "LGTM" without tracing at least one input through at least one change has not completed Step 2 yet.

## Step 3 — Write the Review and Post It

Write every section of the review — TL;DR, findings, disclaimer, open questions, metadata — in the PR's language. Chinese PR → Chinese review, end to end. Translate the required blockquote disclaimer while preserving its meaning (automated, not authoritative, human judgment required).

**Posting is the default.** Draft in chat, show the user, then post.

### Tone

Write in professional prose. Skip emoji, exclamation marks, and softeners like "just a thought" or "maybe". Open with a short greeting addressed to the PR author by their GitHub/GitLab handle (resolve via `gh pr view --json author -q .author.login`, render as `@handle`; fall back to "Hi there," if unavailable). Close with the HTML comment metadata block so readers can attribute the review to a devpilot release.

### Stance

- State system behavior as claims, not questions. A traced claim ("This recurses on a 401 from `/refresh` and will stack-overflow") belongs in Behavior Findings; the corresponding question belongs in `Open Questions` only when the code could not answer it.
- When you see a concrete alternative, name it. Give one sentence on why it is better, and ask the author to confirm the direction. Recommendations do not belong inside vague questions.
- Every finding carries an explicit `Confidence: high | medium | low`. Use `low` when you could not fully verify from the code. Low confidence does not demote severity automatically — a high-severity bug you are moderately sure about is still `Severity: Blocking, Confidence: medium`.
- Reserve `Open Questions` for things the code genuinely could not answer (e.g. "Is this endpoint called by the mobile client? I did not find callers outside this repo."). Omit the section entirely if empty.

### Severity

Severity labels describe *impact if the finding is real*, independent of how sure you are (confidence handles that separately).

- **Blocking** — would cause data loss, security regression, outage, or silently wrong behavior in production.
- **Should-fix** — real bug on a code path users can reach, missing test for a risky path, or an unhandled pitfall from Step 1.3.
- **Consider** — design or maintainability feedback worth the author's attention.
- **Nit** — style, naming, wording. Lives at the bottom of the review.

Report findings at every severity; downstream readers filter.

### Version

The skill itself is unversioned. Every posted review carries the **devpilot binary version** instead. Resolve it at post time via `devpilot --version` (the binary prints e.g. `devpilot version v0.12.2`; take the `vX.Y.Z` token). If the command is unavailable, use `unknown`.

### Template

Render the skeleton below, filling every placeholder. Treat the fenced block as a template, not a literal string to post.

```
<!-- devpilot-pr-review (devpilot <version>) -->
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
#### [Blocking | Should-fix | Consider | Nit] <short title>
- **Where:** `path/file.go:42`
- **Behavior today on this branch:** <what the code actually does>
- **Why that's a problem:** <impact on users / data / operability>
- **Suggested change:** <concrete direction>
- **Confidence:** high | medium | low — <one-line reason if not high>

### What's working well
<2–3 bullets worth preserving>

### Nits (optional)
<style / naming / wording, one line each>

### Open Questions (optional — omit if empty)
<only things the code could not answer, one line each>

---

> **Note:** This review was generated automatically by Claude Code. It is not a substitute for human judgment. Treat every finding — including severity labels — as a prompt for the author and human reviewers to verify, not as a final verdict. Approvals do not waive human review.

<!--
Generated by devpilot-pr-review (devpilot <version>, e.g. v0.12.2)
Model: <model id, e.g. claude-opus-4-7>
Review mode: <request-changes | comment | approve>
-->
```

The trailing HTML comment block renders as nothing in the GitHub/GitLab UI but is preserved in the raw markdown. Fill every field.

### Worked Example

A concrete, filled-in review for a hypothetical auth PR. Use it as a calibration reference for tone, depth, and the Confidence field.

<example_review>
<!-- devpilot-pr-review (devpilot v0.12.2) -->
Hi @alex-chen,

Thanks for the change. Review below.

## PR Review: feat(auth): add session refresh on 401 (#214)

### TL;DR
Not safe to merge as-is. The refresh path calls back into itself when the refresh endpoint returns 401, which will stack-overflow the process. Everything else is addressable after that.

### Unknown-Unknowns Sweep
1. Local pattern fit: `internal/auth/client.go` already has a `doWithRetry` helper. This PR adds a parallel retry path instead of extending it.
2. Blast radius: every HTTP call in `internal/api/` routes through the changed `RoundTrip`, not only the endpoint named in the description.
3. Known pitfalls (auth + retry + security): recursion on 401-from-refresh, token leakage into error logs, race between two concurrent 401s triggering two refreshes. Only the first is partially addressed.
4. Stale-training check: N/A — standard `net/http` usage, verified against `go.mod`.
5. Hand-rolled vs. off-the-shelf: the backoff loop at `client.go:88–104` duplicates behavior available in `golang.org/x/time/rate`, already in `go.mod`.

### Behavior Findings

#### [Blocking] Refresh recursion when /refresh itself returns 401
- **Where:** `internal/auth/client.go:72`
- **Behavior today on this branch:** On a 401, `RoundTrip` calls `refreshToken`, which issues its request through the same `RoundTrip`. If `/refresh` returns 401, the wrapper re-enters refresh and recurses without bound.
- **Why that's a problem:** A single expired refresh token takes the whole process down. Reproducible by revoking the refresh token server-side.
- **Suggested change:** Tag the refresh request with a context value (e.g. `ctxKeyRefreshInFlight`) and short-circuit to error instead of refreshing again. Confirm this direction works for your flow.
- **Confidence:** high — traced through `RoundTrip` → `refreshToken` → `doRequest` on this branch.

#### [Should-fix] Two concurrent 401s trigger two refreshes
- **Where:** `internal/auth/client.go:68`
- **Behavior today on this branch:** No singleflight around refresh. N concurrent failing requests produce N refresh calls; each overwrites the prior token.
- **Why that's a problem:** Under load, tokens churn and some requests see spurious auth failures after a "successful" refresh.
- **Suggested change:** Wrap `refreshToken` with `golang.org/x/sync/singleflight` keyed by user ID. The dep is already in `go.mod`.
- **Confidence:** medium — the race is clear from the code, but I did not measure how often concurrent 401s occur in practice.

#### [Consider] Refresh error includes the full token in the log line
- **Where:** `internal/auth/client.go:83`
- **Behavior today on this branch:** `log.Printf("refresh failed: %s (token=%s)", err, tok)` writes the full access token into whatever sink `log` is wired to.
- **Why that's a problem:** Secrets land in log aggregation.
- **Suggested change:** Log only `tok[:6]+"…"` or drop the token field.
- **Confidence:** high — literal string in the diff.

### What's working well
- Clear separation between `RoundTrip` and `refreshToken` makes the recursion guard above a local fix.
- `TestRoundTrip_RefreshOn401` exercises the happy path cleanly.

### Nits
- `client.go:54` — `err != nil && err == nil` looks like a merge artifact.
- `client.go:91` — magic number `3` for max retries; pull into a named const.

### Open Questions
- Is this `RoundTrip` reused by the mobile SDK? I did not find callers outside this repo.

---

> **Note:** This review was generated automatically by Claude Code. It is not a substitute for human judgment. Treat every finding — including severity labels — as a prompt for the author and human reviewers to verify, not as a final verdict. Approvals do not waive human review.

<!--
Generated by devpilot-pr-review (devpilot v0.12.2)
Model: claude-opus-4-7
Review mode: request-changes
-->
</example_review>

## Step 4 — Post the Review to the PR

Show the user the drafted review first, then post it. Default flow:

- **GitHub** — post as a PR review (not a loose comment) so the review state is visible:
  - Any Blocking finding → `gh pr review <url> --request-changes --body-file -`
  - Only Should-fix / Consider / Nit findings → `gh pr review <url> --comment --body-file -`
  - Zero findings → `gh pr review <url> --approve --body-file -`
  - Pipe the rendered markdown via stdin (`printf '%s' "$review" | gh pr review ...`) to avoid shell-quoting issues.
- **GitLab** — `glab mr note <id> --message "$review"` (GitLab has no request-changes state; severity lives inside the body).

Skip posting when any of these hold, and say so explicitly to the user:
- User opted out ("don't post", "just draft", "local only", "dry run").
- The review is on a patch pasted into chat with no real PR behind it.
- The PR is already merged or closed.

**Inline comments (opt-in):** For line-level feedback, use the GitHub review API:
`gh api -X POST /repos/{owner}/{repo}/pulls/{num}/reviews -f event=... -f body=... -F 'comments[]=...'`
with each item as `{path, line, side, body}`. Use this only when the user asks for inline comments.

## Cross-References

- Code quality at the naming / function / class level → `devpilot-clean-code-principles`.
- Go-specific idiom review → `devpilot-google-go-style`.
- Defer to those skills rather than duplicating their content.

## Rationalization Table

Common shortcuts and what to do instead. The "Reality" column is the rule.

| Excuse | Reality |
|---|---|
| "This PR is small, skip the five questions." | Small PRs change defaults, delete branches, flip flags. Answer all five; "N/A" is fine. |
| "The author explained intent in the description." | Intent ≠ actual behavior. Trace one input through the code. |
| "Diff looks clean, no need to look at callers." | Diff shows what changed, not the blast radius. Check Step 1.2. |
| "I recognize the pattern, I can assert the right way." | Training can be 6–18 months stale. Do Step 1.4 first. |
| "Writing a retry loop / cache / parser here is fine." | Usually there is a mature off-the-shelf option. Do Step 1.5. |
| "LGTM, nothing jumps out." | Trace at least one input through at least one change first. |
| "This feels minor, I'll leave it out to keep the review tidy." | Report it with Confidence + Severity labels. Filtering happens downstream, not here. |
| "I'm unsure, I'll file it as Should-fix to be safe." | Severity is impact-if-true; Confidence is how sure you are. Keep them separate. |
| "I'll just print the review in chat, user can paste it." | Post by default. Only skip when the user opts out or no real PR exists. |
| "Blocking finding, but I'll post as `--comment` to be polite." | Post mode follows severity: a Blocking finding goes with `--request-changes`. |
| "A small emoji softens the tone." | Keep the review in professional prose. The review is part of the PR record. |
| "Greeting feels redundant, skipping it." | The greeting is part of the template and addresses the author by handle. |
| "The version comment is noise, I'll drop it." | Keep `<!-- devpilot-pr-review (devpilot vX.Y.Z) -->` so readers can attribute the review. |
| "Disclaimer feels defensive, I'll skip or shorten it." | Keep the disclaimer. It protects authors from treating AI findings as authoritative. |
| "I'll ask 'what happens when X?' so the author clarifies." | If the code can answer it, state the answer. Author questions live in Open Questions only. |
| "I have a better approach but I'll stay neutral." | Name it, one sentence on why, ask the author to confirm. |

## Self-check before posting

Before sending `gh pr review`, run through this list. A "yes" on any item means the review is not ready.

- Writing line comments before the five questions were answered.
- Findings are all naming / formatting / "could be cleaner".
- Comparing two options the author already listed instead of surfacing ones they did not consider.
- "LGTM" without a single traced input.
- Only files in the diff were opened; no callers, tests, or configs.
- Author questions about behavior the code could have answered.
- Known-better alternatives hidden as vague questions.
- Findings missing a `Confidence` line.

Fix the underlying issue, then re-check.
