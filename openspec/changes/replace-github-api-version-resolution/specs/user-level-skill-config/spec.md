## MODIFIED Requirements

### Requirement: Track installed skills in config
The system SHALL record each installed skill in the appropriate `.devpilot.yaml` with `name`, `source`, and `installedAt` fields. The `version` field is removed — install timestamp is the only tracking metadata.

#### Scenario: Skill entry structure
- **WHEN** a skill is installed via `devpilot skill add`
- **THEN** the config entry SHALL contain `name` (string), `source` (string), and `installedAt` (timestamp)
- **AND** the entry SHALL NOT contain a `version` field

#### Scenario: Loading config with legacy version field
- **WHEN** an existing `.devpilot.yaml` contains a `version` field in a skill entry
- **THEN** the system SHALL load successfully, silently ignoring the `version` field
