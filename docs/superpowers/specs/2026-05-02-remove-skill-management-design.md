# Remove in-repo skill management

## Goal

Skills are now distributed and installed via npx. The Go CLI no longer needs
to fetch the catalog from GitHub, install skills into `.claude/skills/`, or
expose `devpilot skill ...` subcommands. Remove that code so the CLI surface
matches reality.

## Scope

### Deleted

- `internal/skillmgr/` (entire package): catalog, github fetch, install,
  select, commands, all tests.
- `cmd/devpilot/main.go`: the `skillmgr.RegisterCommands(rootCmd)` call and
  its import. `devpilot skill ...` subcommands disappear.
- `internal/initcmd/generate.go`: `InstallSkills`, `SkillInstallOpts`, the
  `skillmgr` import.
- `internal/initcmd/commands.go`: the `InstallSkills(...)` invocation in the
  init flow.
- `internal/initcmd/generate_test.go`: `TestInstallSkills_*` tests.
- `internal/project/config.go`: `SkillEntry` type, `Config.Skills` field,
  `Config.UpsertSkill` method.
- `internal/project/config_test.go`: `TestUpsertSkill*` tests and any
  `Skills:` usage in fixtures.

### Kept (with edits)

- `internal/initcmd/detect.go`: keep `HasSkills` detection — it just looks
  for `.claude/skills/*/SKILL.md` on disk and is a useful signal for the
  init flow's messaging. Replace `skillmgr.InstallDir` with the inline
  string `".claude/skills"` and drop the `skillmgr` import.
- `internal/initcmd/commands.go`: the read-only `s.HasSkills` branches at
  lines 140 / 150 stay; only the install call goes.
- `internal/initcmd/detect_test.go`: tests already drive `HasSkills` via
  filesystem fixtures, no changes needed beyond confirming they still pass.

### Init flow after the change

`devpilot init` continues to:
- detect repo state (git, board config, existing CLAUDE.md, existing
  installed skills),
- scaffold `.devpilot.yaml`, `.gitignore` entries, `CLAUDE.md`.

It no longer offers an interactive skill-selection step. The post-init
"next steps" output gains one line pointing users at the npx-based
installer for skills (exact command lifted from the current README).

### Out of scope

- No migration of users' existing `.claude/skills/` installs. Files stay
  on disk; we just stop managing them from Go.
- No changes to the `skills/` catalog directory or `skills/index.json`.
- No deprecation period for `devpilot skill` subcommands — they're removed
  outright. (Acceptable per user: distribution has already moved to npx.)

## Backward compatibility

Removing `Config.Skills` means old `.devpilot.yaml` files that contain a
`skills:` block will fail strict YAML unmarshal if `yaml.v3` is in
known-fields mode. Verify the project loader uses default (lenient)
unmarshal — unknown keys are ignored — so old configs continue to load
cleanly with the `skills:` block silently dropped on next save. If the
loader uses `KnownFields(true)`, switch it off as part of this change.

## Verification

- `go build ./...` clean.
- `make test` clean.
- `make lint` clean.
- `grep -r skillmgr` returns no hits.
- `devpilot --help` no longer lists a `skill` subcommand.
- `devpilot init` in a fresh dir scaffolds files and prints the npx
  pointer; no network calls to GitHub.
- Loading a `.devpilot.yaml` that still contains a `skills:` block does
  not error.

## Doc updates

- `CLAUDE.md` repo map: drop `skillmgr` from the domain list.
- `docs/cli-reference.md`: remove the `devpilot skill` section if present.
- `README.md`: if it documents `devpilot skill install` or the
  init-time skill picker, replace with the npx instructions.
