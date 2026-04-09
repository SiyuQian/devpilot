## MODIFIED Requirements

### Requirement: Review instructions file
The system SHALL include an embedded `review-prompt.md` file that contains review criteria, false positive guidance, and instructions to output findings as structured JSON.

#### Scenario: Review covers all quality dimensions
- **WHEN** Claude executes Round 1 using the instructions file
- **THEN** the review MUST evaluate: correctness, security vulnerabilities (OWASP top 10), performance concerns, error handling, maintainability, and style consistency

#### Scenario: Severity classification
- **WHEN** Claude identifies an issue during review
- **THEN** it MUST classify the issue as one of: CRITICAL (blocks merge), WARNING (should fix), SUGGESTION (nice to have), or PRAISE (good pattern worth noting)

#### Scenario: Instructions are project-agnostic
- **WHEN** the review is run on any project (Go, Node, Python, etc.)
- **THEN** the instructions apply universally without language-specific assumptions

#### Scenario: JSON output format
- **WHEN** Claude completes the review
- **THEN** it MUST output a JSON object with `summary` (string), `findings` (array of finding objects), and `assessment` (string) — not markdown

#### Scenario: Finding object structure
- **WHEN** Claude reports a finding
- **THEN** each finding object MUST contain: `file` (string), `line` (int), `end_line` (int, optional), `severity` (CRITICAL|WARNING|SUGGESTION|PRAISE), `title` (string), `explanation` (string), and `suggestion` (string, optional)

#### Scenario: Pre-gathered context in prompt
- **WHEN** the Round 1 prompt is assembled
- **THEN** it includes the diff text, PR metadata, and convention file contents directly in the prompt — Claude does NOT need to run `gh` commands or clone repos

#### Scenario: False positive awareness
- **WHEN** Claude evaluates potential issues
- **THEN** the prompt instructs Claude to avoid: pre-existing issues, linter/compiler-catchable issues, intentional behavior changes, issues on unmodified lines, and general quality concerns not called out in project conventions

### Requirement: Scoring prompt
The system SHALL include an embedded `review-scoring.md` file that instructs the scoring model to evaluate each finding independently.

#### Scenario: Scoring input format
- **WHEN** Round 2 is invoked
- **THEN** the prompt includes the original diff context and each finding from Round 1, asking the scorer to assign a confidence score 0-100

#### Scenario: Scoring output format
- **WHEN** the scorer completes
- **THEN** it outputs a JSON array of objects with `index` (int, matching the finding's position) and `score` (int, 0-100)

### Requirement: Prompt assembly
The system SHALL assemble the Round 1 prompt by combining: review instructions, pre-gathered context (diff, metadata, conventions), and review task.

#### Scenario: Complete prompt assembly
- **WHEN** the pipeline prepares Round 1
- **THEN** the assembled prompt includes review instructions, the PR diff, PR metadata (title, body, author, base branch), any detected project conventions, and the task instruction

#### Scenario: Context section is optional
- **WHEN** no project conventions are detected
- **THEN** the prompt omits the project context section

## REMOVED Requirements

### Requirement: Review comment template
**Reason**: Replaced by JSON output format. The markdown template is no longer needed since Round 1 outputs structured JSON and the final human-readable output is assembled by Go code.
**Migration**: Go code formats the filtered findings into human-readable markdown for CLI output.
