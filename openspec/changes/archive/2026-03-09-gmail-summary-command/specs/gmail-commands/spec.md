## MODIFIED Requirements

### Requirement: List unread emails
The system SHALL provide a `devpilot gmail list` command that lists unread emails from the user's inbox.

#### Scenario: List unread emails
- **WHEN** user runs `devpilot gmail list --unread`
- **THEN** the system SHALL display a table with columns: ID, FROM, SUBJECT, DATE for each unread message, sorted by date descending

#### Scenario: List with limit
- **WHEN** user runs `devpilot gmail list --unread --limit 10`
- **THEN** the system SHALL return at most 10 messages

#### Scenario: List with date filter
- **WHEN** user runs `devpilot gmail list --unread --after "2024-01-15"`
- **THEN** the system SHALL only return unread messages received after the specified date

#### Scenario: No unread emails
- **WHEN** user runs `devpilot gmail list --unread` and there are no unread messages
- **THEN** the system SHALL print "No unread messages."

#### Scenario: Not logged in
- **WHEN** user runs `devpilot gmail list` without being logged in
- **THEN** the system SHALL return an error: "Not logged in to Gmail. Run: devpilot login gmail"
