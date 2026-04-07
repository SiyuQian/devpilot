## ADDED Requirements

### Requirement: CLI review command
The system SHALL provide a `devpilot review <pr-url>` command that performs an AI-powered code review on the specified pull request.

#### Scenario: Review a GitHub PR
- **WHEN** user runs `devpilot review https://github.com/owner/repo/pull/123`
- **THEN** the system fetches PR context, assembles the review prompt, executes Claude with thinking mode, and outputs the structured review to stdout

#### Scenario: Review with custom model
- **WHEN** user runs `devpilot review <pr-url> --model claude-sonnet-4-6-20250514`
- **THEN** the system uses the specified model instead of the default Opus 4.6

#### Scenario: Review with model from config
- **WHEN** user runs `devpilot review <pr-url>` and `.devpilot.yaml` has `models.review: claude-sonnet-4-6-20250514`
- **THEN** the system uses the config model when no `--model` flag is provided

#### Scenario: Dry run review
- **WHEN** user runs `devpilot review <pr-url> --dry-run`
- **THEN** the system prints the assembled prompt to stdout without executing Claude

#### Scenario: Default timeout
- **WHEN** user runs `devpilot review <pr-url>` without `--timeout`
- **THEN** the system uses a default timeout of 10 minutes

#### Scenario: Invalid PR URL
- **WHEN** user runs `devpilot review not-a-url`
- **THEN** the system exits with an error message indicating an invalid PR URL

### Requirement: Context gathering
The system SHALL gather project context from the **target repository** (identified by the PR URL, not the user's cwd) before invoking the review, because the review command may be run from any directory against any project's PR.

#### Scenario: Remote repo with CLAUDE.md
- **WHEN** the PR's target repository contains a `CLAUDE.md` file on its base branch
- **THEN** the system fetches it via GitHub API (`gh api repos/{owner}/{repo}/contents/CLAUDE.md?ref={base}`) and includes its content in the review prompt as project conventions

#### Scenario: Local repo matches PR target
- **WHEN** the user's cwd is a git checkout of the same repository as the PR
- **THEN** the system reads convention files from disk (faster than API) instead of fetching via GitHub API

#### Scenario: Remote repo with linter config
- **WHEN** the target repository contains linter configuration files (`.golangci.yml`, `.eslintrc.*`, `pyproject.toml`) on its base branch
- **THEN** the system notes the detected linter in the review prompt so Claude can check against project lint rules

#### Scenario: No project conventions detected
- **WHEN** the target repository has no recognized convention files
- **THEN** the system proceeds with the review using only the PR diff and general review criteria

#### Scenario: GitHub API fetch failure
- **WHEN** convention file fetching via GitHub API fails (404, auth error, rate limit)
- **THEN** the system logs a warning and proceeds with the review without project context (graceful degradation)

### Requirement: Executor extraction
The system SHALL extract the `Executor`, `ExecuteResult`, `OutputLine`, `OutputHandler`, `ClaudeEventHandler`, `ExecutorOption`, all `With*` functions from `internal/taskrunner/executor.go`, and all stream-json parsing types and `ParseLine` from `internal/taskrunner/streamparser.go` into `internal/executor/` so both the review and task runner packages can share them.

#### Scenario: Task runner continues to work after extraction
- **WHEN** the Executor and StreamParser types are moved to `internal/executor/`
- **THEN** `internal/taskrunner/` imports from `internal/executor/` and all existing task runner tests pass (including eventbridge, streamparser, and runner tests)

#### Scenario: Review package uses shared executor
- **WHEN** `internal/review/` invokes a Claude review
- **THEN** it uses `internal/executor.Executor` with thinking mode args

### Requirement: Runner integration
The system SHALL update the task runner's review step to delegate to the new `internal/review/` package, including both Review and Fix functionality.

#### Scenario: Runner uses new review package
- **WHEN** the task runner executes a code review after PR creation
- **THEN** it calls `review.Review()` instead of the old inline `Reviewer.Review()`
- **THEN** the review uses the same high-quality prompt and context gathering as the standalone command

#### Scenario: Runner self-heal uses new fix
- **WHEN** the review finds issues and retries remain in the self-heal loop
- **THEN** the runner calls `review.Fix()` from the new review package instead of the old `Reviewer.Fix()`

#### Scenario: Verdict detection uses structured output
- **WHEN** the runner checks if a review passed
- **THEN** it calls `review.IsApproved()` which parses the structured verdict section for `APPROVE` instead of checking for `"No issues found"`

#### Scenario: Runner review timeout
- **WHEN** the runner's `--review-timeout` is set to 10 minutes
- **THEN** the review execution respects the timeout via context cancellation
