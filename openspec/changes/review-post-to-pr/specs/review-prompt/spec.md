## MODIFIED Requirements

### Requirement: Prompt assembly
The system SHALL assemble the final review prompt by combining: review instructions, comment template, posting instructions (when enabled), and the PR URL.

#### Scenario: Complete prompt assembly with posting
- **WHEN** the review command prepares to invoke Claude and posting is enabled
- **THEN** the assembled prompt includes the review instructions, comment template, posting instructions, and the PR URL

#### Scenario: Complete prompt assembly without posting
- **WHEN** the review command prepares to invoke Claude and `--no-post` is set
- **THEN** the assembled prompt includes the review instructions, comment template, and the PR URL but omits the posting instructions

#### Scenario: Context section is optional
- **WHEN** no project conventions are detected
- **THEN** the prompt omits the project context section and proceeds with the remaining sections only
