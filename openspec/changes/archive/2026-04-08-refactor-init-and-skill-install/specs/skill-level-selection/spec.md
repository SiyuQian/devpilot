## ADDED Requirements

### Requirement: Interactive install level selection
When running `devpilot skill add <name>`, the system SHALL prompt the user to select the install level before fetching the skill. The prompt SHALL display two options: project level (`.claude/skills/`) and user level (`~/.claude/skills/`). The default selection SHALL be project level.

#### Scenario: User selects project level (default)
- **WHEN** user runs `devpilot skill add pm` and presses enter at the level prompt without input
- **THEN** the skill is installed to `.claude/skills/pm/` in the current project directory
- **AND** the skill entry is recorded in `.devpilot.yaml`

#### Scenario: User selects user level
- **WHEN** user runs `devpilot skill add pm` and selects option 2 (User) at the level prompt
- **THEN** the skill is installed to `~/.claude/skills/pm/`
- **AND** the skill entry is NOT recorded in `.devpilot.yaml`

#### Scenario: Non-interactive environment
- **WHEN** `devpilot skill add pm` runs with stdin not connected to a TTY
- **THEN** the system SHALL skip the prompt and default to project level
