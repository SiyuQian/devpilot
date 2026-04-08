## 1. User Config Dir Helper

- [x] 1.1 Add `UserConfigDir()` function to `internal/project/config.go` that returns `~/.config/devpilot/`
- [x] 1.2 Add tests for `UserConfigDir()` in `config_test.go`

## 2. Skill Add — User-Level Config Recording

- [x] 2.1 In `skillmgr/commands.go`, when `userLevel=true`, call `project.Load(userConfigDir)` and `project.Save(userConfigDir, cfg)` with the skill entry instead of skipping
- [x] 2.2 Skip the `.devpilot.yaml` existence check when user selects user-level install
- [x] 2.3 Add tests for user-level skill add writing to user config

## 3. Skill List — Show Both Levels

- [x] 3.1 Update `skill list` command to read user-level config via `project.Load(userConfigDir)` in addition to project config
- [x] 3.2 Add `LEVEL` column to table output showing `project` or `user`
- [x] 3.3 When no project `.devpilot.yaml` exists, show only user-level skills instead of erroring
- [x] 3.4 Add tests for listing skills from both levels
