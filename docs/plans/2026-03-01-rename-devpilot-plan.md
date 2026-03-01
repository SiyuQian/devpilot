# Rename devpilot → DevPilot Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rename the project from "devpilot" / "devpilot" / "DevPilot" to "DevPilot" / "devpilot" across the entire codebase.

**Architecture:** Bulk text replacement in dependency order — Go module first (enables compilation), then source code, then config/scripts, then docs. Directory and file renames happen after text changes to avoid breaking paths mid-edit.

**Tech Stack:** Go, Cobra CLI, shell scripts, GitHub Actions YAML, Markdown docs

---

### Task 1: Update Go Module Path

**Files:**
- Modify: `go.mod:1`

**Step 1: Replace module path in go.mod**

Change line 1 from:
```
module github.com/siyuqian/devpilot
```
to:
```
module github.com/siyuqian/devpilot
```

**Step 2: Verify go.mod is valid**

Run: `head -1 go.mod`
Expected: `module github.com/siyuqian/devpilot`

**Step 3: Commit**

```bash
git add go.mod
git commit -m "chore: rename Go module to github.com/siyuqian/devpilot"
```

---

### Task 2: Update All Go Import Paths

**Files:**
- Modify: `cmd/devpilot/main.go` (lines 8-11)
- Modify: `internal/initcmd/commands.go` (lines 10-11)
- Modify: `internal/initcmd/detect.go` (lines 7-8)
- Modify: `internal/initcmd/detect_test.go` (line 8)
- Modify: `internal/initcmd/generate.go` (line 14)
- Modify: `internal/trello/commands.go` (lines 10-11)
- Modify: `internal/taskrunner/runner.go` (line 12)
- Modify: `internal/taskrunner/commands.go` (lines 15-17)
- Modify: `internal/taskrunner/priority.go` (line 7)
- Modify: `internal/taskrunner/priority_test.go` (find import line)

**Step 1: Replace all import paths**

In every `.go` file in the project, replace:
```
github.com/siyuqian/devpilot
```
with:
```
github.com/siyuqian/devpilot
```

This affects all `import` blocks. The exact files and lines:

`cmd/devpilot/main.go`:
```go
"github.com/siyuqian/devpilot/internal/auth"
"github.com/siyuqian/devpilot/internal/initcmd"
"github.com/siyuqian/devpilot/internal/taskrunner"
"github.com/siyuqian/devpilot/internal/trello"
```

`internal/initcmd/commands.go`:
```go
"github.com/siyuqian/devpilot/internal/auth"
"github.com/siyuqian/devpilot/internal/trello"
```

`internal/initcmd/detect.go`:
```go
"github.com/siyuqian/devpilot/internal/auth"
"github.com/siyuqian/devpilot/internal/project"
```

`internal/initcmd/detect_test.go`:
```go
"github.com/siyuqian/devpilot/internal/project"
```

`internal/initcmd/generate.go`:
```go
"github.com/siyuqian/devpilot/internal/project"
```

`internal/trello/commands.go`:
```go
"github.com/siyuqian/devpilot/internal/auth"
"github.com/siyuqian/devpilot/internal/project"
```

`internal/taskrunner/runner.go`:
```go
"github.com/siyuqian/devpilot/internal/trello"
```

`internal/taskrunner/commands.go`:
```go
"github.com/siyuqian/devpilot/internal/auth"
"github.com/siyuqian/devpilot/internal/project"
"github.com/siyuqian/devpilot/internal/trello"
```

`internal/taskrunner/priority.go`:
```go
"github.com/siyuqian/devpilot/internal/trello"
```

`internal/taskrunner/priority_test.go`: (find the import and replace similarly)

**Step 2: Verify no old imports remain**

Run: `grep -r "siyuqian/devpilot" --include="*.go" .`
Expected: no output

**Step 3: Commit**

```bash
git add -A
git commit -m "chore: update all Go import paths to github.com/siyuqian/devpilot"
```

---

### Task 3: Update Go Source String Constants and Messages

Replace all user-visible "devpilot" strings in Go source files.

**Files:**
- Modify: `cmd/devpilot/main.go` (lines 18, 20)
- Modify: `internal/auth/commands.go` (line 60)
- Modify: `internal/auth/credentials.go` (line 17)
- Modify: `internal/project/config.go` (lines 9, 11, 16, 33, 46)
- Modify: `internal/initcmd/commands.go` (lines 22, 88)
- Modify: `internal/initcmd/generate.go` (lines 127, 130, 211)
- Modify: `internal/trello/commands.go` (lines 38, 59)
- Modify: `internal/taskrunner/commands.go` (lines 57, 64)
- Modify: `internal/taskrunner/runner.go` (lines 258, 293, 321, 332)
- Modify: `internal/taskrunner/tui.go` (line 339)
- Modify: `internal/taskrunner/tui_view.go` (line 73)

**Step 1: Update CLI root command**

`cmd/devpilot/main.go` line 18:
```go
Use:   "devpilot",
```

`cmd/devpilot/main.go` line 20:
```go
Long:  "devpilot manages authentication and integrations for external services like Trello, GitHub, and more.",
```

**Step 2: Update auth package**

`internal/auth/commands.go` line 60:
```go
fmt.Printf("Run 'devpilot login <service>' to get started. Available: %s\n", AvailableNames())
```

`internal/auth/credentials.go` line 17:
```go
return filepath.Join(home, ".config", "devpilot")
```

**Step 3: Update project config**

`internal/project/config.go`:
- Line 9: `const configFile = ".devpilot.json"`
- Line 11: comment `// Config represents project-level configuration stored in .devpilot.json.`
- Line 16: comment `// Load reads .devpilot.json from dir.`
- Line 33: comment `// Save writes cfg to .devpilot.json in dir`
- Line 46: comment `// Exists checks if .devpilot.json exists in dir.`

**Step 4: Update initcmd package**

`internal/initcmd/commands.go`:
- Line 22: `Short: "Initialize a project for use with devpilot",`
- Line 88: `if err := EnsureGitignore(dir, []string{".devpilot/logs/"}); err != nil {`

`internal/initcmd/generate.go`:
- Line 127: comment `// ConfigureBoard sets up the board name in .devpilot.json.`
- Line 130: `fmt.Println("  Skipped: board configuration (use devpilot init without --yes to configure)")`
- Line 211: `block := "\n# DevPilot\n"`

**Step 5: Update trello commands**

`internal/trello/commands.go`:
- Line 38: `fmt.Fprintln(os.Stderr, "Error: --board is required (or run: devpilot init)")`
- Line 59: `fmt.Fprintln(os.Stderr, "Not logged in to Trello. Run: devpilot login trello")`

**Step 6: Update taskrunner package**

`internal/taskrunner/commands.go`:
- Line 57: `fmt.Fprintln(os.Stderr, "Error: --board is required (or run: devpilot init)")`
- Line 64: `fmt.Fprintln(os.Stderr, "Not logged in to Trello. Run: devpilot login trello")`

`internal/taskrunner/runner.go`:
- Line 258: `prBody := fmt.Sprintf("## Task\n%s\n\n🤖 Executed by devpilot runner", cardURL)`
- Line 293: `r.trello.AddComment(card.ID, fmt.Sprintf("✅ Task completed by devpilot runner\nDuration: %s\nPR: %s", duration, prURL))`
- Line 321: `logPath := filepath.Join(r.config.WorkDir, ".devpilot", "logs", card.ID+".log")`
- Line 332: `logDir := filepath.Join(r.config.WorkDir, ".devpilot", "logs")`

`internal/taskrunner/tui.go`:
- Line 26: comment `// TUIModel is the Bubble Tea model for the devpilot run dashboard.`
- Line 339: `return "  Starting devpilot run..."`

`internal/taskrunner/tui_view.go`:
- Line 73: `left := titleStyle.Render("devpilot run")`

**Step 7: Verify compilation**

Run: `go build ./cmd/devpilot`
Expected: successful build (directory not yet renamed, so old path still valid)

**Step 8: Commit**

```bash
git add -A
git commit -m "chore: rename all devpilot strings to devpilot in Go source"
```

---

### Task 4: Update Test Files

**Files:**
- Modify: `internal/project/config_test.go` (lines 88, 106)
- Modify: `internal/initcmd/generate_test.go` (lines 115-117, 139, 144, 162, 167)
- Modify: `internal/initcmd/detect_test.go` (lines 31, 37, 44)

**Step 1: Update config_test.go**

`internal/project/config_test.go`:
- Line 88: `info, err := os.Stat(filepath.Join(dir, ".devpilot.json"))`
- Line 106: `data, err := os.ReadFile(filepath.Join(dir, ".devpilot.json"))`

**Step 2: Update generate_test.go**

Replace all occurrences of `.devpilot.json` with `.devpilot.json`:
- Line 115: comment `// Should not have created .devpilot.json`
- Line 116: `if _, err := os.Stat(filepath.Join(dir, ".devpilot.json")); !os.IsNotExist(err) {`
- Line 117: `t.Error(".devpilot.json should not exist in non-interactive mode")`
- Line 139: `data, err := os.ReadFile(filepath.Join(dir, ".devpilot.json"))`
- Line 144: `t.Errorf(".devpilot.json does not contain board name, got: %s", string(data))`
- Line 162: `data, err := os.ReadFile(filepath.Join(dir, ".devpilot.json"))`
- Line 167: `t.Errorf(".devpilot.json does not contain board name, got: %s", string(data))`

**Step 3: Update detect_test.go**

Replace comments:
- Line 31: `// Without .devpilot.json`
- Line 37: `// With .devpilot.json but no board`
- Line 44: `// With .devpilot.json and board set`

**Step 4: Run tests**

Run: `go test ./...`
Expected: all tests pass

**Step 5: Commit**

```bash
git add -A
git commit -m "chore: rename devpilot to devpilot in test files"
```

---

### Task 5: Rename Directory and Config File

**Step 1: Rename cmd/devpilot/ to cmd/devpilot/**

```bash
mv cmd/devpilot cmd/devpilot
```

**Step 2: Rename .devpilot.json to .devpilot.json and update content**

```bash
mv .devpilot.json .devpilot.json
```

Update the content of `.devpilot.json` — the `board` value "devpilot" should become "devpilot":
```json
{
  "board": "devpilot"
}
```

**Step 3: Update Makefile**

`Makefile` line 1-2:
```makefile
BINARY := bin/devpilot
PKG := ./cmd/devpilot
```

**Step 4: Update .gitignore**

Replace:
```
# Devpilot
.devpilot/logs/
```
with:
```
# DevPilot
.devpilot/logs/
```

**Step 5: Verify build with new paths**

Run: `go build -o bin/devpilot ./cmd/devpilot`
Expected: successful build

Run: `go test ./...`
Expected: all tests pass

**Step 6: Commit**

```bash
git add -A
git commit -m "chore: rename cmd/devpilot to cmd/devpilot, .devpilot.json to .devpilot.json"
```

---

### Task 6: Update CI/CD and Installer

**Files:**
- Modify: `.github/workflows/release.yml`
- Modify: `install.sh`

**Step 1: Update release.yml**

Replace all `devpilot` with `devpilot` in `.github/workflows/release.yml`:
- Line 24: `GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o devpilot-darwin-arm64 ./cmd/devpilot`
- Line 25: `GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o devpilot-darwin-amd64 ./cmd/devpilot`
- Line 26: `GOOS=linux  GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o devpilot-linux-amd64  ./cmd/devpilot`
- Line 29: `run: sha256sum devpilot-* > checksums.txt`
- Line 34: `run: gh release upload ${{ github.event.release.tag_name }} devpilot-* checksums.txt`

**Step 2: Update install.sh**

Replace all occurrences in `install.sh`:
- Line 4: `# devpilot installer`
- Line 6: `#   curl -sSL https://raw.githubusercontent.com/siyuqian/devpilot/main/install.sh | sh`
- Line 9: `REPO="siyuqian/devpilot"`
- Line 45: `BINARY="devpilot-${OS}-${ARCH}"`
- Line 65: `echo "Installing devpilot ${VERSION} (${OS}/${ARCH})..."`
- Line 105: `mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/devpilot"`
- Line 106: `chmod +x "${INSTALL_DIR}/devpilot"`
- Line 109: `sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/devpilot"`
- Line 110: `sudo chmod +x "${INSTALL_DIR}/devpilot"`
- Line 114: `echo "devpilot ${VERSION} installed to ${INSTALL_DIR}/devpilot"`
- Line 117: `echo "  devpilot --version"`

**Step 3: Commit**

```bash
git add .github/workflows/release.yml install.sh
git commit -m "chore: rename devpilot to devpilot in CI/CD and installer"
```

---

### Task 7: Update Skill Files

**Files:**
- Modify: `.claude/skills/task-executor/SKILL.md`
- Modify: `.claude/skills/task-refiner/SKILL.md`
- Modify: `.claude/skills/trello/SKILL.md`
- Modify: `.claude/skills/pm/SKILL.md`

**Step 1: Update all 4 SKILL.md files**

For each skill file, apply these replacements:
- `devpilot:` → `devpilot:` (in `name:` frontmatter)
- `devpilot runner` → `devpilot runner` (in descriptions and body)
- `devpilot run` → `devpilot run` (in body text)
- `devpilot login trello` → `devpilot login trello` (in body text)
- `devpilot init` → `devpilot init` (in body text)
- `~/.config/devpilot/` → `~/.config/devpilot/` (in file paths)

Specific changes per file:

**task-executor/SKILL.md:**
- `name: devpilot:task-executor`
- `description: Executes a task plan autonomously. Used by the devpilot runner...`
- Body: `devpilot run` (one occurrence)

**task-refiner/SKILL.md:**
- `name: devpilot:task-refiner`
- Description: `devpilot runner` (one occurrence)
- Body: all `~/.config/devpilot/` → `~/.config/devpilot/`
- Body: all `devpilot login trello` → `devpilot login trello`
- Body: all `devpilot run` → `devpilot run`

**trello/SKILL.md:**
- `name: devpilot:trello`
- Description: update if contains "devpilot"
- Body: `devpilot login trello` and `devpilot CLI`
- Body: `~/.config/devpilot/` → `~/.config/devpilot/`

**pm/SKILL.md:**
- `name: devpilot:pm`

**Step 2: Also check for references/quality-checklist.md in task-refiner**

Check if `references/quality-checklist.md` in the task-refiner skill contains any "devpilot" references. If so, update those too.

**Step 3: Commit**

```bash
git add .claude/skills/
git commit -m "chore: rename developerkit skill namespace to devpilot"
```

---

### Task 8: Update README.md and CLAUDE.md

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Step 1: Update README.md**

Apply these replacements (in order, longest first):
1. `github.com/siyuqian/devpilot` → `github.com/siyuqian/devpilot`
2. `siyuqian/devpilot` → `siyuqian/devpilot`
3. `DevPilot` → `DevPilot`
4. `devpilot` → `devpilot`
5. `devpilot:` → `devpilot:`
6. `DevPilot` → `DevPilot`
7. `.devpilot.json` → `.devpilot.json`
8. `.devpilot/` → `.devpilot/`
9. `devpilot` → `devpilot` (all remaining)

**Step 2: Update CLAUDE.md**

Apply the same replacement rules to CLAUDE.md.

**Step 3: Review both files for natural language coherence**

Read through both files after replacement to ensure sentences still read naturally (e.g., "This is a **DevPilot** — a Go CLI tool..." should be "**DevPilot** is a Go CLI tool...").

**Step 4: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: rename devpilot to DevPilot in README and CLAUDE.md"
```

---

### Task 9: Update Historical Documentation

**Files (docs/plans/):**
- `2026-02-25-agent-team-blueprints-design.md`
- `2026-02-25-agent-team-blueprints-plan.md`
- `2026-02-25-pm-skill-design.md`
- `2026-02-27-devpilot-cli-design.md`
- `2026-02-27-devpilot-cli-plan.md`
- `2026-02-27-flatten-directory-design.md`
- `2026-02-27-flatten-directory-plan.md`
- `2026-02-27-mcp-cleanup-design.md`
- `2026-02-27-mcp-cleanup-plan.md`
- `2026-02-28-devpilot-push-design.md`
- `2026-02-28-devpilot-push-plan.md`
- `2026-02-28-domain-based-refactoring-design.md`
- `2026-02-28-domain-refactoring-plan.md`
- `2026-02-28-rejected-ideas-tracking-design.md`
- `2026-02-28-rejected-ideas-tracking-plan.md`
- `2026-02-28-task-refiner-design.md`
- `2026-02-28-task-refiner-plan.md`
- `2026-02-28-task-runner-design.md`
- `2026-02-28-task-runner-plan.md`
- `2026-03-01-github-release-workflow-design.md`
- `2026-03-01-pm-skill-caching-design.md`
- `2026-03-01-rename-devpilot-design.md`
- `2026-03-01-stream-json-dashboard-design.md`
- `2026-03-01-stream-json-dashboard-plan.md`
- `2026-03-01-task-retry-error-recovery-plan.md`
- `2026-03-01-test-verification-gate-plan.md`

**Files (docs/rejected/):**
- `2026-03-01-completion-notifications.md`
- `2026-03-01-cost-tracking-budget-controls.md`
- `2026-03-01-github-issues-task-source.md`
- `2026-03-01-mcp-server-for-devpilot.md`
- `2026-03-01-task-analytics-history.md`
- `2026-03-01-task-dependencies.md`
- `README.md`

**Step 1: Bulk replace in all docs/plans/ and docs/rejected/ files**

Apply the same replacement rules (in order, longest first):
1. `github.com/siyuqian/devpilot` → `github.com/siyuqian/devpilot`
2. `siyuqian/devpilot` → `siyuqian/devpilot`
3. `DevPilot` → `DevPilot`
4. `devpilot` → `devpilot`
5. `devpilot:` → `devpilot:`
6. `DevPilot` → `DevPilot`
7. `.devpilot.json` → `.devpilot.json`
8. `.devpilot/` → `.devpilot/`
9. `devpilot` → `devpilot` (all remaining, case-sensitive)
10. `Devpilot` → `Devpilot` (if any)
11. `DevPilot` → `DevPilot` (if any)

**Step 2: Rename file with "devpilot" in name**

```bash
mv docs/rejected/2026-03-01-mcp-server-for-devpilot.md docs/rejected/2026-03-01-mcp-server-for-devpilot.md
```

Also consider renaming docs/plans files that have "devpilot" in the filename:
```bash
mv docs/plans/2026-02-27-devpilot-cli-design.md docs/plans/2026-02-27-devpilot-cli-design.md
mv docs/plans/2026-02-27-devpilot-cli-plan.md docs/plans/2026-02-27-devpilot-cli-plan.md
mv docs/plans/2026-02-28-devpilot-push-design.md docs/plans/2026-02-28-devpilot-push-design.md
mv docs/plans/2026-02-28-devpilot-push-plan.md docs/plans/2026-02-28-devpilot-push-plan.md
```

**Step 3: Commit**

```bash
git add docs/
git commit -m "docs: rename devpilot to devpilot in all historical documentation"
```

---

### Task 10: Final Verification

**Step 1: Full build**

Run: `make build`
Expected: Binary built at `bin/devpilot`

**Step 2: Full test suite**

Run: `make test`
Expected: all tests pass

**Step 3: Check binary works**

Run: `./bin/devpilot --version`
Expected: version output

Run: `./bin/devpilot --help`
Expected: help text with "devpilot" branding

**Step 4: Zero residual check**

Run: `grep -ri "devpilot" --include="*.go" --include="*.md" --include="*.yml" --include="*.sh" --include="*.json" --include="Makefile" .`
Expected: zero matches (no residual "devpilot" anywhere)

Also check for `devpilot`:
Run: `grep -ri "devpilot" --include="*.go" --include="*.md" --include="*.yml" --include="*.sh" --include="*.json" .`
Expected: zero matches

**Step 5: Check for any missed files**

Run: `grep -ri "developerkit" --include="*.md" .`
Expected: zero matches

**Step 6: If any residuals found, fix them and commit**

```bash
git add -A
git commit -m "chore: fix remaining devpilot references"
```
