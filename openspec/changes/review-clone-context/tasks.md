## 1. Update review prompt

- [x] 1.1 Add clone/update instructions to `review-prompt.md` — tell Claude to clone repo to `/tmp/{owner}-{repo}`, or fetch+checkout if already exists
- [x] 1.2 Add context discovery instructions to `review-prompt.md` — tell Claude to search for convention files (CLAUDE.md, AGENTS.md, linter configs, etc.) and read them before reviewing

## 2. Remove Go-side context gathering

- [x] 2.1 Delete `internal/review/context.go`
- [x] 2.2 Update `BuildPrompt` in `prompt.go` to remove `*ProjectContext` parameter, remove convention file embedding logic
- [x] 2.3 Update `Review()` in `review.go` to remove `GatherContext` call
- [x] 2.4 Update `commands.go` to remove dry-run context gathering path

## 3. Update tests

- [x] 3.1 Update `review_test.go` to reflect new `BuildPrompt` signature and verify clone instructions appear in prompt
- [x] 3.2 Run `make test` and `make lint` to verify everything passes
