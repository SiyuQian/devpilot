## MODIFIED Requirements

### Requirement: Interactive install level selection
When running `devpilot skill add <name>` WITHOUT the `--level` flag, the system SHALL prompt the user to select the install level before fetching the skill. The prompt SHALL display two options: project level (`.claude/skills/`) and user level (`~/.claude/skills/`). The default selection SHALL be project level. When the `--level` flag IS set, the system SHALL NOT display the prompt and SHALL use the flag value as the install destination.

#### Scenario: User selects project level (default)
- **WHEN** user runs `devpilot skill add pm` and presses enter at the level prompt without input
- **THEN** the skill is installed to `.claude/skills/pm/` in the current project directory
- **AND** the skill entry is recorded in the project `.devpilot.yaml`

#### Scenario: User selects user level
- **WHEN** user runs `devpilot skill add pm` and selects option 2 (User) at the level prompt
- **THEN** the skill is installed to `~/.claude/skills/pm/`
- **AND** the skill entry is recorded in the user-level `~/.config/devpilot/.devpilot.yaml` (not the project `.devpilot.yaml`)

#### Scenario: Non-interactive environment without --level
- **WHEN** `devpilot skill add pm` runs with stdin not connected to a TTY and no `--level` flag
- **THEN** the system SHALL skip the prompt and default to project level

## ADDED Requirements

### Requirement: Non-interactive level selection via --level flag
The system SHALL accept a `--level` flag on `devpilot skill add` whose value is either `project` or `user`. When `--level` is set, the system SHALL use its value as the install destination and SHALL NOT display the interactive level prompt, regardless of whether stdin is a TTY. An invalid `--level` value SHALL cause a clear error at argument-parse time before any network or filesystem work is performed. The precedence for level resolution is: `--level` flag (if set) > interactive prompt (if stdin is a TTY) > project-level default.

#### Scenario: --level project bypasses prompt
- **WHEN** user runs `devpilot skill add pm --level project` in an interactive terminal
- **THEN** the system does NOT display the level prompt
- **AND** installs the skill to `.claude/skills/pm/`
- **AND** records the skill entry in the project `.devpilot.yaml`

#### Scenario: --level user bypasses prompt
- **WHEN** user runs `devpilot skill add pm --level user` in an interactive terminal
- **THEN** the system does NOT display the level prompt
- **AND** installs the skill to `~/.claude/skills/pm/`
- **AND** records the skill entry in the user-level `~/.config/devpilot/.devpilot.yaml` (not the project `.devpilot.yaml`)

#### Scenario: --level user in non-interactive environment
- **WHEN** user runs `devpilot skill add pm --level user` with stdin not connected to a TTY
- **THEN** the system installs the skill at user level (the flag overrides the non-TTY project default)
- **AND** records the skill entry in `~/.config/devpilot/.devpilot.yaml`

#### Scenario: --level with --all
- **WHEN** user runs `devpilot skill add --all --level user`
- **THEN** the system installs every catalog skill at user level without prompting

#### Scenario: Invalid --level value
- **WHEN** user runs `devpilot skill add pm --level system`
- **THEN** the system returns an error stating that `--level` must be `project` or `user`
- **AND** no catalog fetch or install is performed
