## ADDED Requirements

### Requirement: Use main branch as default ref
The system SHALL use `"main"` as the default git ref when fetching the skill catalog and skill files from `raw.githubusercontent.com`, instead of resolving a release tag.

#### Scenario: Default ref for catalog and file fetching
- **WHEN** no explicit ref is provided (e.g., `devpilot skill add pm` without `@ref`)
- **THEN** the system SHALL use `"main"` as the ref in all `raw.githubusercontent.com` URLs

