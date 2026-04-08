## Context

User-level skills install to `~/.claude/skills/` but have no version tracking. Project-level installs record `SkillEntry` in `.devpilot.yaml` via `project.Load/Save`. The `Load`/`Save` functions already accept a `dir` parameter and `Save` creates intermediate directories — so reusing them for `~/.config/devpilot/` requires minimal code changes.

## Goals / Non-Goals

**Goals:**
- Track user-level skill versions in `~/.config/devpilot/.devpilot.yaml`
- Show user-level skills in `devpilot skill list` output
- Reuse existing `project.Config` and `SkillEntry` types

**Non-Goals:**
- Resolving conflicts when same skill exists at both levels (Claude Code's discovery handles this)
- Adding user-level config for non-skill settings (board, source, models)
- Adding `devpilot skill update` or auto-update functionality

## Decisions

**1. Config file location: `~/.config/devpilot/.devpilot.yaml`**

Same filename as project config, different directory. `project.Load("~/.config/devpilot/")` and `project.Save("~/.config/devpilot/", cfg)` work as-is. The `~/.config/devpilot/` directory is already used for runner logs.

Alternative considered: `~/.claude/skills.yaml` — rejected because it's Claude Code's directory and a different file format.

**2. User config dir resolution: `project.UserConfigDir()` helper**

Add a single function that returns `~/.config/devpilot/`. This centralizes the path and is used by both `skill add` and `skill list`.

**3. Skill list output: level column**

Add a `LEVEL` column to `devpilot skill list` showing `project` or `user`. List project-level skills first, then user-level. When not in a project directory, only show user-level skills (no error).

**4. No `.devpilot.yaml` requirement for user-level installs**

Currently `skill add` requires being in a project directory. For user-level installs, this check should be skipped — the user may want to install a personal skill without being in any project.

## Risks / Trade-offs

- [User config only stores skills] → The `Config` struct has `Board`, `Source`, `Models` fields that are meaningless at user level. They'll just be empty/omitted in the YAML. Acceptable — no need to create a separate type.
- [Same skill at both levels] → Claude Code may load both SKILL.md files. This is Claude Code's behavior, not ours to solve. `skill list` will show both entries clearly.
