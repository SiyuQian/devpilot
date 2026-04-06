## Purpose

Defines requirements for how DevPilot skills are organized, named, and discovered across the project directory structure.

## Requirements

### Requirement: Skills directory location
Product skills SHALL be stored in the `skills/` directory at the project root, not in `.claude/skills/`.

#### Scenario: Skill directory exists at root
- **WHEN** a user clones the devpilot repository
- **THEN** product skills are found under `skills/` with `devpilot-` prefixed directory names

### Requirement: Skill naming convention
All product skills SHALL use a `devpilot-` prefix in their directory name (e.g., `devpilot-learn`, `devpilot-trello`).

#### Scenario: Installed skill has prefix
- **WHEN** `devpilot skill add` installs a skill named `learn`
- **THEN** the skill is written to `skills/devpilot-learn/`

### Requirement: Skill install path
The `skillmgr` package SHALL install skills into `skills/<skillName>/` relative to the project root, instead of `.claude/skills/<skillName>/`.

#### Scenario: Install writes to correct directory
- **WHEN** `InstallSkill` is called with skill name `devpilot-pm`
- **THEN** files are written under `<destDir>/skills/devpilot-pm/`

### Requirement: Remote catalog path
The `skillmgr` catalog SHALL fetch skill listings from the `skills/` path in the remote GitHub repository instead of `.claude/skills/`.

#### Scenario: Catalog lists remote skills
- **WHEN** `FetchCatalog` queries the GitHub Contents API
- **THEN** it requests the path `skills/` (not `.claude/skills/`)

### Requirement: Remote skill fetch path
The `skillmgr` package SHALL fetch individual skill files from `skills/<skillName>/` in the remote GitHub repository instead of `.claude/skills/<skillName>/`.

#### Scenario: FetchSkill downloads from correct path
- **WHEN** `FetchSkill` is called to download skill `devpilot-pm`
- **THEN** it requests files under `skills/devpilot-pm/` (not `.claude/skills/devpilot-pm/`)

### Requirement: No openspec exclude logic in catalog
The `skillmgr` catalog SHALL NOT contain any filtering logic to exclude openspec-prefixed skills. The directory separation (`skills/` vs `.claude/skills/`) provides this guarantee structurally.

#### Scenario: Catalog has no exclude filtering
- **WHEN** `listSkillDirs` lists the contents of the `skills/` directory
- **THEN** all entries are returned without any prefix-based filtering

### Requirement: Init skill detection
The `initcmd` package SHALL detect existing skills in `skills/` instead of `.claude/skills/`.

#### Scenario: Init detects skills in new location
- **WHEN** `devpilot init` checks for existing skills
- **THEN** it scans the `skills/` directory at the project root

### Requirement: OpenSpec skills remain in .claude/skills
OpenSpec-related skills (`openspec-explore`, `openspec-apply-change`, `openspec-archive-change`, `openspec-propose`) and `skill-creator` SHALL remain in `.claude/skills/` and SHALL NOT be moved or renamed.

#### Scenario: OpenSpec skills untouched
- **WHEN** the restructure is complete
- **THEN** all four OpenSpec skills still exist in `.claude/skills/` with their original names
