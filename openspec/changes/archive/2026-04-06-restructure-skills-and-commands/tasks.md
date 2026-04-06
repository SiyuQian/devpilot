## 1. Move and rename skills

- [x] 1.1 Create `skills/` directory at project root
- [x] 1.2 Move each non-OpenSpec skill from `.claude/skills/<name>/` to `skills/devpilot-<name>/` (10 skills total)
- [x] 1.3 Verify all 4 OpenSpec skills and `skill-creator` remain in `.claude/skills/` untouched
- [x] 1.4 Remove the now-empty non-OpenSpec skill directories from `.claude/skills/`
- [x] 1.5 Delete `.claude/commands/opsx/` directory (contains `explore.md`, `apply.md`, `archive.md`, `propose.md` — redundant with OpenSpec skills)

## 2. Update skillmgr package

- [x] 2.1 Update `internal/skillmgr/install.go` — change install path from `.claude/skills/` to `skills/`
- [x] 2.2 Update `internal/skillmgr/catalog.go` — change both GitHub API paths: `listSkillDirs` (line 71) and `fetchSkillMeta` (line 133) from `.claude/skills/` to `skills/`
- [x] 2.3 Update `internal/skillmgr/github.go` — change `basePath` (line 73) from `.claude/skills/<skillName>` to `skills/<skillName>` and update `SkillFile.Path` comment (line 22)
- [x] 2.4 Update `internal/skillmgr/commands.go` — change hardcoded output message at line 76 from `.claude/skills/%s/` to `skills/%s/`
- [x] 2.5 Remove `excludedPrefixes`, `isExcluded()` function, and the filter call in `listSkillDirs` from `catalog.go` — dead code after directory separation
- [x] 2.6 Update all tests in `internal/skillmgr/` to use new paths and remove `TestFetchCatalogExcludesOpenspec` and `TestIsExcluded`

## 3. Update initcmd package

- [x] 3.1 Update `internal/initcmd/detect.go` — change skill detection path from `.claude/skills` to `skills`
- [x] 3.2 Update `internal/initcmd/generate.go` — change skill scaffolding output path and hardcoded output message at line 369 from `.claude/skills/%s/` to `skills/%s/`
- [x] 3.3 Update `internal/initcmd/generate_test.go` — fix test assertions for new path

## 4. Update documentation

- [x] 4.1 Update `CLAUDE.md` — repository structure section and skill references
- [x] 4.2 Verify `make test` and `make lint` pass
