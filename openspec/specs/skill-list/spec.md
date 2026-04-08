## Purpose

Defines requirements for listing installed skills in a DevPilot project.
## Requirements
### Requirement: List installed skills
The system SHALL display all skills tracked in both project-level `.devpilot.yaml` and user-level `~/.config/devpilot/.devpilot.yaml` when the user runs `devpilot skill list`. Output SHALL include name, source, version, installedAt, and level for each skill, formatted as a table. Project-level skills are listed first, then user-level skills.

#### Scenario: Skills installed at both levels
- **WHEN** user runs `devpilot skill list` and both project and user configs contain skill entries
- **THEN** the system prints a table with columns: NAME, SOURCE, VERSION, INSTALLED, LEVEL
- **AND** project-level skills show `project` in the LEVEL column
- **AND** user-level skills show `user` in the LEVEL column
- **AND** project-level skills appear before user-level skills

#### Scenario: Only user-level skills installed
- **WHEN** user runs `devpilot skill list` and only `~/.config/devpilot/.devpilot.yaml` contains skill entries
- **THEN** the system prints the table with user-level skills only

#### Scenario: No skills installed at any level
- **WHEN** user runs `devpilot skill list` and no skills are tracked at either level
- **THEN** the system prints a message indicating no skills are installed

#### Scenario: No project config found
- **WHEN** user runs `devpilot skill list` in a directory without `.devpilot.yaml`
- **THEN** the system still shows user-level skills (does NOT error)
- **AND** only project-level skills are omitted

