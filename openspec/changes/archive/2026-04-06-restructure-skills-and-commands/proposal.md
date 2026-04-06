## Why

DevPilot skills currently live under `.claude/skills/`, which is a Claude Code convention for project-local skills. As DevPilot matures into a distributable skill collection, skills should live at the project root level (`skills/`) to clearly signal they are the product, not just tooling.

## What Changes

- **BREAKING**: Move all non-OpenSpec skills from `.claude/skills/` to a new root-level `skills/` directory.
- Rename all moved skills with a `devpilot-` prefix (e.g., `learn` → `devpilot-learn`, `trello` → `devpilot-trello`).
- Remove `excludedPrefixes` / `isExcluded()` filtering logic from `internal/skillmgr/catalog.go` — no longer needed since `skills/` only contains product skills.
- Update `internal/skillmgr/` to discover and install skills into `skills/` instead of `.claude/skills/`.
- Update `internal/initcmd/` skill scaffolding to use the new path.
- OpenSpec skills (`openspec-explore`, `openspec-apply-change`, `openspec-archive-change`, `openspec-propose`) and `skill-creator` remain in `.claude/skills/` — they are project tooling, not distributable product skills.
- Delete `.claude/commands/opsx/` directory — these are redundant command aliases that duplicate the OpenSpec skills already in `.claude/skills/`.

### Skills to move and rename

| Current path | New path |
|---|---|
| `.claude/skills/confluence-reviewer/` | `skills/devpilot-confluence-reviewer/` |
| `.claude/skills/content-creator/` | `skills/devpilot-content-creator/` |
| `.claude/skills/google-go-style/` | `skills/devpilot-google-go-style/` |
| `.claude/skills/learn/` | `skills/devpilot-learn/` |
| `.claude/skills/news-digest/` | `skills/devpilot-news-digest/` |
| `.claude/skills/pm/` | `skills/devpilot-pm/` |
| `.claude/skills/pr-creator/` | `skills/devpilot-pr-creator/` |
| `.claude/skills/task-executor/` | `skills/devpilot-task-executor/` |
| `.claude/skills/task-refiner/` | `skills/devpilot-task-refiner/` |
| `.claude/skills/trello/` | `skills/devpilot-trello/` |

## Capabilities

### New Capabilities

- `skill-relocation`: Moving skills from `.claude/skills/` to root `skills/` directory with `devpilot-` prefix and updating all references.

### Modified Capabilities

_(none — no existing spec-level requirements are changing)_

## Impact

- **Commands**: `.claude/commands/opsx/` directory removed (4 files: `explore.md`, `apply.md`, `archive.md`, `propose.md`).
- **Go packages**: `internal/skillmgr/` paths updated (install, catalog, GitHub fetch); `excludedPrefixes`/`isExcluded()` dead code removed. `internal/initcmd/` skill detection and scaffolding paths updated.
- **Tests**: All tests referencing `.claude/skills/` paths need updating.
- **CI/CD**: No workflow changes needed.
- **CLAUDE.md**: Repository structure docs need updating.
- **Downstream users**: Anyone referencing skill names without `devpilot-` prefix must update.
