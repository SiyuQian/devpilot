## MODIFIED Requirements

### Requirement: CLI review command
The system SHALL provide a `devpilot review <pr-url>` command that performs an AI-powered code review on the specified pull request, displaying real-time streaming progress during execution, and by default posts the review results to the PR as a GitHub review.

#### Scenario: Review a GitHub PR
- **WHEN** user runs `devpilot review https://github.com/owner/repo/pull/123`
- **THEN** the system shows real-time progress on stderr, streams review text to stdout as it is generated, posts the review to the PR as a GitHub review, and exits with code 0 on success

#### Scenario: Review with custom model
- **WHEN** user runs `devpilot review <pr-url> --model claude-sonnet-4-6-20250514`
- **THEN** the system uses the specified model instead of the default Opus 4.6

#### Scenario: Review with model from config
- **WHEN** user runs `devpilot review <pr-url>` and `.devpilot.yaml` has `models.review: claude-sonnet-4-6-20250514`
- **THEN** the system uses the config model when no `--model` flag is provided

#### Scenario: Dry run review
- **WHEN** user runs `devpilot review <pr-url> --dry-run`
- **THEN** the system prints the assembled prompt to stdout without executing Claude (no streaming, no posting), and the prompt SHALL reflect `--no-post` if set (omitting posting instructions)

#### Scenario: Default timeout
- **WHEN** user runs `devpilot review <pr-url>` without `--timeout`
- **THEN** the system uses a default timeout of 10 minutes

#### Scenario: Invalid PR URL
- **WHEN** user runs `devpilot review not-a-url`
- **THEN** the system exits with an error message indicating an invalid PR URL

#### Scenario: No-post flag skips GitHub posting
- **WHEN** user runs `devpilot review <pr-url> --no-post`
- **THEN** the system performs the review and streams output to the terminal but does NOT post to the PR

## ADDED Requirements

### Requirement: Restricted tool access for review
The review executor SHALL use a restricted `--allowedTools` list instead of `--allowedTools=*`.

#### Scenario: Review executor permitted tools
- **WHEN** the review executor is constructed
- **THEN** `--allowedTools` SHALL be set to `Read,Grep,Glob,Bash` (no Write or Edit)

#### Scenario: Claude can still run gh commands
- **WHEN** Claude needs to run `gh pr diff`, `gh pr view`, `gh api`, or `git clone` during review
- **THEN** these commands succeed because Bash is in the allowed tools list
