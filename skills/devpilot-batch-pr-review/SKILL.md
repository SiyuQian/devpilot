---
name: devpilot-batch-pr-review
description: Use when the user wants to review every pull request currently requesting their review, batch review a review queue, sweep pending PR reviews, or run PR reviews across all review-requested pull requests. Discovers the queue with `devpilot github prs review-queue` and delegates each real review to `devpilot-pr-review`; do not use for reviewing a single specified PR.
---

# Batch PR Review

## Overview

Find the current GitHub PR review queue, normalize it into explicit PR targets, then invoke `devpilot-pr-review` once per eligible PR. This skill owns discovery, ordering, and progress accounting only.

Do not perform the review yourself. The actual review workflow, posting behavior, eligibility gate, inline comments, and confidence filtering belong to `devpilot-pr-review`.

## Workflow

1. Confirm GitHub CLI auth is available:

   ```bash
   gh auth status
   ```

   If auth is missing, stop and ask the user to run `gh auth login`.

2. Discover PRs requesting review:

   ```bash
   devpilot github prs review-queue --user @me --limit 200 --json
   ```

   Add filters only when the user asks or context requires them:

   ```bash
   devpilot github prs review-queue --user <login-or-@me> --owner <owner> --json
   devpilot github prs review-queue --user <login-or-@me> --repo <owner/repo> --json
   devpilot github prs review-queue --user <login-or-@me> --direct --json
   ```

3. Parse the JSON array. Each item is expected to include `url`, `repository.nameWithOwner`, `number`, `title`, `author.login`, `isDraft`, `state`, and timestamps.

4. Apply queue-level filtering:

   - Keep only open PRs.
   - Skip drafts unless explicitly requested with an include-drafts instruction.
   - Preserve the command order unless the user asks for a different order.
   - If the command returns no PRs, report that the review queue is empty and stop.

5. Review each remaining PR by invoking `devpilot-pr-review` with the PR URL:

   ```text
   Use devpilot-pr-review to review <pr-url>.
   ```

   Run one PR at a time by default so review posting, CI context, and failures are easy to attribute. If the user explicitly asks for parallelism, dispatch reviews in parallel only when the review tooling and repository rate limits make that safe.

6. Maintain a short progress ledger:

   ```text
   queued: <n>
   reviewed: <n>
   skipped: <n> (<reason summary>)
   failed: <n> (<pr-url>: <reason>)
   ```

   Continue after a single PR review fails unless the failure indicates a shared blocker such as missing GitHub auth, missing `devpilot`, or repository-wide permissions.

## Command Notes

- Prefer `devpilot github prs review-queue --json` over raw `gh search prs`; PR #148 added the normalized helper for this exact queue-discovery task.
- Use `--user @me` by default. Use a concrete login only when the user names someone else.
- Use `--direct` only when team review requests should be excluded.
- Use `--include-drafts` only when the user explicitly wants draft PRs included; otherwise skip drafts unless they appear because an older helper version did not filter them out.
- In short: skip drafts unless explicitly requested.

## Handoff Contract

For each PR handed to `devpilot-pr-review`, pass only the PR URL plus any user-specified review constraints that apply to every PR in the batch. Do not pass the whole queue JSON unless the single-PR review needs it.

Example handoff:

```text
Use devpilot-pr-review to review https://github.com/SiyuQian/devpilot/pull/148.
Apply the user's batch constraint: focus on correctness and missing tests.
```

## Final Report

End with a concise queue summary:

- PRs discovered and reviewed.
- PRs skipped with reasons.
- PRs that failed and the next action needed.
- Link or identify any reviews posted by `devpilot-pr-review` when that skill reports them.
