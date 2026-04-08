## Context

The skill management flow currently resolves the latest release tag via the GitHub Releases API, then uses that tag as the git ref for `raw.githubusercontent.com` URLs. This API call is rate-limited (60 req/hr unauthenticated) and is the last GitHub API dependency in the skill system. Since users always want the latest version, version tags add complexity without value.

## Goals / Non-Goals

**Goals:**
- Eliminate all GitHub API dependencies from skill management by fetching from `main` branch directly
- Simplify the data model by removing version tracking (keep only install timestamp)
- Maintain `@ref` pinning for advanced use cases

**Non-Goals:**
- Version comparison or upgrade detection (not needed — always install latest)
- Caching fetched skills locally

## Decisions

### 1. Always fetch from `main` branch

Instead of resolving a release tag then fetching at that tag, fetch directly from `main`:
```
https://raw.githubusercontent.com/{owner}/{repo}/main/skills/index.json
https://raw.githubusercontent.com/{owner}/{repo}/main/skills/{name}/{file}
```

This eliminates the `FetchLatestTag` function entirely. The `@ref` syntax (`skill add pm@v0.4.0`) still works — it just uses the user-specified ref instead of `main`.

**Why this over alternatives:**
- **vs. redirect-based tag resolution**: Still requires an extra HTTP call and depends on github.com redirect behavior. Unnecessary since we don't need tags.
- **vs. `latest` marker file**: Adds release process complexity for no benefit.

### 2. Remove `Version` from `SkillEntry`

The `Version` field in `SkillEntry` currently stores a release tag (e.g., `v0.13.0`). Since we're no longer tracking versions, remove the field. The `InstalledAt` timestamp already exists and is sufficient.

**Migration:** YAML unmarshalling in Go silently ignores fields not present in the struct, so existing `.devpilot.yaml` files with a `version` key will load without error — the field is simply dropped.

### 3. Replace VERSION column with INSTALLED in list output

Both `skill list` and `skill list --installed` currently show a VERSION column. Replace it with INSTALLED showing the `InstalledAt` timestamp formatted as `2006-01-02`.

### 4. Use `"main"` as default ref constant

Define `const defaultRef = "main"` in `github.go` alongside the existing constants. This is used by both `skill add` and `skill list` when no ref is specified.

## Risks / Trade-offs

- **[Stale main branch]** → If `main` has unreleased/broken skill changes, users get them. Mitigation: skills are tested in CI before merging to main. This is acceptable since the catalog is a curated set of files.
- **[Lost version info]** → Existing installed skills lose their version record. Mitigation: version tracking was only cosmetic; `InstalledAt` provides the useful signal (when was this installed?).
