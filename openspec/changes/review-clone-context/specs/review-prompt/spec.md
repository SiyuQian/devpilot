## MODIFIED Requirements

### Requirement: Prompt assembly
The system SHALL assemble the final review prompt by combining: review instructions, repo clone/search instructions, comment template, and the PR URL.

#### Scenario: Complete prompt assembly
- **WHEN** the review command prepares to invoke Claude
- **THEN** the assembled prompt includes the review instructions, clone and context discovery instructions, comment template, and the PR URL for Claude to inspect

#### Scenario: No pre-gathered context section
- **WHEN** the prompt is assembled
- **THEN** it SHALL NOT include a pre-gathered project context section; instead it instructs Claude to clone and search the repo itself
