## ADDED Requirements

### Requirement: Show spinner during AI analysis
The system SHALL display an animated spinner with a status message while waiting for Claude to generate the commit plan.

#### Scenario: Analysis in progress
- **WHEN** the system is waiting for Claude's response
- **THEN** the UI SHALL display an animated spinner with the text "Analyzing changes..."

#### Scenario: Analysis completes
- **WHEN** Claude returns a response
- **THEN** the spinner SHALL stop and the UI SHALL transition to the plan display phase

### Requirement: Display formatted commit plan
The system SHALL display the proposed commit plan with colored formatting, showing each commit's message, file list with change status indicators, and diff statistics.

#### Scenario: Multiple commits proposed
- **WHEN** the plan contains more than one commit
- **THEN** the UI SHALL display numbered commits, each showing the commit message (with type highlighted), file list with status indicators (M/A/D in yellow/green/red), and per-file insertion/deletion counts

#### Scenario: Single commit proposed
- **WHEN** the plan contains exactly one commit
- **THEN** the UI SHALL display the commit message and file list without numbering

#### Scenario: Files excluded
- **WHEN** the plan contains excluded files
- **THEN** the UI SHALL display an "Excluded" section listing each file with its reason, styled in muted color

### Requirement: Interactive plan confirmation
The system SHALL present the user with options to accept, edit, or abort the commit plan.

#### Scenario: Accept all commits
- **WHEN** user presses `a` (multi-commit) or `y` (single-commit)
- **THEN** the system SHALL proceed to execute all commits in the plan

#### Scenario: Edit plan
- **WHEN** user presses `e`
- **THEN** the system SHALL open `$EDITOR` with a human-readable representation of the plan, and re-parse the edited result

#### Scenario: Abort
- **WHEN** user presses `n`
- **THEN** the system SHALL exit without committing, leaving the working tree unchanged (run `git reset` to unstage)

### Requirement: Show execution progress
The system SHALL display real-time progress as each commit in the plan is executed.

#### Scenario: Commit in progress
- **WHEN** a commit is being created
- **THEN** the UI SHALL show a spinner next to the current commit and checkmarks next to completed commits

#### Scenario: Commit fails mid-sequence
- **WHEN** a git commit operation fails
- **THEN** the UI SHALL show an error indicator on the failed commit, list which commits succeeded, and exit with a non-zero status

### Requirement: Show completion summary
The system SHALL display a summary after all commits are created, showing the short hash, message, and diff statistics for each commit.

#### Scenario: All commits succeed
- **WHEN** all commits in the plan are created successfully
- **THEN** the UI SHALL display each commit with its short hash (7 chars), first line of message, and `N files changed, N insertions(+), N deletions(-)` statistics

### Requirement: Reuse existing lipgloss style conventions
The commit TUI SHALL use the same color scheme and style patterns as the taskrunner TUI in `internal/taskrunner/tui_view.go`.

#### Scenario: Visual consistency
- **WHEN** the commit TUI renders any styled element
- **THEN** it SHALL use color "12" for primary/focus, "10" for success, "9" for error, "240" for muted, and RoundedBorder for panels — matching the taskrunner dashboard

### Requirement: Support dry-run mode
The system SHALL support a `--dry-run` flag that shows the plan but does not execute any commits.

#### Scenario: Dry run
- **WHEN** user runs `devpilot commit --dry-run`
- **THEN** the system SHALL display the commit plan and exit without committing, showing "(dry-run: not committing)"
