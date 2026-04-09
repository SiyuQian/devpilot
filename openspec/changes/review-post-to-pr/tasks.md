## 1. Posting Instructions

- [x] 1.1 Create `internal/review/review-posting.md` with instructions for Claude to post review as GitHub PR review via `gh api` (APPROVE or COMMENT event, summary body, inline comments with severity tags)
- [x] 1.2 Add `//go:embed review-posting.md` var in `internal/review/embed.go`

## 2. Prompt Assembly

- [x] 2.1 Add `postToGitHub` bool parameter to `BuildPrompt` (or a new `BuildPromptWithPosting` function) that conditionally appends posting instructions
- [x] 2.2 Add `WithPostToGitHub(bool)` option to review options
- [x] 2.3 Wire `postToGitHub` option from `Review()` through `resolveOptions` to `BuildPrompt` call

## 3. Tool Restrictions

- [x] 3.1 Change `--allowedTools=*` to `--allowedTools=Read,Grep,Glob,Bash` in `newReviewExecutor`

## 4. CLI Flag

- [x] 4.1 Add `--no-post` flag to `reviewCmd` in `internal/review/commands.go`
- [x] 4.2 Pass the posting option through to `Review()` call based on flag value (default: post enabled)
- [x] 4.3 Wire `--no-post` into dry-run path so `BuildPrompt` omits posting instructions when both flags are set

## 5. Testing

- [x] 5.1 Add test for `BuildPrompt` with posting enabled — verify posting instructions are included
- [x] 5.2 Add test for `BuildPrompt` with posting disabled — verify posting instructions are omitted
- [x] 5.3 Add test that `newReviewExecutor` args contain `--allowedTools=Read,Grep,Glob,Bash` (not `*`)
- [x] 5.4 Add test for `--no-post` flag parsing
- [x] 5.5 Update CLAUDE.md CLI commands section to document `--no-post` flag
