## ADDED Requirements

### Requirement: Go-primary GitHub review posting
The system SHALL post review results to GitHub as a PR review from Go code as the primary path.

#### Scenario: Post review with inline comments
- **WHEN** the review has findings to post and `--no-post` is not set
- **THEN** Go code constructs a GitHub review with body (summary + verdict + assessment) and inline comments for each finding, then posts via `gh api repos/{owner}/{repo}/pulls/{number}/reviews`

#### Scenario: Finding within diff range
- **WHEN** a finding's line number falls within the diff range for its file
- **THEN** it is posted as an inline comment with `line` (RIGHT side) and `side=RIGHT`

#### Scenario: Finding outside diff range
- **WHEN** a finding's line number falls outside the diff range for its file
- **THEN** it is included in the review body text instead of as an inline comment

#### Scenario: Approve event
- **WHEN** the verdict is APPROVE
- **THEN** the review is posted with event `APPROVE`

#### Scenario: Request changes event
- **WHEN** the verdict is REQUEST_CHANGES
- **THEN** the review is posted with event `COMMENT` (not `REQUEST_CHANGES`, to avoid blocking merge for bot reviews)

### Requirement: LLM fallback posting
The system SHALL fall back to a Haiku invocation for posting when Go-side posting fails with an unexpected error.

#### Scenario: Go posting fails
- **WHEN** the `gh api` call from Go fails
- **THEN** the system invokes Haiku with the error message, findings, diff, and posting instructions, allowing the LLM to adaptively construct and execute the posting

#### Scenario: Haiku fallback succeeds
- **WHEN** the Haiku fallback successfully posts the review
- **THEN** the system logs that fallback was used and continues normally

#### Scenario: Both Go and Haiku fail
- **WHEN** both the Go posting and Haiku fallback fail
- **THEN** the system logs the error to stderr but does not fail the review command (the review text has already been output to stdout)

### Requirement: Diff range validation
The system SHALL validate finding line numbers against the actual diff ranges before constructing inline comments.

#### Scenario: Parse diff ranges
- **WHEN** the system prepares to post
- **THEN** it parses the diff to extract valid line ranges per file (new-side line numbers from `@@` hunk headers)

#### Scenario: Invalid line number
- **WHEN** a finding references a line not in any diff hunk for that file
- **THEN** the finding is moved to the review body instead of being posted as an inline comment
