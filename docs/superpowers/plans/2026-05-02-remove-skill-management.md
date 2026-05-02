# Remove In-Repo Skill Management — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete `internal/skillmgr/`, the `devpilot skill ...` subcommands, and `devpilot init`'s skill-install step, plus the `Skills` field on `project.Config`. Skill distribution is now done via `npx skills add siyuqian/devpilot`; the Go code path is dead.

**Architecture:** Pure-deletion change in dependency order so the build stays green at every commit. `internal/initcmd/` is detangled from `skillmgr` first; then `skillmgr` is deleted along with its main.go registration; then `project.Config` loses the `Skills` field; finally docs are synced. `gopkg.in/yaml.v3`'s default `Unmarshal` is lenient on unknown keys, so old `.devpilot.yaml` files containing a `skills:` block keep loading without error.

**Tech Stack:** Go 1.x, Cobra (CLI), `gopkg.in/yaml.v3`, `make test` / `make lint` (golangci-lint), GitHub issue [#101](https://github.com/SiyuQian/devpilot/issues/101).

**Spec:** [`docs/superpowers/specs/2026-05-02-remove-skill-management-design.md`](../specs/2026-05-02-remove-skill-management-design.md)

---

## File Structure

**Modified:**
- `internal/initcmd/detect.go` — drop `skillmgr` import, inline `".claude/skills"`.
- `internal/initcmd/generate.go` — delete `InstallSkills`, `SkillInstallOpts`, `skillmgr` import. Add a one-line npx pointer to the post-init "Done!" output (or do it in `commands.go` where `Done!` is printed; we'll keep it in `commands.go` because that's where the user-facing flow lives).
- `internal/initcmd/generate_test.go` — delete `TestInstallSkills_*`.
- `internal/initcmd/commands.go` — remove the `InstallSkills(...)` call site; print npx hint before `Done!`.
- `cmd/devpilot/main.go` — remove `skillmgr.RegisterCommands(rootCmd)` and the import.
- `internal/project/config.go` — remove `SkillEntry`, `Config.Skills`, `Config.UpsertSkill`.
- `internal/project/config_test.go` — remove `TestUpsertSkill*`. Add one regression test that loads a YAML containing a `skills:` block without erroring.
- `CLAUDE.md` — repo map drops `skillmgr`.
- `docs/cli-reference.md` — remove `devpilot skill` block and the `skillmgr` test commands.
- `README.md` — remove the skill-catalog feature bullet, the `devpilot skill` snippets, and the `skillmgr` test command.

**Deleted:**
- `internal/skillmgr/` (entire directory).

---

### Task 1: Detangle `initcmd/detect.go` from `skillmgr`

**Files:**
- Modify: `internal/initcmd/detect.go`
- Test: `internal/initcmd/detect_test.go` (no changes; existing tests already drive `HasSkills` via filesystem fixtures)

- [ ] **Step 1: Confirm the existing test passes today**

Run: `go test ./internal/initcmd/ -run TestDetect_HasSkills -v`
Expected: PASS (4 subtests).

- [ ] **Step 2: Inline the path constant in `detect.go`**

In `internal/initcmd/detect.go`, replace:

```go
import (
	...
	"github.com/siyuqian/devpilot/internal/skillmgr"
)
...
skillsDir := filepath.Join(dir, skillmgr.InstallDir)
```

with:

```go
// (remove the skillmgr import line; keep the rest of the import block intact)
...
skillsDir := filepath.Join(dir, ".claude/skills")
```

Do not add a new package-level constant — the literal is used exactly once and inlining keeps the file self-contained.

- [ ] **Step 3: Run the same test to verify it still passes**

Run: `go test ./internal/initcmd/ -run TestDetect_HasSkills -v`
Expected: PASS (4 subtests).

- [ ] **Step 4: Verify the package still builds**

Run: `go build ./internal/initcmd/`
Expected: no output, exit 0. (`generate.go` still imports `skillmgr` — that's expected, we'll fix it in Task 2.)

- [ ] **Step 5: Commit**

```bash
git add internal/initcmd/detect.go
git commit -m "refactor(initcmd): inline .claude/skills path in detect.go

Drops the skillmgr import from detect.go in preparation for removing
internal/skillmgr/. The path is used once and is now a literal.

Refs #101"
```

---

### Task 2: Remove `InstallSkills` and the init-time skill picker

**Files:**
- Modify: `internal/initcmd/generate.go` (delete `InstallSkills`, `SkillInstallOpts`, the `skillmgr` import, and any now-unused imports like `context`, `os`, `time` — re-check imports after the delete)
- Modify: `internal/initcmd/commands.go` (remove the `InstallSkills(...)` call; print an npx hint before the existing `"\nDone!"` line)
- Modify: `internal/initcmd/generate_test.go` (delete `TestInstallSkills_NonInteractiveSkips`, `TestInstallSkills_InteractiveInstalls`, `TestInstallSkills_NoSelection`, and any test helpers used only by them)

- [ ] **Step 1: Delete `InstallSkills` and `SkillInstallOpts` from `generate.go`**

In `internal/initcmd/generate.go`, remove the entire `SkillInstallOpts` struct (the block currently around lines 198–209) and the entire `InstallSkills` function (currently around lines 211–286). After the removal, run `goimports -w internal/initcmd/generate.go` (or rely on your editor) to drop now-unused imports — at minimum `"context"`, `"github.com/siyuqian/devpilot/internal/skillmgr"`, and `"time"` if not used elsewhere in the file. Keep any imports still used by remaining functions in the file (re-check by skimming the file end-to-end).

- [ ] **Step 2: Remove the call site and add the npx hint in `commands.go`**

In `internal/initcmd/commands.go`, locate the block:

```go
		// Install skills from devpilot catalog
		if opts.Interactive {
			if err := InstallSkills(opts, SkillInstallOpts{}); err != nil {
				fmt.Fprintf(os.Stderr, "  Error installing skills: %v\n", err)
			}
		}

		fmt.Println("\nDone!")
```

Replace it with:

```go
		fmt.Println("\nDone!")
		fmt.Println("\nTo install Claude Code skills, run:")
		fmt.Println("  npx skills add siyuqian/devpilot")
```

(One blank line between the success line and the npx hint, matching the existing terminal-output style.)

- [ ] **Step 3: Delete the `InstallSkills` tests**

In `internal/initcmd/generate_test.go`, delete the three test functions:
- `TestInstallSkills_NonInteractiveSkips`
- `TestInstallSkills_InteractiveInstalls`
- `TestInstallSkills_NoSelection`

If any local helper (e.g. fake `selectFn` / fake `fetchCatalogFn`) is defined in this file and only referenced by those tests, delete it too. Run `goimports -w internal/initcmd/generate_test.go` to drop unused test-only imports (likely `skillmgr`).

- [ ] **Step 4: Build and run all initcmd tests**

Run: `go build ./internal/initcmd/ && go test ./internal/initcmd/ -v`
Expected: build succeeds; all remaining tests pass. The package no longer imports `skillmgr`.

- [ ] **Step 5: Verify `skillmgr` is no longer referenced from `initcmd`**

Run: `grep -rn skillmgr internal/initcmd/`
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add internal/initcmd/
git commit -m "feat(initcmd): drop skill install step, point users at npx

devpilot init no longer fetches or installs skills. Prints a one-line
hint pointing at 'npx skills add siyuqian/devpilot' so users know where
the new install path lives. internal/skillmgr is now unreferenced from
internal/initcmd, ready for deletion in the next commit.

Refs #101"
```

---

### Task 3: Delete `internal/skillmgr/` and the `devpilot skill` subcommands

**Files:**
- Delete: `internal/skillmgr/` (entire directory)
- Modify: `cmd/devpilot/main.go` (remove `skillmgr.RegisterCommands(rootCmd)` and the import)

- [ ] **Step 1: Confirm `skillmgr` is no longer imported anywhere except `cmd/devpilot/main.go`**

Run: `grep -rn '"github.com/siyuqian/devpilot/internal/skillmgr"' --include="*.go"`
Expected: exactly one match — `cmd/devpilot/main.go:11`.

If any other file shows up, stop and revisit Task 1 / Task 2 — Task 3 is gated on this.

- [ ] **Step 2: Remove the import and registration from `main.go`**

In `cmd/devpilot/main.go`, delete the line:

```go
	"github.com/siyuqian/devpilot/internal/skillmgr"
```

…from the import block, and delete the line:

```go
	skillmgr.RegisterCommands(rootCmd)
```

Leave the surrounding `RegisterCommands` calls for other domains intact.

- [ ] **Step 3: Delete the package**

Run: `rm -rf internal/skillmgr/`

- [ ] **Step 4: Build the whole module**

Run: `go build ./...`
Expected: clean. (Note: `internal/project` still defines `SkillEntry` / `UpsertSkill`; that's expected and removed in Task 4.)

- [ ] **Step 5: Run the full test suite**

Run: `go test ./...`
Expected: all pass. (Tests in `internal/project` may still reference `SkillEntry`; Task 4 removes them.)

- [ ] **Step 6: Verify the CLI surface**

Run: `go run ./cmd/devpilot --help`
Expected: no `skill` subcommand listed.

- [ ] **Step 7: Confirm full repo grep is clean**

Run: `grep -rn skillmgr --include="*.go"`
Expected: no output.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "chore: delete internal/skillmgr and devpilot skill subcommands

Skills are distributed via 'npx skills add siyuqian/devpilot' now, so
the in-repo skill catalog/install/select code is dead. main.go no longer
registers the 'devpilot skill ...' subcommand tree.

Refs #101"
```

---

### Task 4: Drop `Skills` field from `project.Config`

**Files:**
- Modify: `internal/project/config.go` (remove `SkillEntry`, `Skills` field, `UpsertSkill`; check whether the `time` import is still used elsewhere — if not, remove it)
- Modify: `internal/project/config_test.go` (delete `TestUpsertSkillAdd`, `TestUpsertSkillUpdate`, `TestUpsertSkillMultiple`, and any `Skills:` field usage in fixture configs; add a regression test for backward-compat YAML loading)

- [ ] **Step 1: Write the regression test FIRST (TDD)**

In `internal/project/config_test.go`, add this test (place it after the existing `Load` tests; if there is no Load test, place it after the imports near the top of the file):

```go
func TestLoad_IgnoresLegacySkillsBlock(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `board: my-board
source: trello
skills:
  - name: devpilot-pr-review
    source: github.com/siyuqian/devpilot
    installedAt: 2026-01-15T10:00:00Z
`
	if err := os.WriteFile(filepath.Join(dir, ".devpilot.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error on legacy skills block: %v", err)
	}
	if cfg.Board != "my-board" {
		t.Errorf("Board = %q, want %q", cfg.Board, "my-board")
	}
	if cfg.Source != "trello" {
		t.Errorf("Source = %q, want %q", cfg.Source, "trello")
	}
}
```

If the test file does not already import `os` and `path/filepath`, add them. (They may already be imported by sibling tests — check first.)

- [ ] **Step 2: Run the new test against the current code to confirm it passes pre-deletion**

Run: `go test ./internal/project/ -run TestLoad_IgnoresLegacySkillsBlock -v`
Expected: PASS. (yaml.v3's `Unmarshal` is lenient by default, so this passes even before we remove the field — the point is to lock that behavior in *before* we remove the field, so we know the test is meaningful.)

- [ ] **Step 3: Delete `SkillEntry`, `Config.Skills`, and `Config.UpsertSkill`**

In `internal/project/config.go`:

- Delete the `SkillEntry` type declaration (currently lines 15–20):

  ```go
  // SkillEntry records an installed skill in the project config.
  type SkillEntry struct {
      Name        string    `yaml:"name"`
      Source      string    `yaml:"source"`
      InstalledAt time.Time `yaml:"installedAt"`
  }
  ```

- Delete the `Skills` field from the `Config` struct (line 28):

  ```go
  Skills             []SkillEntry      `yaml:"skills,omitempty"`
  ```

- Delete the `UpsertSkill` method (currently lines 54–62):

  ```go
  // UpsertSkill adds or updates a skill entry by name.
  func (c *Config) UpsertSkill(entry SkillEntry) {
      for i, s := range c.Skills {
          if s.Name == entry.Name {
              c.Skills[i] = entry
              return
          }
      }
      c.Skills = append(c.Skills, entry)
  }
  ```

- If `time` is no longer referenced anywhere in `config.go`, remove `"time"` from the import block.

- [ ] **Step 4: Delete the `UpsertSkill` tests**

In `internal/project/config_test.go`, delete:

- `TestUpsertSkillAdd`
- `TestUpsertSkillUpdate`
- `TestUpsertSkillMultiple`

Also remove any `Skills: []SkillEntry{...}` literal in other tests (e.g. fixture configs) and any reference to `SkillEntry`. Run `grep -n SkillEntry internal/project/config_test.go` after editing — expected: no output.

- [ ] **Step 5: Run the full project tests**

Run: `go test ./internal/project/ -v`
Expected: all pass, including `TestLoad_IgnoresLegacySkillsBlock`.

- [ ] **Step 6: Run the full module build and tests**

Run: `go build ./... && go test ./...`
Expected: all pass.

- [ ] **Step 7: Confirm no lingering references**

Run: `grep -rn 'SkillEntry\|UpsertSkill\|cfg\.Skills\|Config\.Skills' --include="*.go"`
Expected: no output.

- [ ] **Step 8: Commit**

```bash
git add internal/project/
git commit -m "chore(project): remove Skills field and SkillEntry type

The Skills/UpsertSkill API was only written by InstallSkills, which is
gone. yaml.v3's default Unmarshal ignores unknown keys, so existing
.devpilot.yaml files containing a 'skills:' block continue to load
cleanly — covered by a new regression test in TestLoad_IgnoresLegacySkillsBlock.

Refs #101"
```

---

### Task 5: Sync the docs

**Files:**
- Modify: `CLAUDE.md` (line 7 — drop `skillmgr` from the domain list)
- Modify: `docs/cli-reference.md` (remove the `devpilot skill` command block at lines 42–48 and the `skillmgr`-specific test invocations at lines 18–19)
- Modify: `README.md` (remove the skill-catalog feature bullet at line 11, the `devpilot skill list/add` quickstart block at lines 63–64, the `devpilot skill` rows in the command table at lines 85–87, and the `skillmgr` test command at line 177)

- [ ] **Step 1: Edit `CLAUDE.md`**

Change the domain list at line 7 from:

```
- `internal/<domain>/` — self-contained domains (`auth`, `trello`, `gmail`, `slack`, `initcmd`, `skillmgr`, `project`); each owns its Cobra commands in `commands.go`
```

to:

```
- `internal/<domain>/` — self-contained domains (`auth`, `trello`, `gmail`, `slack`, `initcmd`, `project`); each owns its Cobra commands in `commands.go`
```

- [ ] **Step 2: Edit `docs/cli-reference.md`**

- Remove lines 18–19 (the `internal/skillmgr/` test invocations). If those lines sit inside a larger "Testing" code fence with similar invocations for other packages, leave the rest of the fence intact.
- Remove the entire `devpilot skill ...` block (lines 42–48). If the block has its own subheading (e.g. `## Skills`), remove the heading too.
- Re-read the file end-to-end after editing to make sure no orphaned heading or lead-in sentence is left dangling.

- [ ] **Step 3: Edit `README.md`**

- Replace the feature bullet at line 11 (the one starting `**A skill catalog.**`) with a single bullet that points to npx, e.g.:

  ```markdown
  - **A skill catalog.** Install Claude Code skills via `npx skills add siyuqian/devpilot`. `devpilot init` picks sensible defaults for a new project based on detected stack.
  ```

  (Adjust wording to match the surrounding bullet style.)

- Remove the quickstart snippet at lines 63–64:

  ```
  devpilot skill list
  devpilot skill add devpilot-pr-review
  ```

  Replace with the npx command:

  ```
  npx skills add siyuqian/devpilot
  ```

- Remove the three `devpilot skill ...` rows from the command table at lines 85–87. Leave the surrounding table intact.

- Remove line 177 (the `go test ./internal/skillmgr/ ...` example). If it's part of a longer "running tests" code fence, leave the rest of the fence intact.

- [ ] **Step 4: Final sweep for stale references**

Run: `grep -rn 'devpilot skill\|skillmgr\|InstallSkills\|SkillEntry' --include="*.md" --include="*.go"`
Expected: no output. (References inside `docs/superpowers/specs/2026-05-02-remove-skill-management-design.md` and `docs/superpowers/plans/2026-05-02-remove-skill-management.md` are acceptable — they describe the deletion. If any match shows up there, that's fine; for any other path, fix it.)

- [ ] **Step 5: Commit**

```bash
git add CLAUDE.md docs/cli-reference.md README.md
git commit -m "docs: drop devpilot skill / skillmgr references

CLAUDE.md repo map, README quickstart + command table, and
docs/cli-reference.md no longer document the removed surface. README
points users at 'npx skills add siyuqian/devpilot' instead.

Refs #101"
```

---

### Task 6: Final verification

No code changes — this task is the verification gate before opening the PR.

- [ ] **Step 1: Confirm acceptance criteria from issue #101**

Run each check and confirm the expected output:

```bash
grep -rn skillmgr --include='*.go' --include='*.md'
```
Expected: only matches inside `docs/superpowers/specs/` and `docs/superpowers/plans/` (this plan + the spec). Any other hit is a bug — fix and re-commit.

```bash
go build ./...
```
Expected: no output, exit 0.

```bash
make test
```
Expected: all packages pass.

```bash
make lint
```
Expected: no findings.

```bash
go run ./cmd/devpilot --help
```
Expected: no `skill` subcommand in the listing.

- [ ] **Step 2: Smoke-test `devpilot init`**

```bash
TMP=$(mktemp -d) && cd "$TMP" && git init -q && go run /Users/siyu/Works/github.com/siyuqian/devpilot/cmd/devpilot init --non-interactive 2>&1 | tee /tmp/init.out ; cd - >/dev/null
```

Expected: scaffolds files, prints `Done!`, prints the npx hint. No network calls (no GitHub fetch). Exit 0.

If the init command requires `--interactive` for the npx hint to print, run the interactive variant in a TTY and confirm the hint appears at the end. Adjust the hint placement in `commands.go` if it's gated behind interactive mode unintentionally — the spec says the hint should print regardless.

- [ ] **Step 3: Smoke-test legacy `.devpilot.yaml` load**

```bash
TMP=$(mktemp -d) && cat > "$TMP/.devpilot.yaml" <<'YAML'
board: legacy-board
skills:
  - name: devpilot-pr-review
    source: github.com/siyuqian/devpilot
    installedAt: 2026-01-15T10:00:00Z
YAML
cd "$TMP" && go run /Users/siyu/Works/github.com/siyuqian/devpilot/cmd/devpilot --help >/dev/null && echo "load ok"
cd - >/dev/null
```

Expected: `load ok` (no parse error). The `--help` invocation alone doesn't load the config, so if you have a CLI command that actually loads `.devpilot.yaml` (e.g. a list/status command), prefer that. Otherwise rely on the `TestLoad_IgnoresLegacySkillsBlock` test added in Task 4 — note this in the PR description.

- [ ] **Step 4: Open the PR**

Push the branch and open a PR linking to issue #101. The PR title should be `chore: remove in-repo skill management (#101)` and the body should summarize the four logical commits and reference the spec.

```bash
git push -u origin <branch-name>
gh pr create --title "chore: remove in-repo skill management" --body "$(cat <<'EOF'
Closes #101.

Removes `internal/skillmgr/`, the `devpilot skill ...` subcommands, and
`devpilot init`'s skill-install step. Also drops `Skills` /
`UpsertSkill` from `project.Config` (yaml.v3's lenient `Unmarshal`
keeps old configs loadable; covered by a new regression test).

Spec: docs/superpowers/specs/2026-05-02-remove-skill-management-design.md
Plan: docs/superpowers/plans/2026-05-02-remove-skill-management.md

## Test plan
- [x] `go build ./...`
- [x] `make test`
- [x] `make lint`
- [x] `devpilot --help` no longer lists `skill`
- [x] `devpilot init` scaffolds and prints the npx hint
- [x] Legacy `.devpilot.yaml` with a `skills:` block loads without error (covered by TestLoad_IgnoresLegacySkillsBlock)
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- Spec "Deleted" — `internal/skillmgr/` entire package → Task 3.
- Spec "Deleted" — `cmd/devpilot/main.go` skillmgr import + RegisterCommands → Task 3.
- Spec "Deleted" — `internal/initcmd/generate.go` InstallSkills + SkillInstallOpts + skillmgr import → Task 2.
- Spec "Deleted" — `internal/initcmd/commands.go` InstallSkills call site → Task 2.
- Spec "Deleted" — `internal/initcmd/generate_test.go` TestInstallSkills_* → Task 2.
- Spec "Deleted" — `internal/project/config.go` SkillEntry / Skills / UpsertSkill → Task 4.
- Spec "Deleted" — `internal/project/config_test.go` TestUpsertSkill* → Task 4.
- Spec "Kept (with edits)" — `internal/initcmd/detect.go` inline path → Task 1.
- Spec "Init flow after the change" — npx pointer in init output → Task 2.
- Spec "Backward compatibility" — verify yaml loader is lenient (it is, `yaml.v3`'s `Unmarshal`) → Task 4 includes the regression test.
- Spec "Verification" — `grep -r skillmgr` clean, build/test/lint clean, `--help` clean, init smoke, legacy-yaml smoke → Task 6.
- Spec "Doc updates" — CLAUDE.md / cli-reference.md / README.md → Task 5.

All spec requirements have a task. No gaps.

**Placeholder scan:** No "TBD" / "implement later" / "handle edge cases" / "similar to Task N" entries. Each step has the actual code or command to run.

**Type consistency:** No new types or signatures introduced — this is a pure deletion. The one helper signature touched (`Config.UpsertSkill`) is being deleted, and the test name `TestLoad_IgnoresLegacySkillsBlock` is used consistently in Tasks 4 and 6.
