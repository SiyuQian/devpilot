## Purpose

Defines requirements for installing skills into a DevPilot project via the `devpilot skill add` command.

## Requirements

### Requirement: Install a named skill from default source
The system SHALL fetch and install a single named skill from `github.com/siyuqian/devpilot` when the user runs `devpilot skill add <name>`. The skill is fetched at the latest release tag by default. All files from `.claude/skills/<name>/` in the source repo are written to `.claude/skills/<name>/` in the current project. Existing files are silently overwritten. The installed skill is recorded in `.devpilot.yaml` with name, source, version, and installedAt timestamp.

#### Scenario: Add skill at latest version
- **WHEN** user runs `devpilot skill add pm`
- **THEN** the system fetches the latest release tag from `github.com/siyuqian/devpilot`
- **AND** downloads all files from `.claude/skills/pm/` at that tag
- **AND** writes them to `.claude/skills/pm/` in the current project
- **AND** records `{name: pm, source: github.com/siyuqian/devpilot, version: <tag>, installedAt: <now>}` in `.devpilot.yaml`
- **AND** prints a success message indicating the skill and version installed

#### Scenario: Add skill at pinned version
- **WHEN** user runs `devpilot skill add pm@v0.4.0`
- **THEN** the system fetches files from `.claude/skills/pm/` at tag `v0.4.0`
- **AND** records `version: v0.4.0` in `.devpilot.yaml`

#### Scenario: Overwrite existing skill
- **WHEN** user runs `devpilot skill add pm` and `.claude/skills/pm/` already exists
- **THEN** the system overwrites all files silently without prompting
- **AND** updates the skill entry in `.devpilot.yaml` with the new version and installedAt

#### Scenario: Skill name not found
- **WHEN** user runs `devpilot skill add nonexistent-skill`
- **THEN** the system returns an error indicating the skill was not found in the source repo

#### Scenario: No skill name provided
- **WHEN** user runs `devpilot skill add` with no arguments
- **THEN** the system returns an error stating that a skill name is required

### Requirement: Require execution inside a project directory
The system SHALL require `.devpilot.yaml` to exist in the current directory before installing a skill, to ensure skills are installed into a configured project.

#### Scenario: No project config found
- **WHEN** user runs `devpilot skill add pm` in a directory without `.devpilot.yaml`
- **THEN** the system returns an error instructing the user to run `devpilot init` first
