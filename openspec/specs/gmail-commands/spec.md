## Requirements

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

### Requirement: Read email content
The system SHALL provide a `devpilot gmail read` command that displays the full content of a specific email.

#### Scenario: Read email by ID
- **WHEN** user runs `devpilot gmail read <message-id>`
- **THEN** the system SHALL display From, Subject, Date headers followed by the email body in plain text

#### Scenario: HTML-only email
- **WHEN** the email has no text/plain part, only text/html
- **THEN** the system SHALL strip HTML tags and display the text content

#### Scenario: Invalid message ID
- **WHEN** user runs `devpilot gmail read <invalid-id>`
- **THEN** the system SHALL return an error: "Message not found: <invalid-id>"

### Requirement: Mark emails as read
The system SHALL provide a `devpilot gmail mark-read` command that marks one or more emails as read.

#### Scenario: Mark single email as read
- **WHEN** user runs `devpilot gmail mark-read <message-id>`
- **THEN** the system SHALL remove the UNREAD label from the message and print "Marked 1 message as read."

#### Scenario: Mark multiple emails as read
- **WHEN** user runs `devpilot gmail mark-read <id1> <id2> <id3>`
- **THEN** the system SHALL remove the UNREAD label from all specified messages and print "Marked 3 messages as read."

#### Scenario: Batch modify via API
- **WHEN** marking more than one message as read
- **THEN** the system SHALL use the Gmail batch modify API endpoint for efficiency rather than individual requests
