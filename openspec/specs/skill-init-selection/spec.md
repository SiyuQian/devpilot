## Purpose

Defines requirements for the skill selection step during `devpilot init` and the catalog used to present available skills.
## Requirements
### Requirement: Skill selection step in devpilot init
During `devpilot init`, the system SHALL present an interactive multi-select checklist of available skills from the devpilot catalog after the board configuration step and before the custom skill creation step. Selected skills SHALL be fetched and installed using the same mechanism as `devpilot skill add`. The checklist SHALL be skipped when `-y` (accept defaults) is passed; in that case no skills are auto-installed.

The init command SHALL NOT generate or detect CLAUDE.md. The init command SHALL prompt "Configure task source? [Y/n]:" before the task source configuration step. If the user declines, the entire task source configuration (both Trello and GitHub paths) SHALL be skipped.

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

#### Scenario: User skips task source configuration
- **WHEN** user runs `devpilot init` and responds "n" to "Configure task source? [Y/n]:"
- **THEN** the system skips board/source selection entirely
- **AND** no source or board is written to `.devpilot.yaml`

#### Scenario: CLAUDE.md not mentioned in status
- **WHEN** user runs `devpilot init`
- **THEN** the status output does NOT include any line about CLAUDE.md

### Requirement: Catalog fetched from remote
The system SHALL fetch the skill catalog from the remote repository via `skillmgr.FetchCatalog()` (which downloads `skills/index.json` from `raw.githubusercontent.com`). The catalog requires network access. The catalog SHALL include name and description for each skill.

#### Scenario: Catalog fetched at init time
- **WHEN** user runs `devpilot init` and reaches the skill selection step
- **THEN** the system fetches the catalog from `raw.githubusercontent.com` via `skillmgr.FetchCatalog`
- **AND** displays the fetched skills with names and descriptions

#### Scenario: Network unavailable during init
- **WHEN** user runs `devpilot init` without network access and reaches the skill selection step
- **THEN** the catalog fetch fails and the system reports an error fetching the skill catalog

