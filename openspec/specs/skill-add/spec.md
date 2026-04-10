## Purpose

Defines requirements for installing skills into a DevPilot project via the `devpilot skill add` command.
## Requirements
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

### Requirement: Auto-create project config if missing
The system SHALL NOT require `.devpilot.yaml` to exist before installing a skill at project level. If the config file does not exist, `project.Load` returns a zero-value Config and `project.Save` creates it automatically after recording the skill entry.

#### Scenario: No project config found (project level)
- **WHEN** user runs `devpilot skill add pm`, selects project level, in a directory without `.devpilot.yaml`
- **THEN** the system installs the skill and creates `.devpilot.yaml` with the skill entry

#### Scenario: No project config found (user level)
- **WHEN** user runs `devpilot skill add pm` and selects user level in a directory without `.devpilot.yaml`
- **THEN** the system proceeds normally and installs the skill at user level

### Requirement: Install all skills in bulk
The system SHALL provide a `--all` flag on `devpilot skill add` that installs every skill listed in the catalog (`skills/index.json` on the default ref). When `--all` is set, the system SHALL fetch the catalog once, iterate all entries, install each one into the resolved install level, record each successful install in the in-memory config, and persist the config file ONCE at the end of the batch. Individual install failures SHALL NOT abort the batch; the system SHALL collect errors and print a summary indicating how many skills were installed and which ones failed with their error messages. The exit code SHALL be non-zero if any skill failed to install. If the catalog fetch itself fails, the system SHALL abort before attempting any individual install.

#### Scenario: Bulk install at project level
- **WHEN** user runs `devpilot skill add --all` in a project directory and the level resolves to project
- **THEN** the system fetches the skill catalog once
- **AND** installs every catalog skill into `.claude/skills/<name>/`
- **AND** records every successfully installed skill in the project `.devpilot.yaml` and persists the file once
- **AND** prints a summary like `Installed N/M skills into .claude/skills/`
- **AND** exits with code 0 when all installs succeed

#### Scenario: Bulk install at user level
- **WHEN** user runs `devpilot skill add --all --level user`
- **THEN** the system installs every catalog skill into `~/.claude/skills/<name>/`
- **AND** records every successfully installed skill in the user-level `~/.config/devpilot/.devpilot.yaml` (not the project `.devpilot.yaml`) and persists the file once
- **AND** prints a summary indicating user-level install

#### Scenario: Partial failure continues and reports
- **WHEN** user runs `devpilot skill add --all` and one or more skills fail to fetch or install
- **THEN** the system continues installing the remaining skills
- **AND** prints a summary listing the failed skills and their error messages
- **AND** still persists successfully installed entries to the config file
- **AND** exits with a non-zero code

#### Scenario: Catalog fetch fails
- **WHEN** user runs `devpilot skill add --all` and the catalog (`skills/index.json`) cannot be fetched
- **THEN** the system returns an error describing the fetch failure
- **AND** no skill is installed and no config file is modified

### Requirement: Skill name required unless --all is set
The system SHALL require exactly one positional argument (the skill name) for `devpilot skill add` unless the `--all` flag is set. The system SHALL reject combining a positional skill name with `--all`. When neither `--all` nor a skill name is provided, the system SHALL return a clear error instructing the user to provide a skill name or pass `--all`. Argument validation SHALL happen before any catalog fetch or filesystem work.

#### Scenario: No args and no --all
- **WHEN** user runs `devpilot skill add` with no arguments and no flags
- **THEN** the system returns an error stating that a skill name is required, or `--all` may be used to install the full catalog
- **AND** no catalog fetch or install is performed

#### Scenario: Cannot combine --all with a skill name
- **WHEN** user runs `devpilot skill add pm --all`
- **THEN** the system returns an error stating that `--all` cannot be combined with a skill name
- **AND** no catalog fetch or install is performed

#### Scenario: Single skill still works
- **WHEN** user runs `devpilot skill add pm`
- **THEN** the system installs only the `pm` skill (existing behavior is preserved)

