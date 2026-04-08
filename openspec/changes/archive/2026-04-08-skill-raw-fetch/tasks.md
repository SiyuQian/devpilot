## 1. Create skills/index.json

- [x] 1.1 Define `IndexEntry` struct in `internal/skillmgr/` with Name, Description, Files fields
- [x] 1.2 Scan all `skills/devpilot-*/` directories and generate `skills/index.json` with current skill data
- [x] 1.3 Add CLAUDE.md rule: "When adding, removing, or modifying skills in `skills/`, update `skills/index.json` accordingly"

## 2. Replace catalog fetch with raw URL

- [x] 2.1 Add `FetchIndex` function that downloads `skills/index.json` from `raw.githubusercontent.com/{owner}/{repo}/{ref}/skills/index.json` and parses it into `[]IndexEntry`
- [x] 2.2 Rewrite `FetchCatalog` to call `FetchIndex` and convert entries to `[]CatalogEntry` (drop the `files` field)
- [x] 2.3 Remove `listSkillDirs` and `fetchSkillMeta` functions (no longer needed)
- [x] 2.4 Update tests for `FetchCatalog` to use the new raw URL approach

## 3. Replace skill file fetch with raw URL

- [x] 3.1 Rewrite `FetchSkill` to: fetch index, look up skill by name, download each file from raw URL
- [x] 3.2 Remove `fetchContentsRecursive` function (no longer needed)
- [x] 3.3 Update tests for `FetchSkill` to use the new raw URL approach

## 4. Cleanup and verify

- [x] 4.1 Remove unused GitHub API imports and helper code from `github.go` and `catalog.go`
- [x] 4.2 Run `make test` and `make lint` to verify all tests pass
