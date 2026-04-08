## ADDED Requirements

### Requirement: Maintain a skill catalog index file
The repo SHALL contain a `skills/index.json` file that lists all available skills with their name, description, and file paths. This file MUST be updated whenever skills are added, removed, or modified.

#### Scenario: Index file structure
- **WHEN** `skills/index.json` is read
- **THEN** it SHALL contain a JSON object with a `skills` array
- **AND** each entry SHALL have `name` (string), `description` (string), and `files` (array of relative file paths)

#### Scenario: Index reflects all catalog skills
- **WHEN** a new skill directory is added under `skills/`
- **THEN** a corresponding entry MUST be added to `skills/index.json` with the correct name, description, and file list

### Requirement: Fetch catalog via raw URL
The system SHALL fetch the skill catalog by downloading `skills/index.json` from `raw.githubusercontent.com/{owner}/{repo}/{ref}/skills/index.json` instead of using the GitHub Contents API.

#### Scenario: Successful catalog fetch
- **WHEN** the system fetches the catalog for a given ref
- **THEN** it SHALL make a single HTTP GET to the raw URL for `skills/index.json`
- **AND** parse the JSON to produce catalog entries

#### Scenario: Catalog fetch failure
- **WHEN** the raw URL returns a non-200 status
- **THEN** the system SHALL return an error describing the failure

### Requirement: Fetch skill files via raw URLs
The system SHALL download individual skill files from `raw.githubusercontent.com/{owner}/{repo}/{ref}/skills/{skillName}/{filePath}` instead of using the GitHub Contents API recursive directory listing.

#### Scenario: Download skill files from index
- **WHEN** `devpilot skill add <name>` is run
- **THEN** the system SHALL look up the skill in `index.json` to get its file list
- **AND** download each file from its raw URL
- **AND** return them as SkillFile entries

#### Scenario: Skill not found in index
- **WHEN** the requested skill name is not present in `index.json`
- **THEN** the system SHALL return an error indicating the skill was not found
