## Why

`devpilot review` produces structured code review output but only displays it in the terminal. The review results are not visible on the GitHub PR itself, so team members must run the tool locally to see findings. Posting the review directly to the PR as a GitHub review with inline comments makes the feedback visible to everyone and integrates into the normal PR workflow.

## What Changes

- Add prompt instructions for Claude to post review results to the PR via `gh api` after completing the review
- Tighten the `--allowedTools` flag on the review executor to only permit read and shell tools (no Write/Edit), since review should be a read-only operation
- Design a GitHub review comment template: a summary body + inline comments for each finding
- Add `--no-post` flag to `devpilot review` to skip posting (default: post to PR)
- When verdict is APPROVE, submit as GitHub `APPROVE` review; when verdict is REQUEST_CHANGES, submit as `COMMENT` review (never `REQUEST_CHANGES` status, to avoid blocking PRs)

## Capabilities

### New Capabilities
- `review-github-posting`: Instructions and template for Claude to post review results to the PR as a GitHub review with inline comments via `gh api`

### Modified Capabilities
- `review-command`: Add `--no-post` flag, tighten `--allowedTools` to exclude Write/Edit
- `review-prompt`: Add posting instructions section; update review template to include inline-comment-friendly output format

## Impact

- `internal/review/review-prompt.md` — add posting instructions
- `internal/review/review-template.md` — update output format to support inline comments
- `internal/review/review.go` — update executor args (`--allowedTools`)
- `internal/review/commands.go` — add `--no-post` flag
- No impact on `devpilot run` (runner mode unchanged)
