## ADDED Requirements

### Requirement: Skill selection step in devpilot init
During `devpilot init`, the system SHALL present an interactive multi-select checklist of available skills from the devpilot catalog after the board configuration step and before the custom skill creation step. Selected skills SHALL be fetched and installed using the same mechanism as `devpilot skill add`. The checklist SHALL be skipped when `-y` (accept defaults) is passed; in that case no skills are auto-installed.

#### Scenario: User selects skills during init
- **WHEN** user runs `devpilot init` and reaches the skill selection step
- **THEN** the system displays a checklist of available skills with names and descriptions
- **AND** the user can toggle selections with space and confirm with enter
- **AND** the system fetches and installs all selected skills
- **AND** each installed skill is recorded in `.devpilot.yaml`

#### Scenario: User selects no skills
- **WHEN** user runs `devpilot init` and confirms with no skills selected
- **THEN** the system skips skill installation and continues init

#### Scenario: Init with defaults flag
- **WHEN** user runs `devpilot init -y`
- **THEN** the skill selection step is skipped entirely
- **AND** no skills are automatically installed

### Requirement: Catalog manifest for skill selection
The system SHALL use a hardcoded catalog of available skills for the init checklist. The catalog SHALL include name and description for each skill.

#### Scenario: Catalog is available offline
- **WHEN** user runs `devpilot init` without network access and reaches the skill selection step
- **THEN** the system displays the hardcoded catalog without making network requests
- **AND** network requests only occur when the user confirms their selection and skills are fetched
