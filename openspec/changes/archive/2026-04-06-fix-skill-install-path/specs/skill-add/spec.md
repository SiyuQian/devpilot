## MODIFIED Requirements

### Requirement: Install a named skill from default source
The system SHALL fetch and install a single named skill from `github.com/siyuqian/devpilot` when the user runs `devpilot skill add <name>`. The skill is fetched at the latest release tag by default. All files from `skills/<name>/` in the source repo are written to `.claude/skills/<name>/` in the current project. Existing files are silently overwritten. The installed skill is recorded in `.devpilot.yaml` with name, source, version, and installedAt timestamp.

#### Scenario: Add skill at latest version
- **WHEN** user runs `devpilot skill add pm`
- **THEN** the system fetches the latest release tag from `github.com/siyuqian/devpilot`
- **AND** downloads all files from `skills/pm/` at that tag
- **AND** writes them to `.claude/skills/pm/` in the current project
- **AND** records `{name: pm, source: github.com/siyuqian/devpilot, version: <tag>, installedAt: <now>}` in `.devpilot.yaml`
- **AND** prints a success message indicating the skill and version installed

#### Scenario: Overwrite existing skill
- **WHEN** user runs `devpilot skill add pm` and `.claude/skills/pm/` already exists
- **THEN** the system overwrites all files silently without prompting
- **AND** updates the skill entry in `.devpilot.yaml` with the new version and installedAt
