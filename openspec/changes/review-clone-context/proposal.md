## Why

The current review context gathering (`GatherContext`) uses hardcoded file lists and up to 10 GitHub API calls per review, wasting rate limit and silently failing on permission/auth errors. It also cannot discover convention files beyond the hardcoded list. By delegating repo cloning and context discovery to Claude itself, we eliminate API overhead, get proper error handling (clone fails loudly on no access), and allow Claude to dynamically find relevant convention files.

## What Changes

- Remove the Go-side context gathering logic (`GatherContext`, `gatherFromLocal`, `gatherFromGitHub`, `isLocalCheckout`, `getBaseBranch`, `fetchFileFromGitHub`, `ProjectContext`, `ConventionFile` types)
- Delete `context.go` entirely
- Update `BuildPrompt` to no longer accept `ProjectContext` — instead embed clone + search instructions in the review prompt
- Update `Review()` to no longer call `GatherContext`
- The review prompt instructs Claude to: clone the repo to `/tmp/{owner}-{repo}` (or update if already cloned), checkout the PR's base branch, and search for convention/config files itself

## Capabilities

### New Capabilities
- `review-clone-context`: Claude clones the target repo to a temp directory and autonomously discovers project context files, replacing the hardcoded Go-side gathering

### Modified Capabilities
- `review-command`: Remove context gathering requirement scenarios (remote fetch, local checkout detection, API failure handling) — context is now Claude's responsibility
- `review-prompt`: Prompt assembly no longer includes a pre-gathered project context section; instead includes instructions for Claude to clone and search

## Impact

- `internal/review/context.go` — deleted
- `internal/review/review.go` — simplified (remove GatherContext call)
- `internal/review/prompt.go` — simplified (remove ProjectContext parameter), add clone instructions
- `internal/review/commands.go` — simplified (remove dry-run context gathering)
- `internal/review/review_test.go` — update tests
- `internal/review/review-prompt.md` — add clone + context discovery instructions
