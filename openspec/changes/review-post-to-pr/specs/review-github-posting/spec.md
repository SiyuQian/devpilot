## Purpose

Defines the prompt instructions and comment template for Claude to post review results to a GitHub PR as a formal review with inline comments.

## ADDED Requirements

### Requirement: Post review as GitHub PR review
The system SHALL include embedded posting instructions (`review-posting.md`) that direct Claude to submit review results to the PR via `gh api` after completing the review.

#### Scenario: Approved review posts APPROVE status
- **WHEN** Claude completes a review with verdict APPROVE
- **THEN** Claude SHALL submit a GitHub PR review with event `APPROVE`, body containing the Summary and Overall Assessment, and inline comments for any SUGGESTION or PRAISE findings

#### Scenario: Non-approved review posts COMMENT status
- **WHEN** Claude completes a review with verdict REQUEST_CHANGES
- **THEN** Claude SHALL submit a GitHub PR review with event `COMMENT` (never `REQUEST_CHANGES`), body containing the Summary and Overall Assessment, and inline comments for each finding

#### Scenario: Inline comments include severity and explanation
- **WHEN** Claude posts an inline comment for a finding
- **THEN** the comment body SHALL include the severity tag (e.g., `[CRITICAL]`, `[WARNING]`, `[SUGGESTION]`, `[PRAISE]`), a brief title, and the full explanation

#### Scenario: Inline comment uses line/side parameters
- **WHEN** Claude posts an inline comment
- **THEN** Claude SHALL use the `line` (file line number) and `side` (`RIGHT`) parameters in the API call, NOT the legacy `position` field

#### Scenario: Finding on line outside diff range
- **WHEN** Claude has a finding on a line that is not part of the diff
- **THEN** Claude SHALL include the finding as a top-level review body comment instead of an inline comment

#### Scenario: Review body template
- **WHEN** Claude posts the review body
- **THEN** the body SHALL follow this structure:
  1. Greeting: `Nice work on this PR, @{author}! Here are some thoughts from my review.` (where `{author}` is the PR author's GitHub username, obtained via `gh pr view`)
  2. Summary (1-2 sentences describing the change and overall impression)
  3. Verdict line: `**Verdict: APPROVED**` or `**Verdict: NEEDS ATTENTION**`
  4. Overall Assessment (2-3 sentences on code quality and observations)
  5. Footer: `— Automated review by DevPilot`

#### Scenario: Posting failure does not block review output
- **WHEN** the `gh api` call to post the review fails (auth error, network error, etc.)
- **THEN** Claude SHALL report the error in its output but the review text SHALL still be fully streamed to the terminal

### Requirement: Posting instructions are a separate embedded file
The system SHALL maintain posting instructions in a separate `review-posting.md` file, embedded alongside `review-prompt.md` and `review-template.md`.

#### Scenario: Posting instructions assembled into prompt
- **WHEN** posting is enabled (no `--no-post` flag)
- **THEN** the assembled prompt includes the posting instructions section after the review template

#### Scenario: Posting instructions omitted when disabled
- **WHEN** the `--no-post` flag is set
- **THEN** the assembled prompt does NOT include the posting instructions section
