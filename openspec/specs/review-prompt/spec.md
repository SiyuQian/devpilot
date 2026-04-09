## Purpose

Defines the embedded review instructions and comment template used by the AI-powered code review system for structured, project-agnostic review output.

## Requirements

### Requirement: Review instructions file
The system SHALL include an embedded `review-prompt.md` file that contains comprehensive review criteria and methodology for the LLM.

#### Scenario: Review covers all quality dimensions
- **WHEN** Claude executes a review using the instructions file
- **THEN** the review MUST evaluate: correctness, security vulnerabilities (OWASP top 10), performance concerns, error handling, maintainability, and style consistency

#### Scenario: Severity classification
- **WHEN** Claude identifies an issue during review
- **THEN** it MUST classify the issue as one of: CRITICAL (blocks merge), WARNING (should fix), SUGGESTION (nice to have), or PRAISE (good pattern worth noting)

#### Scenario: Instructions are project-agnostic
- **WHEN** the review is run on any project (Go, Node, Python, etc.)
- **THEN** the instructions apply universally without language-specific assumptions

### Requirement: Review comment template
The system SHALL include an embedded `review-template.md` that defines the structured output format for review results.

#### Scenario: Template structure
- **WHEN** Claude produces review output
- **THEN** it MUST follow the template structure: Summary (1-2 sentences), Verdict (APPROVE or REQUEST_CHANGES), File-by-File Findings (with severity, line references, and explanations), and Overall Assessment

#### Scenario: Approve verdict
- **WHEN** Claude finds no CRITICAL or WARNING issues
- **THEN** the verdict MUST be APPROVE with a brief confirmation

#### Scenario: Request changes verdict
- **WHEN** Claude finds one or more CRITICAL issues
- **THEN** the verdict MUST be REQUEST_CHANGES with clear explanations of blocking issues

### Requirement: Prompt assembly
The system SHALL assemble the final review prompt by combining: review instructions, comment template, gathered project context, and the PR URL.

#### Scenario: Complete prompt assembly
- **WHEN** the review command prepares to invoke Claude
- **THEN** the assembled prompt includes the review instructions, comment template, any detected project conventions, and the PR URL for Claude to inspect

#### Scenario: Context section is optional
- **WHEN** no project conventions are detected
- **THEN** the prompt omits the project context section and proceeds with instructions + template + PR URL only
