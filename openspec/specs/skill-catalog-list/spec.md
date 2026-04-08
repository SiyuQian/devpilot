## Purpose

Defines requirements for displaying the full skill catalog with installation status indicators.
## Requirements
### Requirement: Display full skill catalog with installation status
The system SHALL fetch all available skills from the devpilot GitHub catalog and display them in a table with columns: NAME, DESCRIPTION, VERSION, LEVEL. For installed skills, VERSION and LEVEL SHALL show the installed version and level (project/user). For uninstalled skills, VERSION and LEVEL SHALL show "—".

#### Scenario: Mix of installed and uninstalled skills
- **WHEN** user runs `devpilot skill list` and some catalog skills are installed
- **THEN** the system prints a table showing ALL catalog skills
- **AND** installed skills display their version and level
- **AND** uninstalled skills display "—" for version and level

#### Scenario: No skills installed
- **WHEN** user runs `devpilot skill list` and no skills are installed
- **THEN** the system prints the full catalog table with all skills showing "—" for version and level

#### Scenario: All skills installed
- **WHEN** user runs `devpilot skill list` and every catalog skill is installed
- **THEN** the system prints the full catalog table with all skills showing their installed version and level

### Requirement: Truncate long descriptions
The system SHALL truncate skill descriptions to 40 characters followed by "..." when they exceed 40 characters to maintain readable table formatting.

#### Scenario: Description exceeds 40 characters
- **WHEN** a skill's description is longer than 40 characters
- **THEN** the displayed description SHALL be truncated to 40 characters followed by "..."

### Requirement: Graceful catalog fetch failure
The system SHALL display an error message and fall back to showing only installed skills when the GitHub catalog fetch fails (network error, rate limit, etc.).

#### Scenario: Network error during catalog fetch
- **WHEN** user runs `devpilot skill list` and the GitHub API is unreachable
- **THEN** the system prints a warning about the catalog fetch failure
- **AND** the system falls back to displaying installed skills only
