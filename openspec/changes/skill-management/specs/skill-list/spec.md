## ADDED Requirements

### Requirement: List installed skills
The system SHALL display all skills tracked in `.devpilot.yaml` when the user runs `devpilot skill list`. Output SHALL include name, source, version, and installedAt for each skill, formatted as a table.

#### Scenario: Skills are installed
- **WHEN** user runs `devpilot skill list` and `.devpilot.yaml` contains skill entries
- **THEN** the system prints a table with columns: NAME, SOURCE, VERSION, INSTALLED
- **AND** each row corresponds to one installed skill

#### Scenario: No skills installed
- **WHEN** user runs `devpilot skill list` and no skills are tracked in `.devpilot.yaml`
- **THEN** the system prints a message indicating no skills are installed

#### Scenario: No project config found
- **WHEN** user runs `devpilot skill list` in a directory without `.devpilot.yaml`
- **THEN** the system returns an error instructing the user to run `devpilot init` first
