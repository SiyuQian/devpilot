## Why

When users install devpilot into a new project, there is no way to bring in the built-in skills that ship with devpilot — they must be manually copied. This makes it hard to benefit from the skill library without cloning the devpilot repo itself.

## What Changes

- **BREAKING**: `.devpilot.json` renamed to `.devpilot.yaml` (no backward compatibility); config is now YAML with an added `skills` tracking section
- New `devpilot skill add <name>[@version]` command fetches a named skill from `github.com/siyuqian/devpilot` at the latest release tag (or a pinned tag) and installs it into `.claude/skills/<name>/`
- New `devpilot skill list` command shows installed skills with their source and version
- `devpilot init` gains a multi-select checklist step to install skills from the devpilot catalog during project setup
- Installed skills are tracked in `.devpilot.yaml` with name, source, version, and installedAt timestamp
- Re-running `devpilot skill add` on an existing skill overwrites files silently (no prompt, no flag)

## Capabilities

### New Capabilities

- `skill-add`: Fetch and install a single named skill from a GitHub source at a specific tag into the current project
- `skill-list`: List skills installed in the current project with source and version metadata
- `skill-init-selection`: Interactive multi-select checklist in `devpilot init` for installing skills during project setup

### Modified Capabilities

- `project-config`: Config file changes from `.devpilot.json` (JSON) to `.devpilot.yaml` (YAML) and gains a `skills` array field for tracking installed skills

## Impact

- `internal/project/config.go` — swap JSON for YAML, rename file constant, add `SkillEntry` struct and `Skills []SkillEntry` to `Config`
- `internal/initcmd/` — update all references to config file; add skill selection step in `generate.go`
- New package `internal/skillmgr/` — GitHub fetching logic, catalog manifest, install/overwrite logic
- New `internal/skillmgr/commands.go` — Cobra `skill` subcommand with `add` and `list` subcommands
- `cmd/devpilot/main.go` — register new `skill` command
- Dependency: `gopkg.in/yaml.v3` added to `go.mod`
- No external API auth required (devpilot repo is public); GitHub Contents API used for fetching
