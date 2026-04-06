## 1. Move and rename skills

- [ ] 1.1 Create `skills/` directory at project root
- [ ] 1.2 Move each non-OpenSpec skill from `.claude/skills/<name>/` to `skills/devpilot-<name>/` (10 skills total)
- [ ] 1.3 Verify all 4 OpenSpec skills and `skill-creator` remain in `.claude/skills/` untouched
- [ ] 1.4 Remove the now-empty non-OpenSpec skill directories from `.claude/skills/`
- [ ] 1.5 Delete `.claude/commands/opsx/` directory (contains `explore.md`, `apply.md`, `archive.md`, `propose.md` — redundant with OpenSpec skills)

## 2. Update skillmgr package

- [ ] 2.1 Update `internal/skillmgr/install.go` — change install path from `.claude/skills/` to `skills/`
- [ ] 2.2 Update `internal/skillmgr/catalog.go` — change both GitHub API paths: `listSkillDirs` (line 71) and `fetchSkillMeta` (line 133) from `.claude/skills/` to `skills/`
- [ ] 2.3 Update `internal/skillmgr/github.go` — change `basePath` (line 73) from `.claude/skills/<skillName>` to `skills/<skillName>` and update `SkillFile.Path` comment (line 22)
- [ ] 2.4 Update `internal/skillmgr/commands.go` — change hardcoded output message at line 76 from `.claude/skills/%s/` to `skills/%s/`
- [ ] 2.5 Remove `excludedPrefixes`, `isExcluded()` function, and the filter call in `listSkillDirs` from `catalog.go` — dead code after directory separation
- [ ] 2.6 Update all tests in `internal/skillmgr/` to use new paths and remove `TestFetchCatalogExcludesOpenspec` and `TestIsExcluded`

## 3. Update initcmd package

- [ ] 3.1 Update `internal/initcmd/detect.go` — change skill detection path from `.claude/skills` to `skills`
- [ ] 3.2 Update `internal/initcmd/generate.go` — change skill scaffolding output path and hardcoded output message at line 369 from `.claude/skills/%s/` to `skills/%s/`
- [ ] 3.3 Update `internal/initcmd/generate_test.go` — fix test assertions for new path

## 4. Update documentation

- [ ] 4.1 Update `CLAUDE.md` — repository structure section and skill references
- [ ] 4.2 Verify `make test` and `make lint` pass
