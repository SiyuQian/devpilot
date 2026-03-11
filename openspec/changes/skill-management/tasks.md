## 1. Config Migration (JSON → YAML)

- [x] 1.1 Add `gopkg.in/yaml.v3` dependency to `go.mod`
- [x] 1.2 Add `SkillEntry` struct to `internal/project/config.go` with fields: Name, Source, Version, InstalledAt
- [x] 1.3 Add `Skills []SkillEntry` field to `Config` struct with yaml tags
- [x] 1.4 Replace `encoding/json` with `gopkg.in/yaml.v3` in `internal/project/config.go`; rename `configFile` constant to `.devpilot.yaml`
- [x] 1.5 Update `Load`, `Save`, and `Exists` functions in `internal/project/config.go` for YAML
- [x] 1.6 Add `UpsertSkill(entry SkillEntry)` method to `Config` to add or update a skill entry by name
- [x] 1.7 Update all references to `.devpilot.json` across the codebase (initcmd, detect, generate, templates)
- [x] 1.8 Write tests for YAML load/save round-trip and `UpsertSkill`

## 2. Skill Manager Package

- [x] 2.1 Create `internal/skillmgr/` package directory
- [x] 2.2 Create `internal/skillmgr/catalog.go` with hardcoded `BuiltinCatalog []CatalogEntry` (name + description for each devpilot skill)
- [x] 2.3 Create `internal/skillmgr/github.go` — implement `FetchLatestTag(owner, repo string) (string, error)` using GitHub Releases API
- [x] 2.4 Implement `FetchSkill(owner, repo, skillName, tag string) ([]SkillFile, error)` using GitHub Contents API with recursive directory traversal
- [x] 2.5 Implement `InstallSkill(destDir, skillName string, files []SkillFile) error` — writes files to `.claude/skills/<name>/`, creates directories as needed, silently overwrites
- [x] 2.6 Write tests for `FetchLatestTag`, `FetchSkill`, and `InstallSkill` (use httptest for GitHub API mocking)

## 3. `devpilot skill` Command

- [x] 3.1 Create `internal/skillmgr/commands.go` with Cobra `skill` command and `add`, `list` subcommands
- [x] 3.2 Implement `skill add <name[@version]>` — parse optional `@version` suffix, resolve latest tag if omitted, fetch and install skill, upsert entry in `.devpilot.yaml`, print success
- [x] 3.3 Implement error cases for `skill add`: no args → error; skill not found → error; no `.devpilot.yaml` → error with init hint
- [x] 3.4 Implement `skill list` — load `.devpilot.yaml`, print table of installed skills (NAME / SOURCE / VERSION / INSTALLED), handle empty case
- [x] 3.5 Register `skill` command in `cmd/devpilot/main.go`
- [x] 3.6 Write tests for `skill add` and `skill list` command logic

## 4. `devpilot init` Skill Selection

- [x] 4.1 Add skill selection step to `internal/initcmd/generate.go` after board config step — present multi-select checklist using `BuiltinCatalog`
- [x] 4.2 Skip skill selection step when `-y` flag is set
- [x] 4.3 For each selected skill, call skillmgr install logic and upsert into config
- [x] 4.4 Update `internal/initcmd/detect.go` skill detection to reflect YAML config awareness if needed
- [x] 4.5 Write tests for init skill selection step (selected skills installed, `-y` skips step)

## 5. Verification

- [x] 5.1 Run `make test` — all tests pass
- [x] 5.2 Manual smoke test: `devpilot init` in a temp dir, select pm and trello, verify `.claude/skills/` files and `.devpilot.yaml` entries
- [x] 5.3 Manual smoke test: `devpilot skill add google-go-style` in configured project, verify files written and config updated
- [x] 5.4 Manual smoke test: `devpilot skill add nonexistent` returns clear error
- [x] 5.5 Manual smoke test: `devpilot skill list` shows installed skills table
