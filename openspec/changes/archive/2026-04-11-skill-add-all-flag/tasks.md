## 1. Flag and argument plumbing

- [x] 1.1 Add `--all` (bool) and `--level` (string) flags to `skillAddCmd` in `internal/skillmgr/commands.go`
- [x] 1.2 Replace `Args: cobra.ExactArgs(1)` with a custom validator that enforces: name-xor-all, and errors clearly on `skill add` (no args, no --all) and `skill add pm --all`
- [x] 1.3 Validate `--level` value at PreRunE (accept `project`, `user`, or empty; any other value → error)

## 2. Level resolution helper

- [x] 2.1 Extract a `resolveInstallLevel(levelFlag string, projectDir string, reader *bufio.Reader) (baseDir string, userLevel bool, err error)` helper
- [x] 2.2 Make it apply the precedence: `--level` flag > interactive prompt (if reader != nil) > project default
- [x] 2.3 Update single-install path in `skillAddCmd.RunE` to call the new helper
- [x] 2.4 Keep `promptInstallLevel` as an internal helper used only when `--level` is unset and stdin is a TTY

## 3. Bulk install path

- [x] 3.1 When `--all` is set, call `fetchCatalogFn(ctx, defaultOwner, defaultRepo, defaultRef)` to load the catalog
- [x] 3.2 Resolve the install level ONCE via `resolveInstallLevel` and reuse it for every skill
- [x] 3.3 Load the target `.devpilot.yaml` once (project or user config dir)
- [x] 3.4 Iterate catalog entries: for each, call `FetchSkill` + `InstallSkill` and on success `cfg.UpsertSkill(...)`; on failure append `{name, err}` to a failures slice and continue
- [x] 3.5 Save the config once at the end
- [x] 3.6 Print a summary: `Installed N/M skills into <baseDir>` followed by any failed skills with their error messages
- [x] 3.7 Return a non-nil error (or set a non-zero exit) if any skill failed

## 4. Tests

- [x] 4.1 Unit test: arg validator rejects `skill add` (no args, no --all) and `skill add pm --all`; accepts `skill add pm` and `skill add --all`
- [x] 4.2 Unit test: `--level` flag parsing rejects invalid values with a clear error
- [x] 4.3 Unit test: `resolveInstallLevel` precedence (flag > prompt > default)
- [x] 4.4 Integration test (with `fetchCatalogFn` stub and a temp dir): `skill add --all --level project` installs every stubbed skill and writes all entries to `.devpilot.yaml`
- [x] 4.5 Integration test: bulk install with one failing skill reports the failure in the summary, installs the others, and exits non-zero
- [x] 4.6 Integration test: `skill add --all --level user` writes into the user config dir (override `userConfigDirFn` to a tempdir)
- [x] 4.7 Integration test: `skill add pm --level user` skips the interactive prompt entirely

## 5. Docs

- [x] 5.1 Update the `devpilot skill add` examples in `CLAUDE.md` to include `--all` and `--level`
- [x] 5.2 Update the `skill add` short/long help text in `commands.go` to mention the new flags
- [x] 5.3 Run `make lint` and `make test`; ensure both pass
