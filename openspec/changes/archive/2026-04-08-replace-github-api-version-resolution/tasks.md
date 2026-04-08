## 1. Remove FetchLatestTag and add defaultRef

- [x] 1.1 Add `const defaultRef = "main"` to `internal/skillmgr/github.go`
- [x] 1.2 Remove `FetchLatestTag`, `fetchLatestTagFromURL` functions and `encoding/json` import from `github.go`
- [x] 1.3 Remove `TestFetchLatestTag`, `TestFetchLatestTagNotFound`, `TestFetchLatestTagRateLimit` from `github_test.go`

## 2. Update skill add to use main ref

- [x] 2.1 In `commands.go`: remove `fetchLatestTagFn` variable; replace version resolution block with `ref := defaultRef`; keep `@ref` parsing for pinned refs
- [x] 2.2 Update `skill add` output messages to not print version/ref
- [x] 2.3 Remove `Version` field from `cfg.UpsertSkill` call in add command

## 3. Update skill list to use main ref

- [x] 3.1 In `commands.go`: replace `fetchLatestTagFn` call in list command with `defaultRef`
- [x] 3.2 Replace VERSION column with INSTALLED column (formatted as `2006-01-02`) in both `printInstalledOnly` and `printCatalogView`

## 4. Remove Version from SkillEntry

- [x] 4.1 Remove `Version` field from `SkillEntry` struct in `internal/project/config.go`
- [x] 4.2 Update all tests in `internal/project/config_test.go` that reference `Version`

## 5. Update commands tests

- [x] 5.1 Update `stubCatalogFns` and any test helpers that mock `fetchLatestTagFn`
- [x] 5.2 Update test assertions that check VERSION column output

## 6. Verify

- [x] 6.1 Run `make test` — all tests pass
- [x] 6.2 Run `make lint` — no lint errors
