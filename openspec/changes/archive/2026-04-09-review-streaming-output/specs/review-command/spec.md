## MODIFIED Requirements

### Requirement: CLI review command
The system SHALL provide a `devpilot review <pr-url>` command that performs an AI-powered code review on the specified pull request, displaying real-time streaming progress during execution.

#### Scenario: Review a GitHub PR
- **WHEN** user runs `devpilot review https://github.com/owner/repo/pull/123`
- **THEN** the system shows real-time progress on stderr, streams review text to stdout as it is generated, and exits with code 0 on success

#### Scenario: Review with custom model
- **WHEN** user runs `devpilot review <pr-url> --model claude-sonnet-4-6-20250514`
- **THEN** the system uses the specified model instead of the default Opus 4.6

#### Scenario: Review with model from config
- **WHEN** user runs `devpilot review <pr-url>` and `.devpilot.yaml` has `models.review: claude-sonnet-4-6-20250514`
- **THEN** the system uses the config model when no `--model` flag is provided

#### Scenario: Dry run review
- **WHEN** user runs `devpilot review <pr-url> --dry-run`
- **THEN** the system prints the assembled prompt to stdout without executing Claude (no streaming)

#### Scenario: Default timeout
- **WHEN** user runs `devpilot review <pr-url>` without `--timeout`
- **THEN** the system uses a default timeout of 10 minutes

#### Scenario: Invalid PR URL
- **WHEN** user runs `devpilot review not-a-url`
- **THEN** the system exits with an error message indicating an invalid PR URL
