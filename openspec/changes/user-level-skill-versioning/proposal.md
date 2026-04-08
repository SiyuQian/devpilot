## Why

User-level skill installs (`~/.claude/skills/`) currently have no version tracking. Project-level installs record version info in `.devpilot.yaml`, but user-level installs skip this entirely. This means there's no way to know what version of a user-level skill is installed, whether it needs updating, or to list it via `devpilot skill list`.

## What Changes

- Record user-level skill installs in `~/.config/devpilot/.devpilot.yaml` using the same `SkillEntry` format as project-level config
- `devpilot skill list` reads both project-level and user-level configs, showing all installed skills with their level indicated
- `devpilot skill add` at user level writes to `~/.config/devpilot/.devpilot.yaml` instead of silently skipping config recording
- Remove the requirement for `.devpilot.yaml` to exist when installing at user level (user may not be in a project directory)

## Capabilities

### New Capabilities

- `user-level-skill-config`: Reading and writing skill version entries in `~/.config/devpilot/.devpilot.yaml`

### Modified Capabilities

- `skill-add`: User-level installs now record version info in user-level config instead of skipping
- `skill-list`: Shows skills from both project and user level configs, with level indicator

## Impact

- `internal/project/config.go` — may need a helper to resolve user config dir path
- `internal/skillmgr/commands.go` — skill add writes user-level config; skill list reads both levels
- `openspec/specs/skill-add/spec.md` — scenarios for user-level install change
- `openspec/specs/skill-list/spec.md` — scenarios for listing user-level skills added
