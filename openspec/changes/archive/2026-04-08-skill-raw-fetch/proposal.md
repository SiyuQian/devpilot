## Why

The skill management commands (`skill list`, `skill add`) use unauthenticated GitHub REST API calls to discover and download skills. The unauthenticated rate limit is 60 requests/hour/IP. A single `skill list` with N skills costs 2+N API calls; `skill add` costs even more due to recursive directory listing. This makes the CLI fragile during development and unreliable for users behind shared IPs (offices, CI).

## What Changes

- Add a `skills/index.json` file to the repo that catalogs all available skills (name, description, file list)
- Replace GitHub REST API calls in `internal/skillmgr/` with HTTP fetches to `raw.githubusercontent.com` URLs, which have no rate limit
- Add a rule to `CLAUDE.md` requiring `index.json` to be updated whenever skills are added, removed, or modified

## Capabilities

### New Capabilities
- `skill-raw-fetch`: Fetching skill catalog and files via raw.githubusercontent.com + index.json instead of GitHub API

### Modified Capabilities
- `skill-catalog-list`: Catalog discovery now reads from index.json via raw URL instead of GitHub Contents API
- `skill-add`: File download now uses raw URLs derived from index.json instead of recursive API calls

## Impact

- `internal/skillmgr/github.go` — replace API calls with raw URL fetches
- `internal/skillmgr/catalog.go` — read index.json instead of listing directories via API
- `skills/index.json` — new file, must be maintained alongside skill changes
- `CLAUDE.md` — new rule for index.json maintenance
