## Context

Today `devpilot skill add` installs exactly one skill per invocation, identified by a required positional argument, and interactively prompts the user to pick install level (project vs user). For first-time setup and for scripted/CI usage this is awkward: users repeat the command 13+ times, and non-TTY callers can't choose user level at all (the prompt is skipped and project is forced).

The skill catalog is already fetched once by `FetchCatalog` (used by `skill list`). Reusing that fetch for bulk install is straightforward — the remaining work is argument parsing, flag handling, and a batch install loop with error aggregation.

## Goals / Non-Goals

**Goals:**
- Single-command install of the entire catalog via `devpilot skill add --all`.
- Non-interactive level selection via `devpilot skill add --level project|user`.
- Preserve current default behavior: `devpilot skill add <name>` with no flags still prompts interactively in a TTY and defaults to project.
- Batch install is resilient: a single failing skill does not abort the remaining installs.

**Non-Goals:**
- No `--all` pinned to a ref per skill — `--all` uses the default ref (`main`). Per-skill `@ref` syntax is not supported with `--all`.
- No uninstall / remove-all command.
- No parallel downloads; sequential is fine for ~13 skills.
- No change to catalog format, fetch layer, or `project` config schema.

## Decisions

### 1. Flag shape: `--all` plus `--level`, not a separate subcommand
Alternatives considered: a `devpilot skill add-all` subcommand, or an `--install-all` flag on `skill list`. Keeping everything under `skill add` matches user intuition ("I want to add skills") and reuses the existing install pipeline. `--level` is orthogonal and works with both single and bulk mode.

### 2. Argument validation: custom validator replacing `ExactArgs(1)`
The current `Args: cobra.ExactArgs(1)` is replaced with a custom function that enforces:
- `--all` + positional arg → error ("cannot combine --all with a skill name")
- no `--all` + no positional arg → error ("skill name required, or use --all")
- `--all` + no positional arg → ok
- no `--all` + 1 positional arg → ok (existing behavior)

Rationale: Cobra's built-in validators can't express "exactly one of flag-or-arg". A custom validator keeps the logic in one place and produces clear errors.

### 3. Level resolution precedence
`--level` flag value (if set) > interactive prompt (if TTY) > project default (non-TTY fallback). Accepted values: `project`, `user`. Any other value → error at flag parse time. This replaces the conditional prompt inside `skillAddCmd.RunE` with a single `resolveInstallLevel(flag, reader)` helper that both single and bulk paths call.

### 4. Bulk install: continue-on-error with summary
Instead of aborting on the first failing skill, `--all` iterates the catalog, installs each skill, collects errors into a slice, and at the end prints:

```
Installed 11/13 skills into .claude/skills/
Failed: devpilot-foo (fetching skill: 404), devpilot-bar (writing file: permission denied)
```

Exit code is 0 if all succeed, non-zero if any failed. Rationale: a transient 404 on one skill shouldn't block the other 12; the user can re-run `skill add <name>` for the failures.

### 5. Catalog fetch reuse
`--all` calls `fetchCatalogFn(ctx, defaultOwner, defaultRepo, defaultRef)` — the same hook already used by `skill list` and already stubbed in tests. This means bulk install is testable with zero new network mocking.

### 6. Config writes batched per level
For `--all`, load config once, `UpsertSkill` for each successful install, save once at the end. Avoids 13 read-modify-write cycles on `.devpilot.yaml`.

## Risks / Trade-offs

- **Partial failure leaves config and filesystem in a mixed state** → Mitigation: the summary output names every failed skill so the user can see exactly what needs retrying. Successfully installed skills are still recorded in config, which matches single-install semantics.
- **Catalog drift between fetch and install** → Not mitigated; catalog is fetched once at the start of `--all`. Acceptable since the catalog changes rarely and any drift affects only one invocation.
- **`--level user` with no `~/.config/devpilot/` yet** → `project.Save` auto-creates it (already true for single install); no new code path.
- **User passes `--level` with invalid value** → Cobra validates at parse time via a custom `cobra.Command.PreRunE` check or a string-enum pattern; emit a clear "must be 'project' or 'user'" error.
