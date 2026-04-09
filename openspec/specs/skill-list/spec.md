## Purpose

Defines requirements for listing installed skills in a DevPilot project.
## Requirements
### Requirement: List installed skills
The system SHALL display only installed skills (from both project-level and user-level configs) when the user runs `devpilot skill list --installed`. Output SHALL include NAME, DESCRIPTION, INSTALLED, LEVEL columns. Project-level skills are listed first, then user-level skills.

#### Scenario: Skills installed at both levels with --installed flag
- **WHEN** user runs `devpilot skill list --installed` and both project and user configs contain skill entries
- **THEN** the system prints a table with columns: NAME, DESCRIPTION, INSTALLED, LEVEL
- **AND** project-level skills show `project` in the LEVEL column
- **AND** user-level skills show `user` in the LEVEL column
- **AND** the INSTALLED column shows the installedAt timestamp formatted as `2006-01-02`
- **AND** project-level skills appear before user-level skills

#### Scenario: No skills installed with --installed flag
- **WHEN** user runs `devpilot skill list --installed` and no skills are tracked at either level
- **THEN** the system prints a message indicating no skills are installed

#### Scenario: No project config found
- **WHEN** user runs `devpilot skill list --installed` in a directory without `.devpilot.yaml`
- **THEN** the system still shows user-level skills (does NOT error)
- **AND** only project-level skills are omitted

