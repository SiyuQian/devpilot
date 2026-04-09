# user-level-skill-config Specification

## Purpose

Defines requirements for managing user-level skill configuration, including resolving the user config directory and persisting skill entries in the user-level `.devpilot.yaml`.
## Requirements
### Requirement: Resolve user config directory
The system SHALL provide a `UserConfigDir()` function in the `project` package that returns `~/.config/devpilot/` (resolved to an absolute path). This path is used for reading and writing user-level `.devpilot.yaml`.

#### Scenario: Resolve user config dir
- **WHEN** `UserConfigDir()` is called
- **THEN** it returns `<home>/.config/devpilot/` where `<home>` is the user's home directory

#### Scenario: Home directory unavailable
- **WHEN** `UserConfigDir()` is called and the home directory cannot be resolved
- **THEN** it returns an error

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

