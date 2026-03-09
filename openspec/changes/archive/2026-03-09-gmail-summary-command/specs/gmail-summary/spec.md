## ADDED Requirements

### Requirement: Summarize today's unread emails
The system SHALL provide a `devpilot gmail summary` command that fetches today's unread emails, generates an AI-powered summary via `claude -p`, and marks them as read.

#### Scenario: Normal summary with unread emails
- **WHEN** user runs `devpilot gmail summary` and there are unread emails from today
- **THEN** the system SHALL fetch all unread emails for today, send them to `claude -p` for summarization, print the summary to stdout, and mark all processed emails as read

#### Scenario: No unread emails today
- **WHEN** user runs `devpilot gmail summary` and there are no unread emails from today
- **THEN** the system SHALL print "No unread emails for today." and exit successfully

#### Scenario: Not logged in
- **WHEN** user runs `devpilot gmail summary` without being logged in to Gmail
- **THEN** the system SHALL print "Not logged in to Gmail. Run: devpilot login gmail" and exit with error

#### Scenario: claude CLI not available
- **WHEN** user runs `devpilot gmail summary` and `claude` is not on PATH
- **THEN** the system SHALL print an error indicating Claude Code CLI is required and exit with error

#### Scenario: claude -p fails
- **WHEN** `claude -p` returns an error or produces empty output
- **THEN** the system SHALL print the error to stderr and exit without marking emails as read

### Requirement: Send summary to Slack channel
The system SHALL support sending the summary to a Slack channel via the `--channel` flag.

#### Scenario: Send to channel
- **WHEN** user runs `devpilot gmail summary --channel daily-digest`
- **THEN** the system SHALL send the summary to the `daily-digest` Slack channel via `devpilot slack send` in addition to printing to stdout

#### Scenario: Slack send failure
- **WHEN** the Slack send fails
- **THEN** the system SHALL print the error to stderr but still mark emails as read (since the summary was produced)

### Requirement: Send summary as DM
The system SHALL support sending the summary as a direct message via the `--dm` flag.

#### Scenario: Send as DM
- **WHEN** user runs `devpilot gmail summary --dm U0123ABCDE`
- **THEN** the system SHALL send the summary as a DM to the specified Slack user ID via `devpilot slack send --channel U0123ABCDE`

### Requirement: Preview mode
The system SHALL support a `--no-mark-read` flag that skips marking emails as read.

#### Scenario: Preview without marking
- **WHEN** user runs `devpilot gmail summary --no-mark-read`
- **THEN** the system SHALL fetch and summarize emails, print the summary to stdout, but NOT mark any emails as read

### Requirement: Date-scoped fetching
The system SHALL automatically scope email fetching to today's date.

#### Scenario: Only today's emails
- **WHEN** user runs `devpilot gmail summary`
- **THEN** the system SHALL use the Gmail query `is:unread after:YYYY/MM/DD` where the date is today, fetching all matching emails without an arbitrary limit

### Requirement: Email body truncation
The system SHALL truncate long email bodies before sending to the LLM.

#### Scenario: Long email body
- **WHEN** an email body exceeds 1000 characters
- **THEN** the system SHALL truncate the body to 1000 characters and append "[truncated]" before including it in the summarization prompt
