## Why

`devpilot init` currently generates CLAUDE.md, which is a Claude Code project instruction file — not a devpilot concern. The init command should only manage devpilot's own configuration. Additionally, task source configuration cannot be skipped in interactive mode, and `devpilot skill add` only supports project-level installation with no option for user-level skills.

## What Changes

- **Remove CLAUDE.md generation** from `devpilot init` entirely (detect, generate, template, status display)
- **Make task source configuration skippable** in interactive mode with a "Configure task source? [Y/n]" prompt
- **Add install-level selection** to `devpilot skill add` — interactive prompt to choose between project (`.claude/skills/`) and user (`~/.claude/skills/`) level

## Capabilities

### New Capabilities
- `skill-level-selection`: Interactive level selection (project vs user) when installing skills via `devpilot skill add`

### Modified Capabilities
- `skill-init-selection`: Task source configuration becomes skippable during `devpilot init`
- `skill-add`: `InstallSkill` supports a configurable base directory instead of hardcoded project-level path

## Impact

- `internal/initcmd/` — Remove CLAUDE.md detection, generation, template; add skip prompt for task source
- `internal/skillmgr/` — `InstallSkill` accepts target directory; `skill add` command gains interactive level prompt
- `internal/initcmd/templates.go` — Delete entire file
- Tests in `initcmd` and `skillmgr` packages need updating
