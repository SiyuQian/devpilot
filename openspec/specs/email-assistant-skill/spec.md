## Purpose

**NOTE: This skill is NOT YET IMPLEMENTED. The spec below represents the planned design.**

Defines the email-assistant skill that reads unread Gmail messages, generates an intelligent digest summary, and sends it to a Slack channel.

## Requirements

### Requirement: Email digest generation
The skill SHALL read unread Gmail messages, generate an intelligent summary, and send it to a Slack channel.

#### Scenario: Normal digest with mixed emails
- **WHEN** the skill is invoked and there are unread emails
- **THEN** the skill SHALL list unread emails via `devpilot gmail list --unread`, read each email via `devpilot gmail read`, generate a summary grouped by priority (action required vs informational), and send via `devpilot slack send`

#### Scenario: No unread emails
- **WHEN** the skill is invoked and there are no unread emails
- **THEN** the skill SHALL send a brief "No new emails" message to Slack (or skip sending entirely)

#### Scenario: Large volume (over 50 unread)
- **WHEN** there are more than 50 unread emails
- **THEN** the skill SHALL process only the 50 most recent and note the remaining count in the digest

### Requirement: Email classification
The skill SHALL classify emails into categories for the digest summary.

#### Scenario: Action required emails
- **WHEN** an email requires a response, review, or action from the user
- **THEN** the skill SHALL list it under "ACTION REQUIRED" with a brief description of what's needed

#### Scenario: Informational emails
- **WHEN** an email is informational (updates, notifications, FYI)
- **THEN** the skill SHALL list it under "INFORMATIONAL" with a one-line summary

#### Scenario: Promotional/noise emails
- **WHEN** an email is promotional, newsletter, or automated notification
- **THEN** the skill SHALL count them and note "N promotional emails skipped" without individual summaries

### Requirement: Mark processed emails as read
The skill SHALL mark all processed emails as read after successfully sending the digest.

#### Scenario: Mark as read after successful send
- **WHEN** the digest has been successfully sent to Slack
- **THEN** the skill SHALL call `devpilot gmail mark-read` with all processed message IDs

#### Scenario: Do not mark as read on send failure
- **WHEN** the Slack send fails
- **THEN** the skill SHALL NOT mark emails as read, so they appear in the next digest

### Requirement: Slack channel configuration
The skill SHALL accept a target Slack channel as an argument.

#### Scenario: Channel specified as argument
- **WHEN** the skill is invoked with a channel argument (e.g., `/email-assistant --channel daily-digest`)
- **THEN** the skill SHALL send the digest to the specified channel

### Requirement: Cron-compatible execution
The skill SHALL work unattended when invoked via `claude -p`.

#### Scenario: Unattended execution
- **WHEN** the skill is run via `claude -p "/email-assistant --channel daily-digest"`
- **THEN** the skill SHALL complete without requiring user interaction

#### Scenario: Auth error handling
- **WHEN** Gmail or Slack credentials are missing or expired
- **THEN** the skill SHALL output a clear error message indicating which service needs re-authentication

### Requirement: Skill creation method
The skill SHALL be created using the skill-creator skill to ensure proper structure and validation.

#### Scenario: Skill structure
- **WHEN** the skill is created
- **THEN** it SHALL follow the SKILL.md format with YAML frontmatter (name, description, triggers) and markdown body with instructions
