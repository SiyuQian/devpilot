## ADDED Requirements

### Requirement: Analyze staged changes and produce a commit plan
The system SHALL send the full staged diff content to Claude and receive a structured JSON response containing one or more commit groups, each with a conventional commit message and a list of files.

#### Scenario: Multiple logical changes detected
- **WHEN** staged changes span multiple unrelated concerns (e.g., a feature addition and a config fix)
- **THEN** the plan SHALL contain multiple commits, each grouping related files together with an appropriate conventional commit message

#### Scenario: Single logical change
- **WHEN** all staged changes relate to one concern
- **THEN** the plan SHALL contain exactly one commit covering all changed files

#### Scenario: No staged changes
- **WHEN** there are no staged changes after `git add .`
- **THEN** the system SHALL display "No changes to commit." and exit without calling Claude

### Requirement: Exclude sensitive and non-committable files
The system SHALL identify files that should not be committed — including files containing secrets, debug artifacts, and build outputs — and return them in an `excluded` array with a reason for each.

#### Scenario: Secret file detected
- **WHEN** staged changes include a file like `.env`, `.env.local`, or a file containing API keys/tokens
- **THEN** the file SHALL appear in the `excluded` array with a reason explaining why

#### Scenario: No sensitive files
- **WHEN** no staged files match exclusion criteria
- **THEN** the `excluded` array SHALL be empty

### Requirement: Send actual diff content to Claude
The system SHALL send the output of `git diff --cached` as the primary context for commit message generation, not just file names and stat summaries.

#### Scenario: Small diff
- **WHEN** the total diff is under 15,000 characters
- **THEN** the full diff content SHALL be sent to Claude

#### Scenario: Large diff with truncation
- **WHEN** the total diff exceeds 15,000 characters
- **THEN** individual file diffs SHALL be truncated to 200 lines each, and the total SHALL be capped at 15,000 characters, with truncation markers appended

#### Scenario: Binary files in diff
- **WHEN** staged changes include binary files
- **THEN** binary files SHALL be listed as `Binary file: <path>` without diff content

### Requirement: Validate commit plan against actual changes
The system SHALL verify that every file listed in the commit plan exists in the actual staged changes, and that every staged file (not excluded) appears in exactly one commit group.

#### Scenario: Plan references unknown file
- **WHEN** Claude's plan includes a file not in the staged changes
- **THEN** the system SHALL remove it from the plan and warn the user

#### Scenario: Staged file missing from plan
- **WHEN** a staged file (not excluded) does not appear in any commit group
- **THEN** the system SHALL add it to the last commit group and warn the user

### Requirement: Fallback on JSON parse failure
The system SHALL gracefully handle cases where Claude's output cannot be parsed as valid JSON.

#### Scenario: Malformed JSON response
- **WHEN** Claude returns output that is not valid JSON
- **THEN** the system SHALL create a single-commit plan containing all staged files, using the raw Claude output as the commit message
