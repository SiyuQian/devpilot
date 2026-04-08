## Why

`devpilot skill list` currently only shows installed skills. Users have no way to discover what skills are available in the catalog without browsing the GitHub repo. The command should show the full catalog with installation status, like `brew list` vs `brew search` — but combined into one view, since the catalog is small enough.

## What Changes

- **Redesign `skill list`** to fetch and display the full skill catalog from GitHub, showing each skill's name, description, and installation status (installed version or "not installed").
- **Add `--installed` flag** to filter to only installed skills (preserving the current behavior for scripts/automation).
- Existing `FetchCatalog` function already provides the catalog data; the change wires it into the list command.

## Capabilities

### New Capabilities
- `skill-catalog-list`: Display all available skills from the devpilot catalog with their installation status, description, and version info.

### Modified Capabilities
- `skill-list`: Current behavior becomes a subset (the `--installed` filter). The default output changes from installed-only to full catalog view.

## Impact

- `internal/skillmgr/commands.go` — `skillListCmd` rewritten
- `internal/skillmgr/catalog.go` — already has `FetchCatalog`, may need minor adjustments
- `internal/skillmgr/commands_test.go` — tests updated for new behavior
- No breaking API changes; CLI output format changes (users piping the old table format will need `--installed`)
