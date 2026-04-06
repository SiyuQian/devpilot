## 1. Separate catalog and install paths

- [x] 1.1 In `internal/skillmgr/github.go`, rename `SkillsDir` to `CatalogDir` (used for GitHub fetch path) and add `InstallDir = ".claude/skills"` (used for local install)
- [x] 1.2 Update `fetchSkillFromBase` to use `CatalogDir`
- [x] 1.3 Update `InstallSkill` in `install.go` to use `InstallDir` instead of `SkillsDir`
- [x] 1.4 Update the success message in `commands.go` to show `.claude/skills/<name>/`

## 2. Fix tests

- [x] 2.1 Update any tests that assert on the install path to expect `.claude/skills/`
- [x] 2.2 Run `make test` and `make lint` to verify

## 3. Update references

- [x] 3.1 Update `SkillsDir` references in `select.go` or any other files that use the constant for install purposes
