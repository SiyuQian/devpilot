## Why

`FetchLatestTag` uses the GitHub Releases API (`api.github.com/repos/.../releases/latest`), which is rate-limited to 60 requests/hour for unauthenticated requests. This causes `devpilot skill list` and `devpilot skill add` to fail with "GitHub API rate limit exceeded." Skills should always install from the latest `main` branch via `raw.githubusercontent.com` — version tags are unnecessary since users always want the latest version.

## What Changes

- Remove `FetchLatestTag` entirely — no more version resolution step
- `skill add <name>` fetches from `main` branch instead of a release tag
- `skill add <name>@<ref>` still supported for pinning to any git ref
- `skill list` fetches catalog from `main` branch instead of latest tag
- `SkillEntry.Version` field removed from config — `InstalledAt` timestamp is sufficient
- List output replaces VERSION column with INSTALLED column showing install timestamp

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `skill-raw-fetch`: Remove tag resolution; always use `main` as the ref for raw URLs
- `skill-add`: Remove version resolution step; fetch from `main`; drop version from config
- `skill-list`: Replace VERSION column with INSTALLED timestamp column
- `skill-catalog-list`: Replace VERSION column with INSTALLED timestamp column
- `user-level-skill-config`: Remove `version` field from `SkillEntry`

## Impact

- `internal/skillmgr/github.go`: Remove `FetchLatestTag`, `fetchLatestTagFromURL`, and `encoding/json` import
- `internal/skillmgr/github_test.go`: Remove tag resolution tests
- `internal/skillmgr/commands.go`: Remove `fetchLatestTagFn`, use `"main"` as ref directly
- `internal/project/config.go`: Remove `Version` field from `SkillEntry`
- `internal/project/config_test.go`: Update tests that reference `Version`
- **BREAKING**: Existing `.devpilot.yaml` files with `version` field will have it ignored on load (YAML unmarshalling silently drops unknown fields when the struct field is removed)
