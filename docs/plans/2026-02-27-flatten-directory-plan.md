# Flatten Directory Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restructure the repository from a nested `cli/` subdirectory to the Go standard project layout with `cmd/`, `internal/` at the root.

**Architecture:** Move all Go source from `cli/` to root. Entry point at `cmd/devkit/main.go` imports `internal/cli` for Cobra commands. Module path changes from `github.com/siyuqian/developer-kit/cli` to `github.com/siyuqian/developer-kit`.

**Tech Stack:** Go 1.25.6, Cobra

---

### Task 1: Create root go.mod and go.sum

**Files:**
- Create: `go.mod`
- Create: `go.sum`

**Step 1: Create go.mod with updated module path**

Write `go.mod`:

```go
module github.com/siyuqian/developer-kit

go 1.25.6

require github.com/spf13/cobra v1.10.2

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)
```

**Step 2: Copy go.sum**

Copy `cli/go.sum` to `go.sum` (contents unchanged).

**Step 3: Verify module is valid**

Run: `go mod verify`
Expected: `all modules verified`

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add root go.mod with updated module path"
```

---

### Task 2: Create internal/config/ (no import changes)

**Files:**
- Create: `internal/config/credentials.go` (copy from `cli/internal/config/credentials.go`, unchanged)
- Create: `internal/config/credentials_test.go` (copy from `cli/internal/config/credentials_test.go`, unchanged)

**Step 1: Copy config files**

Copy `cli/internal/config/credentials.go` to `internal/config/credentials.go`. No changes needed — this package has no internal imports.

Copy `cli/internal/config/credentials_test.go` to `internal/config/credentials_test.go`. No changes needed.

**Step 2: Run config tests**

Run: `go test ./internal/config/ -v`
Expected: All 4 tests pass (TestSaveAndLoad, TestLoadMissing, TestRemove, TestListServices)

**Step 3: Commit**

```bash
git add internal/config/
git commit -m "chore: move config package to internal/config"
```

---

### Task 3: Create internal/services/ (update config import)

**Files:**
- Create: `internal/services/service.go` (copy unchanged)
- Create: `internal/services/registry.go` (copy unchanged)
- Create: `internal/services/trello.go` (update import)
- Create: `internal/services/trello_test.go` (copy unchanged)

**Step 1: Copy service.go and registry.go**

Copy `cli/internal/services/service.go` to `internal/services/service.go`. No changes needed.

Copy `cli/internal/services/registry.go` to `internal/services/registry.go`. No changes needed.

**Step 2: Copy trello.go with updated import**

Copy `cli/internal/services/trello.go` to `internal/services/trello.go`. Change the import:

```go
// OLD:
"github.com/siyuqian/developer-kit/cli/internal/config"

// NEW:
"github.com/siyuqian/developer-kit/internal/config"
```

**Step 3: Copy trello_test.go**

Copy `cli/internal/services/trello_test.go` to `internal/services/trello_test.go`. No changes needed.

**Step 4: Run services tests**

Run: `go test ./internal/services/ -v`
Expected: All 2 tests pass (TestTrelloVerify_Success, TestTrelloVerify_InvalidCreds)

**Step 5: Commit**

```bash
git add internal/services/
git commit -m "chore: move services package to internal/services"
```

---

### Task 4: Create internal/cli/ (Cobra commands with updated imports)

**Files:**
- Create: `internal/cli/root.go`
- Create: `internal/cli/login.go`
- Create: `internal/cli/logout.go`
- Create: `internal/cli/status.go`

**Step 1: Create root.go**

Copy `cli/cmd/root.go` to `internal/cli/root.go`. Change the package declaration:

```go
// OLD:
package cmd

// NEW:
package cli
```

No import changes needed — root.go only imports cobra and stdlib.

**Step 2: Create login.go**

Copy `cli/cmd/login.go` to `internal/cli/login.go`. Two changes:

```go
// OLD:
package cmd

// NEW:
package cli
```

```go
// OLD:
"github.com/siyuqian/developer-kit/cli/internal/services"

// NEW:
"github.com/siyuqian/developer-kit/internal/services"
```

**Step 3: Create logout.go**

Copy `cli/cmd/logout.go` to `internal/cli/logout.go`. Same two changes as login.go:

- `package cmd` -> `package cli`
- `".../cli/internal/services"` -> `".../developer-kit/internal/services"`

**Step 4: Create status.go**

Copy `cli/cmd/status.go` to `internal/cli/status.go`. Three changes:

- `package cmd` -> `package cli`
- `".../cli/internal/config"` -> `".../developer-kit/internal/config"`
- `".../cli/internal/services"` -> `".../developer-kit/internal/services"`

**Step 5: Verify compilation**

Run: `go build ./internal/cli/`
Expected: Builds without errors

**Step 6: Commit**

```bash
git add internal/cli/
git commit -m "chore: move Cobra commands to internal/cli"
```

---

### Task 5: Create cmd/devkit/main.go entry point

**Files:**
- Create: `cmd/devkit/main.go`

**Step 1: Create main.go**

Write `cmd/devkit/main.go`:

```go
package main

import "github.com/siyuqian/developer-kit/internal/cli"

func main() {
	cli.Execute()
}
```

**Step 2: Build the binary**

Run: `go build -o devkit ./cmd/devkit`
Expected: Produces `devkit` binary at repo root

**Step 3: Verify binary works**

Run: `./devkit --help`
Expected: Shows help text with "Developer toolkit for managing service integrations"

**Step 4: Run all tests**

Run: `go test ./... -v`
Expected: All 6 tests pass across both packages

**Step 5: Commit**

```bash
git add cmd/devkit/main.go
git commit -m "chore: add cmd/devkit entry point"
```

---

### Task 6: Remove cli/ directory and clean up

**Files:**
- Delete: `cli/` (entire directory)
- Modify: `CLAUDE.md`
- Create: `.gitignore` (ignore compiled binary)

**Step 1: Delete the cli/ directory**

```bash
rm -rf cli/
```

**Step 2: Add .gitignore**

Create `.gitignore` at repo root:

```
# Compiled binary
devkit
```

**Step 3: Clean up devkit binary if present**

```bash
rm -f devkit
```

**Step 4: Update CLAUDE.md**

Replace the build commands and architecture sections to reflect the new layout:

```markdown
## Repository Structure

- `cmd/devkit/` — CLI entry point
- `internal/cli/` — Cobra command definitions (login, logout, status)
- `internal/config/` — Credential storage (~/.config/devkit/credentials.json)
- `internal/services/` — Service interface + implementations (Trello)
- `.claude/skills/` — Built-in development skills
  - `skill-creator/` — Guide + scripts for creating new skills
  - `mcp-builder/` — Guide + scripts for building MCP servers
- `docs/plans/` — Design and planning documents

## Build & Development Commands

### Devkit CLI

```bash
go build -o devkit ./cmd/devkit   # Build binary
go test ./...                      # Run all tests
./devkit --help                    # Show help
./devkit login trello              # Login to Trello
./devkit status                    # Check auth status
```

## Architecture

### Devkit CLI

Go CLI tool using Cobra for subcommand routing:
- `cmd/devkit/` — Entry point (`main.go` calls `cli.Execute()`)
- `internal/cli/` — Cobra commands (root, login, logout, status)
- `internal/config/` — Credential storage (~/.config/devkit/credentials.json)
- `internal/services/` — Service interface + implementations (Trello)
- Adding a new service: implement the Service interface in a new file, register in registry.go
```

**Step 5: Verify everything still works**

Run: `go test ./... -v`
Expected: All 6 tests pass

Run: `go build -o devkit ./cmd/devkit && ./devkit --help`
Expected: Builds and shows help

**Step 6: Clean up binary and commit**

```bash
rm -f devkit
git add -A
git commit -m "chore: flatten cli/ to Go standard project layout

Move Go code from cli/ subdirectory to repo root following
golang-standards/project-layout conventions:
- cmd/devkit/ for entry point
- internal/cli/ for Cobra commands
- internal/config/ for credential storage
- internal/services/ for service implementations

Module path: github.com/siyuqian/developer-kit"
```
