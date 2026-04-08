## MODIFIED Requirements

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
