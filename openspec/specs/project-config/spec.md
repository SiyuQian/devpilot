## Purpose

Defines the project configuration file format, location, and content structure for DevPilot projects.

## Requirements

### Requirement: Project config file format and location
The system SHALL store project configuration in `.devpilot.yaml` (YAML format) in the project root directory. The previous `.devpilot.json` format is no longer supported. The config file SHALL contain board name, source, models, openspecMinVersion, and a skills array. No automatic migration from `.devpilot.json` is provided.

#### Scenario: Load existing config
- **WHEN** `.devpilot.yaml` exists in the current directory
- **THEN** the system parses it as YAML and returns a populated Config struct

#### Scenario: Config file does not exist
- **WHEN** `.devpilot.yaml` does not exist in the current directory
- **THEN** the system returns a zero-value Config without error

#### Scenario: Save config
- **WHEN** the system writes project configuration
- **THEN** it creates or overwrites `.devpilot.yaml` with YAML-encoded content

### Requirement: Track installed skills in project config
The config file SHALL include a `skills` array where each entry records a skill installation with fields: `name` (string), `source` (string), and `installedAt` (RFC3339 timestamp). There is no `version` field.

#### Scenario: Skill entry written on install
- **WHEN** a skill is installed via `devpilot skill add` or `devpilot init`
- **THEN** `.devpilot.yaml` contains an entry for that skill under the `skills` key
- **AND** the entry includes name, source, and installedAt

#### Scenario: Skill entry updated on reinstall
- **WHEN** a skill is installed again (overwrite)
- **THEN** the existing entry in `.devpilot.yaml` is updated with the new installedAt
- **AND** no duplicate entries are created
