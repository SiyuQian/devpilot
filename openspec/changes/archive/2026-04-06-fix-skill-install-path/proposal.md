## Why

`devpilot skill add` currently installs skills into `skills/<name>/` at the project root. This is wrong — `skills/` is the catalog source directory in the devpilot repo itself. Installed skills should go to `.claude/skills/<name>/` so Claude Code can discover them as project-level skills. The existing spec (`skill-add`) already specifies `.claude/skills/` as the correct destination, but the implementation drifted.

## What Changes

- **BREAKING**: Change the install destination from `skills/<name>/` to `.claude/skills/<name>/` in the project directory
- Update the `SkillsDir` constant (or install logic) to use `.claude/skills` as the target path
- Keep the fetch source path as `skills/<name>` since that's where the catalog lives in the source repo

## Capabilities

### New Capabilities

_None_

### Modified Capabilities

- `skill-add`: Install destination changes from `skills/` to `.claude/skills/` to match the existing spec

## Impact

- `internal/skillmgr/install.go` — install path changes
- `internal/skillmgr/github.go` — may need to separate fetch source dir from install target dir (currently both use `SkillsDir`)
- `internal/skillmgr/commands.go` — success message path update
- Existing tests for skill installation
