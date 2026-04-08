## 1. Remove CLAUDE.md from devpilot init

- [x] 1.1 Delete `internal/initcmd/templates.go` entirely
- [x] 1.2 Remove `HasClaudeMD` field from `Status` struct and its detection in `detect.go`
- [x] 1.3 Remove CLAUDE.md status line from `formatStatus()` in `commands.go`
- [x] 1.4 Remove `HasClaudeMD` check from `allConfigured()` in `commands.go`
- [x] 1.5 Remove CLAUDE.md generation block (the `shouldGenerate` call) from init command `Run` in `commands.go`
- [x] 1.6 Remove `GenerateClaudeMD`, `detectProjectType`, `parseGoModuleName`, `parsePackageJSONName` from `generate.go`
- [x] 1.7 Update tests in `commands_test.go`, `detect_test.go`, `generate_test.go` to remove CLAUDE.md references

## 2. Make task source configuration skippable

- [x] 2.1 Wrap the task source configuration block in `commands.go` with `shouldGenerate(opts, "Configure task source? [Y/n]: ")`
- [x] 2.2 Add test for skipping task source configuration

## 3. Add install level selection to devpilot skill add

- [x] 3.1 Change `InstallSkill` signature to accept absolute base directory instead of building path from `destDir + InstallDir`
- [x] 3.2 Update `InstallSkill` callers in `commands.go` (skillmgr) and `generate.go` (initcmd) to pass the resolved path
- [x] 3.3 Add interactive level prompt to `skillAddCmd` in `internal/skillmgr/commands.go`
- [x] 3.4 When user level is selected, install to `~/.claude/skills/` and skip `.devpilot.yaml` recording
- [x] 3.5 Add tests for level selection behavior and user-level install path

## 4. Verify

- [ ] 4.1 Run `make test` — all tests pass
- [ ] 4.2 Run `make lint` — no lint errors
