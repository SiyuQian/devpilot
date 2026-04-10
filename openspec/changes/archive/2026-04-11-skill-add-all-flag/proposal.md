## Why

Users who want to try DevPilot often want to install the entire skill catalog at once rather than running `devpilot skill add <name>` 13+ times. Additionally, automation and scripting need a non-interactive way to pick the install level (project vs user) instead of answering the interactive prompt.

## What Changes

- Add `--all` flag to `devpilot skill add` that installs every skill from the catalog in a single command.
- Add `--level` flag (values: `project`, `user`) to `devpilot skill add` that bypasses the interactive prompt and selects the install destination non-interactively.
- Relax `skill add` positional args: `<name>` is required unless `--all` is passed. Passing both `<name>` and `--all` is an error.
- With `--all`, iterate the catalog and install each skill; continue past individual failures and report a summary (installed / failed counts) at the end.
- When `--all` is used without `--level` in a non-interactive environment, default to project level (same as today). In an interactive TTY, prompt once for the level and apply it to the whole batch.

## Capabilities

### New Capabilities
<!-- None - all changes extend existing specs. -->

### Modified Capabilities
- `skill-add`: Add bulk install mode via `--all` and the argument-validation rules that go with it.
- `skill-level-selection`: Add non-interactive `--level` flag to select project or user level without prompting, and specify how it interacts with `--all`.

## Impact

- `internal/skillmgr/commands.go` — `skillAddCmd` gains `--all` and `--level` flags; args validation changes from `ExactArgs(1)` to a custom validator.
- `internal/skillmgr/commands_test.go` — new tests for bulk install, level flag, and arg validation.
- `CLAUDE.md` — update the CLI command examples for `devpilot skill add`.
- No changes to the catalog format, network fetch layer, or `project` config package.
