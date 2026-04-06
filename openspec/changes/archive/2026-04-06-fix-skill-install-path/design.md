## Context

`devpilot skill add` uses a single constant `SkillsDir = "skills"` for both the GitHub fetch path (source catalog) and the local install path. The source catalog correctly lives at `skills/` in the devpilot repo, but installed skills should go to `.claude/skills/` so Claude Code discovers them as project-level skills.

## Goals / Non-Goals

**Goals:**
- Install skills to `.claude/skills/<name>/` instead of `skills/<name>/`
- Keep fetching from `skills/<name>/` in the source repo (that's the catalog)

**Non-Goals:**
- Changing the fetch source path or catalog structure
- Migrating previously mis-installed skills

## Decisions

**Separate fetch source dir from install target dir.** Currently both use `SkillsDir`. Split into two constants: `CatalogDir = "skills"` (for GitHub fetch) and `InstallDir = ".claude/skills"` (for local install). Alternative: pass the install dir as a parameter — but a constant is simpler since the install location is fixed by Claude Code's conventions.

## Risks / Trade-offs

- [Breaking change for anyone who relied on `skills/` install path] → Low risk since the current behavior is a bug; no one should depend on it.
