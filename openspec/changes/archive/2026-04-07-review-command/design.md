## Context

DevPilot's current review capability is a thin wrapper in `internal/taskrunner/reviewer.go` that passes `"Code review: <url>"` to the Executor. This produces shallow reviews because:
1. No structured review criteria or prompt engineering
2. No project context gathering (conventions, CLAUDE.md, linter config)
3. No thinking mode — Claude can't reason deeply about complex code
4. Tightly coupled to the task runner — can't be used standalone

The Executor (`internal/taskrunner/executor.go`) handles Claude CLI invocation with streaming output, process groups, and timeout management. It defaults to `claude -p --verbose --output-format stream-json --allowedTools=*`.

## Goals / Non-Goals

**Goals:**
- Standalone `devpilot review <pr-url>` command usable on any project
- High-quality reviews via Opus 4.6 with extended thinking
- Structured, consistent review output via a comment template
- Rich context gathering: PR diff, changed files, project conventions
- Drop-in replacement for the runner's review step
- Configurable model via `--model` flag and `.devpilot.yaml`

**Non-Goals:**
- Posting review comments directly to GitHub (out of scope — output to stdout or let Claude use `gh`)
- Supporting non-GitHub PRs (GitLab, Bitbucket)
- Review of local uncommitted changes (PR URL required)
- Building a review TUI dashboard

## Decisions

### 1. New `internal/review/` package

**Decision**: Create a dedicated `internal/review/` package rather than extending `internal/taskrunner/reviewer.go`.

**Rationale**: The review command is a standalone tool usable on any project. Coupling it to the task runner package creates an unwanted dependency. The runner's `Reviewer` will be updated to delegate to this new package.

**Alternative considered**: Extending `reviewer.go` in-place — rejected because it would force standalone users to import the entire taskrunner package.

### 2. Extract Executor and StreamParser to shared packages

**Decision**: Move `Executor`, `ExecuteResult`, `OutputLine`, `OutputHandler`, `ClaudeEventHandler`, `ExecutorOption`, and all `With*` functions from `internal/taskrunner/executor.go` to `internal/executor/`. Move `ClaudeEvent`, `ParseLine`, and all stream-json parsing types from `internal/taskrunner/streamparser.go` to `internal/executor/` as well (they are consumed by the executor's streaming handler).

**Affected files** (must update imports):
- `internal/taskrunner/runner.go` — uses `Executor`, `ExecutorOption`, `NewExecutor`, `WithClaudeEventHandler`
- `internal/taskrunner/eventbridge.go` — uses `ClaudeEvent`, all parsed event types
- `internal/taskrunner/executor_test.go` — tests for Executor
- `internal/taskrunner/streamparser_test.go` — tests for ParseLine and event types
- `internal/taskrunner/eventbridge_test.go` — tests that construct ClaudeEvent instances
- `internal/taskrunner/reviewer.go` — uses `Executor`, `ExecutorOption`, `NewExecutor`
- `internal/taskrunner/reviewer_test.go`, `runner_review_test.go`, `runner_test.go` — use `ExecuteResult`, `WithCommand`

**Rationale**: The Executor and stream parser are generic Claude CLI wrappers with no task-runner-specific logic. Both `review` and `taskrunner` need them. Keeping them together in `internal/executor/` avoids a separate `internal/streamparser/` package since the executor's streaming handler is their primary consumer.

**Alternative considered**: Duplicate the executor in `internal/review/` — rejected due to code duplication. Separate `internal/streamparser/` package — rejected as over-splitting since the types are tightly coupled to the executor's streaming pipeline.

### 3. Claude invocation with thinking mode

**Decision**: Use `claude -p --thinking --model claude-opus-4-6-20250415 --verbose --output-format stream-json --allowedTools=*` for review execution.

**Rationale**: Extended thinking enables deeper reasoning about code correctness, edge cases, and architectural concerns. Opus 4.6 is the most capable model for nuanced review. The user explicitly requested this configuration.

The model is configurable: `--model` flag > `.devpilot.yaml` models.review > default (Opus 4.6). The default model is stored as a package-level constant (`DefaultReviewModel`) so it can be updated in one place when newer models are released. An empty string from config means "use the constant default", not "let Claude CLI pick".

### 4. Prompt assembly architecture

**Decision**: Build the review prompt from three composable parts:

1. **Review instructions file** (`review-prompt.md`): Embedded in the binary via `go:embed`. Contains the review methodology, criteria (correctness, security, performance, maintainability, style), and severity levels. This is the "LLM file" that ensures high-quality reviews.

2. **Comment template** (`review-template.md`): Also embedded. Defines the structured output format — summary, file-by-file findings, severity ratings, and overall verdict (approve/request-changes).

3. **Context block**: Dynamically assembled per review — PR URL, project conventions detected at runtime.

**Rationale**: Separating instructions from template allows either to be improved independently. Embedding ensures the tool works without external file dependencies.

### 5. Context gathering strategy

**Decision**: Before invoking Claude, gather context from the **target repository** (not cwd) and include it in the prompt. This is critical because `devpilot review` may be run from any directory to review any project's PR.

1. **PR metadata**: Parse `owner/repo` from the PR URL, then use `gh pr view <url> --json title,body,baseRefName,headRefName,files` to get PR info
2. **Repository convention files**: Use `gh api` or `gh pr view` to fetch convention files from the PR's base branch in the target repository:
   - `CLAUDE.md` / `AGENTS.md` (project-specific instructions)
   - `.devpilot.yaml` (devpilot config)
   - Linter configs (`.golangci.yml`, `.eslintrc.*`, `pyproject.toml`)
   
   Fetch via GitHub API: `gh api repos/{owner}/{repo}/contents/{path}?ref={base_branch}`. This avoids requiring a local clone of the target repo.
   
   If the target repo happens to be a local checkout (cwd matches the PR's repo), prefer reading from disk for speed. Otherwise fall back to the API approach.
3. **Allowed tools**: Grant Claude access to `gh`, `git`, file reading, and shell commands so it can inspect the PR diff and files itself

**Rationale**: Unlike `devpilot commit` and `devpilot readme` which always operate on the local repo, `devpilot review` targets a remote PR that may belong to any repository. We cannot assume cwd is the target repo. The GitHub API approach ensures context gathering works regardless of where the command is run. Claude also has `--allowedTools=*` so it can fetch additional context itself during review.

### 6. Runner integration

**Decision**: Update `internal/taskrunner/reviewer.go` to import `internal/review/` and delegate to both `Review()` and `Fix()` functions, passing the PR URL and runner's executor options.

The runner's self-heal loop (runner.go:254-311) calls both `Reviewer.Review()` and `Reviewer.Fix()`. Both must be migrated. The `IsApproved()` function must be updated to parse the new structured verdict format — checking for `## Verdict` + `APPROVE` in the review output instead of the current `"No issues found"` string match.

The review package will expose:
- `Review(ctx, prURL, opts) (*executor.ExecuteResult, error)` — run a code review
- `Fix(ctx, prURL, opts) (*executor.ExecuteResult, error)` — fix issues from review comments
- `IsApproved(stdout string) bool` — parse structured verdict from review output

**Rationale**: Minimal change to the runner. The runner continues to manage timeouts and event handling; the review package handles prompt assembly, context, and verdict parsing.

### 7. Dry-run semantics

**Decision**: `--dry-run` for the review command prints the assembled prompt to stdout **without executing Claude**. This intentionally differs from the `generate` commands where dry-run executes Claude but doesn't write files.

**Rationale**: The review command's value is in the prompt quality. Printing the prompt lets users debug and iterate on context gathering without burning tokens. This is the more useful semantic for a review tool.

## Risks / Trade-offs

- **Token cost**: Opus 4.6 + thinking mode is expensive (~10-50x more than a basic review). → Mitigation: Model is configurable via `--model` flag and `.devpilot.yaml`. Users can use Sonnet for cheaper reviews. Runner's `--review-timeout` provides a cost backstop. The `--timeout` flag on the standalone command defaults to 10 minutes.
- **Execution time**: Deep thinking reviews take 2-5 minutes. → Mitigation: Acceptable for quality; runner already has configurable review timeout. Standalone command has its own `--timeout` flag.
- **Context size**: Large PRs may exceed context limits. → Mitigation: We don't stuff the full diff into the prompt. We provide PR URL + `--allowedTools=*` so Claude fetches what it needs incrementally. This lets Claude prioritize which files to inspect deeply.
- **Executor extraction scope**: Moving types to `internal/executor/` touches ~10 files across executor, streamparser, eventbridge, runner, reviewer, and their tests. → Mitigation: Pure mechanical import path changes with no logic changes. Run `make test` after extraction to catch any missed imports. All affected files are enumerated in Decision #2 above.
- **Prompt quality determines review quality**: A bad prompt produces bad reviews regardless of model choice. → Mitigation: Ship a solid v1 prompt based on established code review best practices (OWASP, Clean Code principles). Iterate based on real review output. The prompt is embedded via `go:embed` so updates ship with each binary release.
- **`IsApproved` verdict parsing**: The new structured output format must be reliably parseable. → Mitigation: Use a simple substring check for `APPROVE` in the `## Verdict` section. The template instructs Claude to output exactly one of `APPROVE` or `REQUEST_CHANGES`. Add a unit test with sample outputs to prevent regressions.
