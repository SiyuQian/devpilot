## ADDED Requirements

### Requirement: Real-time progress display
The system SHALL display real-time progress to stderr while a review is executing, so the user can see that work is in progress.

#### Scenario: Tool call progress
- **WHEN** Claude invokes a tool during the review (e.g., `gh pr diff`, `Read`)
- **THEN** the system prints the tool name to stderr (e.g., `[tool] gh pr diff`)

#### Scenario: Thinking indicator
- **WHEN** Claude produces a thinking block during the review
- **THEN** the system prints a brief indicator to stderr (e.g., `[thinking] ...`)

#### Scenario: Text output streaming
- **WHEN** Claude produces text content during the review
- **THEN** the system streams the text to stdout in real-time instead of buffering until completion

#### Scenario: Non-TTY output
- **WHEN** stdout is not a TTY (e.g., piped to a file)
- **THEN** the system still streams text to stdout but suppresses progress indicators on stderr

### Requirement: Readable final output
The system SHALL output the review as human-readable text, not raw stream-json.

#### Scenario: Normal review output
- **WHEN** a review completes successfully
- **THEN** stdout contains the review text extracted from Claude's assistant message content blocks, not raw JSON

#### Scenario: Piped output
- **WHEN** user runs `devpilot review <url> > review.md`
- **THEN** the file contains only the review text, with no progress indicators mixed in
