## 1. Extract Executor and StreamParser to shared package

- [x] 1.1 Create `internal/executor/` package and move `Executor`, `ExecuteResult`, `OutputLine`, `OutputHandler`, `ClaudeEventHandler`, `ExecutorOption`, and all `With*` functions from `internal/taskrunner/executor.go`
- [x] 1.2 Move `ClaudeEvent`, `ParseLine`, and all stream-json parsing types from `internal/taskrunner/streamparser.go` to `internal/executor/`
- [x] 1.3 Update all `internal/taskrunner/` files to import from `internal/executor/` — affected files: `runner.go`, `eventbridge.go`, `reviewer.go`, `executor_test.go`, `streamparser_test.go`, `eventbridge_test.go`, `reviewer_test.go`, `runner_review_test.go`, `runner_test.go`
- [x] 1.4 Verify all existing tests pass with `go test ./internal/taskrunner/... ./internal/executor/...`

## 2. Review prompt and template files

- [x] 2.1 Create `internal/review/review-prompt.md` with comprehensive review instructions covering correctness, security (OWASP top 10), performance, error handling, maintainability, and style — must be project-agnostic (no language-specific assumptions)
- [x] 2.2 Create `internal/review/review-template.md` with structured output format: Summary, Verdict (APPROVE/REQUEST_CHANGES), File-by-File Findings (severity: CRITICAL/WARNING/SUGGESTION/PRAISE + line refs), Overall Assessment
- [x] 2.3 Add `go:embed` declarations in the review package to embed both files

## 3. Core review package

- [x] 3.1 Create `internal/review/review.go` with `Review(ctx, prURL, opts)` function that assembles the prompt and invokes Claude via the shared Executor
- [x] 3.2 Create `internal/review/context.go` with context gathering that fetches convention files from the **target repository** (parsed from PR URL via GitHub API), not from cwd — detect local checkout as optimization
- [x] 3.3 Create `internal/review/prompt.go` with prompt assembly: combine review instructions + template + project context + PR URL into the final prompt
- [x] 3.4 Define `DefaultReviewModel` constant for `claude-opus-4-6-20250415`; configure Executor with `--thinking --model` args (model overridable via options)
- [x] 3.5 Add PR URL validation and parsing: extract `owner`, `repo`, `number` from `https://github.com/{owner}/{repo}/pull/{number}` pattern
- [x] 3.6 Create `internal/review/fix.go` with `Fix(ctx, prURL, opts)` function that reads review comments and addresses them
- [x] 3.7 Create `internal/review/verdict.go` with `IsApproved(stdout string) bool` that parses the structured verdict section for `APPROVE` (replaces the old `"No issues found"` check)
- [x] 3.8 Write unit tests for: context detection (local vs remote), prompt assembly, URL validation/parsing, verdict parsing (`IsApproved` with APPROVE/REQUEST_CHANGES/malformed output), and GitHub API fetch failure graceful degradation

## 4. CLI command

- [x] 4.1 Create `internal/review/commands.go` with `RegisterCommands` following the existing pattern (flags: `--model`, `--dry-run`, `--timeout` with 10min default)
- [x] 4.2 Register the review command in `cmd/devpilot/main.go`
- [x] 4.3 Add `models.review` support in `project.Config.ModelFor()`

## 5. Runner integration

- [x] 5.1 Update `internal/taskrunner/reviewer.go` to delegate `Review()` to `internal/review.Review()`
- [x] 5.2 Update `internal/taskrunner/reviewer.go` to delegate `Fix()` to `internal/review.Fix()`
- [x] 5.3 Replace `IsApproved()` in `internal/taskrunner/reviewer.go` with a call to `review.IsApproved()` — this is critical for the runner's self-heal loop (runner.go:271)
- [x] 5.4 Pass runner's timeout and executor options through to the review package
- [x] 5.5 Verify runner review tests pass with `go test ./internal/taskrunner/... -run Review`

## 6. Final verification

- [x] 6.1 Run `make test` — all tests pass
- [x] 6.2 Run `make lint` — no lint errors
- [x] 6.3 Manual test: `devpilot review <test-pr-url> --dry-run` outputs assembled prompt with correct context from the target repo
