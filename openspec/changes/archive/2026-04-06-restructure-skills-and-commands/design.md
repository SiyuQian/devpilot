## Context

Skills currently reside in `.claude/skills/`, following Claude Code's convention for project-local skills. The `skillmgr` package hardcodes `.claude/skills/` as both the remote catalog path (GitHub API) and local install path.

Current directory structure:
```
.claude/skills/
├── confluence-reviewer/    # product skill
├── content-creator/        # product skill
├── google-go-style/        # product skill
├── learn/                  # product skill
├── news-digest/            # product skill
├── openspec-apply-change/  # project tooling (stays)
├── openspec-archive-change/# project tooling (stays)
├── openspec-explore/       # project tooling (stays)
├── openspec-propose/       # project tooling (stays)
├── skill-creator/          # project tooling (stays)
├── pm/                     # product skill
├── pr-creator/             # product skill
├── task-executor/          # product skill
├── task-refiner/           # product skill
└── trello/                 # product skill
```

## Goals / Non-Goals

**Goals:**
- Move distributable product skills to `skills/` at the project root with `devpilot-` prefix
- Update `skillmgr` to use `skills/` as install target and `skills/` as remote catalog source
- Update `initcmd` to detect/scaffold skills in the new location

**Non-Goals:**
- Changing skill content or behavior — this is purely a relocation
- Moving OpenSpec skills or `skill-creator` — they are project tooling and stay in `.claude/skills/`
- Changing the `devpilot sync` command or `internal/openspec/` package — out of scope for this change

## Decisions

### 1. Root-level `skills/` directory

Product skills move to `skills/` at the repo root. This separates "skills we ship" from "skills we use for development" (OpenSpec skills in `.claude/skills/`).

**Alternative considered**: `pkg/skills/` or `.devpilot/skills/` — rejected because `skills/` is simpler and aligns with the repo being a skills collection.

### 2. `devpilot-` prefix on all product skills

Every skill in `skills/` gets a `devpilot-` prefix. This namespaces them for discoverability when installed into other projects.

**Alternative considered**: No prefix — rejected because skills installed in user projects would have generic names like `learn` or `pm` that could collide.

### 3. Remove openspec exclude logic from catalog

Currently `catalog.go` has `excludedPrefixes = []string{"openspec-"}` and an `isExcluded()` function to filter openspec skills out of the catalog. After moving product skills to `skills/`, this directory will never contain openspec skills, making the exclude logic dead code. Remove `excludedPrefixes`, `isExcluded()`, the filter call in `listSkillDirs`, and related tests.

**Alternative considered**: Keep the exclude logic as a safety net — rejected because it masks a structural guarantee (separate directories) with runtime filtering, which is confusing for future developers.

### 4. Remote catalog path follows local path

The `skillmgr` catalog fetches from GitHub at `.claude/skills/` path. After this change, it fetches from `skills/` to match the new local structure. There are two hardcoded paths in `catalog.go`: the directory listing (`listSkillDirs`, line 71) and individual SKILL.md fetch (`fetchSkillMeta`, line 133). Both must be updated. Similarly, `github.go` line 73 (`basePath`) and the `SkillFile.Path` comment (line 22) need updating.

## Risks / Trade-offs

- **[Breaking change]** Skill rename → Anyone referencing old skill names must update. Mitigation: skills are referenced by name in Claude Code conversations, not by imports — impact is limited to muscle memory.
- **[Path change in tests]** Many test fixtures reference `.claude/skills/` → Must update all test paths. Mitigation: straightforward find-and-replace.
