## Context

Currently `devpilot review <pr-url>` runs Claude with `--allowedTools=*` and streams the structured review output to the terminal. The review output includes Summary, Verdict, Findings (with file paths and line numbers), and Overall Assessment. However, nothing is posted to the GitHub PR — the output stays local.

The review executor at `internal/review/review.go:84` passes `--allowedTools=*`, which grants Claude Write/Edit access that a read-only review operation should not have.

## Goals / Non-Goals

**Goals:**
- Claude posts review results directly to the PR as a GitHub review (summary + inline comments) via `gh api`
- Tighten `--allowedTools` to only what review needs (read tools + Bash for `gh`/`git`)
- Add `--no-post` flag to skip posting when user just wants terminal output
- Use APPROVE status when verdict is approve; use COMMENT status otherwise (never REQUEST_CHANGES)

**Non-Goals:**
- Changing the `devpilot run` (runner) review flow — runner still only uses stdout for verdict parsing
- Implementing Go-side parsing of review output to construct API calls
- Adding posting support for non-GitHub platforms (GitLab, Bitbucket)
- Tightening `newFixExecutor` tools — fix needs Write/Edit to modify code, so `--allowedTools=*` is correct there

## Decisions

### Decision 1: Claude posts via `gh api` (prompt-driven, not Go-parsed)

Instead of parsing Claude's structured output in Go and constructing GitHub API calls, we add instructions to the review prompt telling Claude to post the review itself using `gh api`.

**Why:** Claude already knows every finding's file, line, severity, and content. Parsing this in Go would require a brittle parser that breaks if the output format changes. Claude can directly construct the API payload with full fidelity.

**Alternative considered:** Go-side parsing + `gh api` call. Rejected because it adds a fragile parser layer with no benefit — Claude is already the source of truth for the findings.

### Decision 2: Tighten allowedTools to read + Bash only

Change from `--allowedTools=*` to `--allowedTools=Read,Grep,Glob,Bash` (or the equivalent comma-separated list). This removes Write/Edit access since review is a read-only operation.

**Why:** Principle of least privilege. A review tool should not be able to modify code. The only "write" action it takes is posting via `gh api` through Bash.

### Decision 3: Posting instructions as a separate embedded prompt section

Add a new `review-posting.md` embedded file with posting instructions, appended after the existing review prompt and template. The posting section tells Claude:
1. After completing the review, construct a GitHub PR review via `gh api`
2. Use `APPROVE` event when verdict is APPROVE; use `COMMENT` event otherwise
3. Include the Summary + Overall Assessment as the review body
4. Add each finding as an inline comment on the relevant file and line

**Why:** Keeping posting instructions separate from review instructions maintains clean separation of concerns. The review prompt focuses on *how to review*; the posting prompt focuses on *how to publish*.

### Decision 4: `--no-post` flag controls posting

A new `--no-post` boolean flag on `devpilot review`. When set, the posting instructions are omitted from the prompt entirely (not included in the assembled prompt). Default: false (posting enabled).

**Why:** Omitting the instructions entirely is cleaner than adding a conditional "don't post" instruction. It also saves tokens.

### Decision 5: GitHub review API via `gh api`

Claude will use the GitHub PR review API with the newer `line`/`side` parameters (instead of the legacy `position` field):

```
gh api repos/{owner}/{repo}/pulls/{number}/reviews \
  --method POST \
  -f body="..." \
  -f event="APPROVE|COMMENT" \
  -f 'comments[0][path]=...' \
  -f 'comments[0][line]=...' \
  -f 'comments[0][side]=RIGHT' \
  -f 'comments[0][body]=...'
```

The `line` field is the file line number in the diff's RIGHT side (new version), which is much more intuitive than the legacy `position` field (diff hunk offset). Claude already knows file line numbers from the review findings, so no mapping is needed.

**Why `line`/`side` over `position`:** The `position` field requires counting lines within a diff hunk, which is error-prone. The `line`/`side` API (available since 2022) accepts the actual file line number directly.

## Risks / Trade-offs

**[Risk] Claude may fail to post correctly** → The review output is still streamed to terminal regardless, so the user always sees the review. A failed post is a degraded experience, not a lost review. The prompt should instruct Claude to report any posting errors clearly.

**[Risk] Inline comment may land on wrong line** → Using `line`/`side` parameters (file line numbers) instead of `position` (diff offset) significantly reduces this risk. Claude already knows file line numbers from the review. Lines outside the diff range will be rejected by GitHub API — Claude should fall back to a non-inline comment in that case.

**[Risk] Token cost increases slightly** → Claude must read the diff (already does) and make `gh api` calls. The incremental cost is small relative to the review itself.

**[Trade-off] Prompt-driven posting is less deterministic than Go-parsed posting** → Accepted. The benefit of zero parsing code and natural accuracy outweighs the small risk of posting errors.
