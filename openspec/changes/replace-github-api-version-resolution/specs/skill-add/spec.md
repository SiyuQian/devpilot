## MODIFIED Requirements

### Requirement: Install a named skill from default source
The system SHALL fetch and install a single named skill from `github.com/siyuqian/devpilot` when the user runs `devpilot skill add <name>`. The skill is fetched from the `main` branch by default. The system SHALL resolve the skill's file list from `skills/index.json` via raw URL and download each file from `raw.githubusercontent.com`. The install target directory is determined by the user's level selection (project or user). Existing files are silently overwritten. The installed skill is recorded in the appropriate `.devpilot.yaml` with name, source, and installedAt timestamp.

#### Scenario: Add skill at latest (project level)
- **WHEN** user runs `devpilot skill add pm` and selects project level
- **THEN** the system downloads `skills/index.json` from `raw.githubusercontent.com` at ref `main`
- **AND** looks up `pm` in the index to get its file list
- **AND** downloads each file from `raw.githubusercontent.com`
- **AND** writes them to `.claude/skills/pm/` in the current project
- **AND** records `{name: pm, source: github.com/siyuqian/devpilot, installedAt: <now>}` in `.devpilot.yaml`
- **AND** prints a success message indicating the skill and install level

#### Scenario: Add skill at user level
- **WHEN** user runs `devpilot skill add pm` and selects user level
- **THEN** the system downloads files via raw URL at ref `main`
- **AND** writes files to `~/.claude/skills/pm/`
- **AND** records `{name: pm, source: github.com/siyuqian/devpilot, installedAt: <now>}` in `~/.config/devpilot/.devpilot.yaml`
- **AND** prints a success message indicating the skill was installed at user level

#### Scenario: Add skill at pinned ref
- **WHEN** user runs `devpilot skill add pm@v0.4.0`
- **THEN** the system downloads `skills/index.json` at ref `v0.4.0` from `raw.githubusercontent.com`
- **AND** fetches files listed in the index via raw URLs
- **AND** records `{name: pm, source: github.com/siyuqian/devpilot, installedAt: <now>}` in the appropriate `.devpilot.yaml`

#### Scenario: Overwrite existing skill
- **WHEN** user runs `devpilot skill add pm` and the target directory already exists
- **THEN** the system overwrites all files silently without prompting
- **AND** updates the skill entry in the appropriate `.devpilot.yaml` with the new installedAt

#### Scenario: Skill name not found
- **WHEN** user runs `devpilot skill add nonexistent-skill`
- **THEN** the system returns an error indicating the skill was not found in the index

#### Scenario: No skill name provided
- **WHEN** user runs `devpilot skill add` with no arguments
- **THEN** the system returns an error stating that a skill name is required
