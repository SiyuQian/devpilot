## Purpose

Defines requirements for displaying the full skill catalog with installation status indicators.
## Requirements
### Requirement: Display full skill catalog with installation status
The system SHALL fetch all available skills from the devpilot GitHub catalog at ref `main` and display them in a table with columns: NAME, DESCRIPTION, INSTALLED, LEVEL. For installed skills, INSTALLED and LEVEL SHALL show the install date and level (project/user). For uninstalled skills, INSTALLED and LEVEL SHALL show "—".

#### Scenario: Mix of installed and uninstalled skills
- **WHEN** user runs `devpilot skill list` and some catalog skills are installed
- **THEN** the system prints a table showing ALL catalog skills
- **AND** installed skills display their install date (formatted as `2006-01-02`) and level
- **AND** uninstalled skills display "—" for installed and level

#### Scenario: No skills installed
- **WHEN** user runs `devpilot skill list` and no skills are installed
- **THEN** the system prints the full catalog table with all skills showing "—" for installed and level

#### Scenario: All skills installed
- **WHEN** user runs `devpilot skill list` and every catalog skill is installed
- **THEN** the system prints the full catalog table with all skills showing their install date and level

### Requirement: Truncate long descriptions
The system SHALL truncate skill descriptions to 40 characters followed by "..." when they exceed 40 characters to maintain readable table formatting.

#### Scenario: Description exceeds 40 characters
- **WHEN** a skill's description is longer than 40 characters
- **THEN** the displayed description SHALL be truncated to 40 characters followed by "..."

### Requirement: Graceful catalog fetch failure
The system SHALL display an error message and fall back to showing only installed skills when the catalog fetch fails (network error, HTTP error, etc.).

#### Scenario: Network error during catalog fetch
- **WHEN** user runs `devpilot skill list` and the raw URL is unreachable
- **THEN** the system prints a warning about the catalog fetch failure
- **AND** the system falls back to displaying installed skills only

