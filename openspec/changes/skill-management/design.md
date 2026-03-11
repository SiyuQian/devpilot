## Context

DevPilot ships with a library of Claude Code skills in `.claude/skills/`, but these only exist in the devpilot repository itself. When a user runs `devpilot init` on their own project, they get a config file and a boilerplate skill scaffold — but none of the useful built-in skills (pm, trello, task-executor, etc.).

The current config format is `.devpilot.json` (JSON). There is no tracking of which skills are installed or where they came from.

## Goals / Non-Goals

**Goals:**
- Allow users to install devpilot's built-in skills into any project with a single command
- Track installed skills (name, source, version) in project config
- Expose skill selection during `devpilot init`
- Migrate config from JSON to YAML

**Non-Goals:**
- Supporting skill sources other than GitHub (local paths, npm-style registries)
- Skill dependency resolution
- Skill removal/uninstall command
- Backward compatibility with `.devpilot.json`

## Decisions

### 1. YAML over JSON for config

**Decision**: Migrate `.devpilot.json` → `.devpilot.yaml` using `gopkg.in/yaml.v3`.

**Rationale**: YAML is more human-readable and supports inline comments, which is valuable for a config file users are expected to edit. No backward compat needed per requirements.

**Alternative considered**: Keep JSON and add skills tracking there. Rejected because YAML was already preferred and this is a good forcing function.

### 2. GitHub Contents API for fetching skills

**Decision**: Use the GitHub Contents API (`https://api.github.com/repos/{owner}/{repo}/contents/{path}?ref={tag}`) to list and download skill files recursively.

**Rationale**: No git binary required. Works with any HTTP client. Public repos need no auth. Supports fetching at a specific tag ref.

**Alternative considered**: `git clone --sparse`. Requires git binary and is heavier than needed.

**Alternative considered**: Embed skills in the binary with `embed.FS`. Means skills can't be updated without upgrading the binary. Rejected.

### 3. Latest release tag as default version

**Decision**: When no version is specified, `devpilot skill add <name>` fetches the latest release tag via the GitHub Releases API, then uses that tag as the `ref` for contents fetching.

**Rationale**: Reproducible by default. Users know exactly what version was installed. Tag-based because devpilot does tagged releases.

**Format**: `devpilot skill add pm@v0.5.0` for pinning.

### 4. Hardcoded catalog manifest for `devpilot init` skill selection

**Decision**: The list of available skills shown during `devpilot init` is a hardcoded Go slice in `internal/skillmgr/catalog.go`.

**Rationale**: Keeps `init` fast and offline-capable. The catalog only changes when new skills are added to devpilot, which requires a new release anyway. Live-fetching the list from GitHub adds latency and a failure mode to every `devpilot init`.

### 5. Silent overwrite

**Decision**: `devpilot skill add` on an already-installed skill silently overwrites all files. No prompt, no `--force` flag.

**Rationale**: Simple mental model — the command always installs. Doubles as an update mechanism.

### 6. New `internal/skillmgr` package

**Decision**: All skill management logic lives in `internal/skillmgr/`: GitHub fetching, catalog manifest, install logic, and Cobra commands.

**Rationale**: Follows the existing convention where each domain owns its own commands.go.

## Risks / Trade-offs

- **GitHub API rate limiting** → Unauthenticated requests are limited to 60/hour per IP. Skill installs involve several API calls (get latest tag, list directory, download files). Unlikely to be hit in practice, but could affect CI environments. Mitigation: surface clear error message if rate limited.
- **Skill directory structure changes** → If a skill's files are reorganized in the devpilot repo, `skill add` will fetch the new structure. This is intentional (always gets latest at the specified tag) but could surprise users who expected idempotency. Mitigation: version pinning via `@tag` syntax.
- **No rollback** → If a skill add partially fails (e.g. network drops mid-download), some files may be written. Mitigation: on error, log which files were written. Full rollback is out of scope.

## Migration Plan

1. On first `devpilot init` run after upgrade, the wizard detects `.devpilot.json` does not exist (it was renamed) and offers to create `.devpilot.yaml` from scratch.
2. No automatic migration of existing `.devpilot.json` files — users on old projects need to rename manually or re-run `devpilot init`.
3. Document the breaking change in release notes.

## Open Questions

- Should `devpilot skill list --available` fetch the live catalog from GitHub, or always use the hardcoded manifest? Currently scoped to hardcoded only.
