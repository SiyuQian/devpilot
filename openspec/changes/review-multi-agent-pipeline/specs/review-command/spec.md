## MODIFIED Requirements

### Requirement: CLI review command
The system SHALL provide a `devpilot review <pr-url>` command that performs an AI-powered code review using a multi-round pipeline, displaying real-time streaming progress during execution.

#### Scenario: Review a GitHub PR
- **WHEN** user runs `devpilot review https://github.com/owner/repo/pull/123`
- **THEN** the system gathers context, runs the review pipeline (Round 1 review + Round 2 scoring + filtering), streams progress to stderr, outputs the filtered review to stdout, and exits with code 0 on success

#### Scenario: Review with custom review model
- **WHEN** user runs `devpilot review <pr-url> --model claude-sonnet-4-6-20250514`
- **THEN** the system uses the specified model for Round 1 (review) instead of the default Sonnet

#### Scenario: Review with custom scoring model
- **WHEN** user runs `devpilot review <pr-url> --scoring-model claude-haiku-4-5-20251001`
- **THEN** the system uses the specified model for Round 2 (scoring) instead of the default Haiku

#### Scenario: Review with custom threshold
- **WHEN** user runs `devpilot review <pr-url> --threshold 75`
- **THEN** only findings scoring ≥ 75 are included in the output

#### Scenario: Review with model from config
- **WHEN** user runs `devpilot review <pr-url>` and `.devpilot.yaml` has `models.review: claude-sonnet-4-6-20250514`
- **THEN** the system uses the config model when no `--model` flag is provided

#### Scenario: Dry run review
- **WHEN** user runs `devpilot review <pr-url> --dry-run`
- **THEN** the system prints the assembled Round 1 prompt to stdout without executing Claude

#### Scenario: Default timeout
- **WHEN** user runs `devpilot review <pr-url>` without `--timeout`
- **THEN** the system uses a default timeout of 10 minutes for the entire pipeline

#### Scenario: Invalid PR URL
- **WHEN** user runs `devpilot review not-a-url`
- **THEN** the system exits with an error message indicating an invalid PR URL

### Requirement: Context gathering
The system SHALL gather project context from the **target repository** (identified by the PR URL, not the user's cwd) in Go code before invoking Claude.

#### Scenario: Remote repo with CLAUDE.md
- **WHEN** the PR's target repository contains a `CLAUDE.md` file on its base branch
- **THEN** the system fetches it via `gh api repos/{owner}/{repo}/contents/CLAUDE.md?ref={base}` and includes its content in the Round 1 prompt as project conventions

#### Scenario: Local repo matches PR target
- **WHEN** the user's cwd is a git checkout of the same repository as the PR
- **THEN** the system reads convention files from disk instead of fetching via GitHub API

#### Scenario: Remote repo with linter config
- **WHEN** the target repository contains linter configuration files on its base branch
- **THEN** the system notes the detected linter in the Round 1 prompt

#### Scenario: No project conventions detected
- **WHEN** the target repository has no recognized convention files
- **THEN** the system proceeds with the review using only the PR diff and general review criteria

#### Scenario: GitHub API fetch failure
- **WHEN** convention file fetching fails
- **THEN** the system logs a warning and proceeds without project context
