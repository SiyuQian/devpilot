## Context

The skill management system (`internal/skillmgr/`) currently uses the GitHub REST API (Contents API, Releases API) for all skill operations. These are unauthenticated calls with a 60 req/hour/IP rate limit. A single `skill list` with N skills costs 2+N API calls; `skill add` uses recursive directory listing + individual file downloads.

## Goals / Non-Goals

**Goals:**
- Eliminate GitHub REST API usage for skill catalog and file fetching
- Use `raw.githubusercontent.com` URLs which have no rate limit
- Maintain an `index.json` in the repo as the skill catalog source of truth
- Keep the same CLI UX — no user-facing behavior changes

**Non-Goals:**
- Changing the skill installation directory structure
- Adding authentication or token-based access
- Changing the config format in `.devpilot.yaml`
- Automating `index.json` generation via CI (maintained manually per CLAUDE.md rule)

## Decisions

### 1. Catalog index via `skills/index.json`

`raw.githubusercontent.com` cannot list directories, so we need a static index. The file lives at `skills/index.json` and contains an array of skill entries:

```json
{
  "skills": [
    {
      "name": "devpilot-pm",
      "description": "Product management skill for...",
      "files": ["SKILL.md", "references/guide.md"]
    }
  ]
}
```

**Why index.json over alternatives:**
- A single HTTP request fetches the entire catalog (vs. 2+N API calls)
- The `files` array lets `skill add` download exactly the right files without directory listing
- JSON is trivial to parse in Go

**Trade-off:** Index must be kept in sync manually. A CLAUDE.md rule ensures this. If index drifts, `skill list` shows stale data but nothing breaks catastrophically.

### 2. Raw URL pattern for all fetches

All HTTP fetches use:
```
https://raw.githubusercontent.com/{owner}/{repo}/{ref}/{path}
```

- `FetchLatestTag` — still needs the GitHub API (1 call) to resolve "latest" to a tag name. This is acceptable: it's a single call, not N calls. Alternatively, we could maintain a `latest` branch or file, but that adds complexity for minimal gain.
- `FetchCatalog` — single GET to `raw.githubusercontent.com/.../skills/index.json`
- `FetchSkill` — reads `files` from index, downloads each via raw URL

### 3. `FetchLatestTag` remains on GitHub API

The Releases API endpoint (`/repos/{owner}/{repo}/releases/latest`) is the only reliable way to get the latest tag. This is 1 API call regardless of skill count — acceptable within rate limits. No change needed here.

## Risks / Trade-offs

- **[Stale index]** → CLAUDE.md rule enforces updates; worst case is `skill list` showing outdated info, fixable by updating index.json
- **[raw.githubusercontent.com caching]** → GitHub CDN caches raw content for ~5 minutes; new releases may have brief delay. Acceptable for this use case.
- **[Breaking change for test mocks]** → Tests use `fetchCatalogFn` / `fetchLatestTagFn` overrides; these stay the same, only the default implementations change
