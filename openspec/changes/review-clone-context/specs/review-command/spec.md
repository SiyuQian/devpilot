## REMOVED Requirements

### Requirement: Context gathering
**Reason**: Context gathering is now delegated to Claude via the review prompt. Claude clones the repo and searches for convention files autonomously, replacing the Go-side `GatherContext` logic.
**Migration**: The `review-clone-context` capability replaces this. No API migration needed — the behavior is now handled in the prompt.

## MODIFIED Requirements

### Requirement: CLI review command
The system SHALL provide a `devpilot review <pr-url>` command that performs an AI-powered code review on the specified pull request.

#### Scenario: Review a GitHub PR
- **WHEN** user runs `devpilot review https://github.com/owner/repo/pull/123`
- **THEN** the system assembles the review prompt with clone/search instructions, executes Claude with thinking mode, and outputs the structured review to stdout

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
