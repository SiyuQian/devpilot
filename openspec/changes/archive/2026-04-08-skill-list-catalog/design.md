## Context

`devpilot skill list` currently reads `.devpilot.yaml` at project and user levels, displaying only installed skills. The `FetchCatalog` function in `catalog.go` already fetches all available skills from GitHub (name + description), but it's not wired into the list command.

## Goals / Non-Goals

**Goals:**
- `skill list` shows the full catalog with installation status by default
- Users can filter to installed-only with `--installed`
- Each skill shows: name, description, version (if installed), and level (if installed)

**Non-Goals:**
- Caching the catalog locally (acceptable to hit GitHub API each time; catalog is small)
- Showing skills from non-default sources
- Skill search/filter by keyword

## Decisions

### Merge catalog + installed data in the list command

Fetch catalog via `FetchCatalog`, load installed skills from both config levels, then merge by skill name. Display a unified table where uninstalled skills show a dash or "—" for version/level.

**Alternative**: Separate `skill search` command for catalog browsing. Rejected because the catalog is small (~13 skills) and a single view is more discoverable.

### Add `--installed` flag instead of a separate subcommand

The old behavior (installed-only) is preserved via `--installed`. This avoids breaking scripts that parse the output.

**Alternative**: Make `--installed` the default and add `--all` for catalog view. Rejected because the primary use case is discovery — users want to see what's available.

### Output format

Use the same tabwriter approach but with columns: `NAME | DESCRIPTION | VERSION | LEVEL`. Description is truncated to 40 chars to fit terminal width. Installed skills show version + level; uninstalled show "—".

## Risks / Trade-offs

- **[Network dependency]** → `skill list` now requires internet by default. The `--installed` flag works offline. Acceptable trade-off since the primary value is catalog discovery.
- **[API rate limiting]** → `FetchCatalog` makes 1 + N requests (directory listing + one per skill for SKILL.md). For ~13 skills this is fine. `catalog.go` already handles rate limit errors gracefully.
