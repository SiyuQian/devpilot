## Context

`devpilot init` currently handles CLAUDE.md generation (detection, templating, project type inference). This is outside devpilot's responsibility — CLAUDE.md belongs to Claude Code. The init command also forces task source configuration with no way to skip it. Separately, `devpilot skill add` only installs to the project-level `.claude/skills/` directory, but Claude Code also supports user-level skills at `~/.claude/skills/`.

## Goals / Non-Goals

**Goals:**
- Remove all CLAUDE.md logic from `devpilot init` (detect, generate, template, status)
- Allow users to skip task source configuration during interactive init
- Let `devpilot skill add` install to either project or user level via interactive prompt

**Non-Goals:**
- Changing `devpilot init` skill installation to support user-level (stays project-only)
- Adding a `--user` flag to `devpilot skill add` (interactive-only for now)
- Modifying `devpilot skill list` to show user-level skills

## Decisions

### Decision 1: Delete templates.go entirely

The file only contains the CLAUDE.md template. With CLAUDE.md generation removed, the entire file is dead code. The helper functions `detectProjectType`, `parseGoModuleName`, and `parsePackageJSONName` in `generate.go` also become unused and should be deleted.

### Decision 2: Skip prompt placement for task source

Add `"Configure task source? [Y/n]: "` before the existing source selection logic in `commands.go`. Selecting "n" skips the entire source configuration block (both trello and github paths). This keeps the change minimal — one `shouldGenerate` call wrapping the existing block.

### Decision 3: InstallSkill takes an absolute base directory

Currently `InstallSkill(destDir, skillName, files)` builds the path as `destDir + InstallDir + skillName`. Change to accept the full base directory (either `.claude/skills` or `~/.claude/skills`) so the caller controls the target. The command layer handles prompting and path resolution.

### Decision 4: User-level skills not recorded in .devpilot.yaml

User-level skills are personal, not project-scoped. When installing to `~/.claude/skills/`, skip the `.devpilot.yaml` upsert. The install path alone determines the level.

### Decision 5: Interactive prompt uses numbered selection

```
Install level:
  1) Project (.claude/skills/)
  2) User (~/.claude/skills/)
Select [1]:
```

Default is 1 (project). Empty input = project. Non-TTY (piped input) defaults to project with no prompt.

## Risks / Trade-offs

- **User-level skills invisible to `skill list`**: `devpilot skill list` reads `.devpilot.yaml` and won't show user-level skills. Acceptable for now — user-level is a power-user feature.
- **No `--user` flag**: Users who want scripted/non-interactive install to user level can't do it yet. Can be added later if needed.
