## MODIFIED Requirements

### Requirement: Load and save user-level skill entries
The system SHALL use the existing `project.Load` and `project.Save` functions with the user config directory to read and write user-level skill entries. The user-level `.devpilot.yaml` uses the same format as the project-level config. Skill entries contain `name`, `source`, and `installedAt` fields (no `version` field).

#### Scenario: Save a user-level skill entry
- **WHEN** a skill is installed at user level
- **THEN** the system calls `project.Load(userConfigDir)` to read existing config
- **AND** calls `cfg.UpsertSkill(entry)` with the skill's name, source, and installedAt
- **AND** calls `project.Save(userConfigDir, cfg)` to persist

#### Scenario: User config file does not exist yet
- **WHEN** a skill is installed at user level and `~/.config/devpilot/.devpilot.yaml` does not exist
- **THEN** `project.Load` returns a zero-value Config (existing behavior)
- **AND** `project.Save` creates the directory and file (existing behavior)

#### Scenario: Loading config with legacy version field
- **WHEN** an existing `.devpilot.yaml` contains a `version` field in a skill entry
- **THEN** the system SHALL load successfully, silently ignoring the `version` field
