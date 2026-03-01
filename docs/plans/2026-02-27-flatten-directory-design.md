# Flatten Directory to Go Standard Layout

**Date:** 2026-02-27
**Status:** Approved

## Goal

Restructure the repository from a nested `cli/` subdirectory to follow the [golang-standards/project-layout](https://github.com/golang-standards/project-layout) convention, with Go code at the project root.

## Current Structure

```
devpilot/
├── cli/
│   ├── main.go
│   ├── go.mod          # module github.com/siyuqian/devpilot/cli
│   ├── go.sum
│   ├── devpilot           # compiled binary
│   ├── cmd/
│   │   ├── root.go
│   │   ├── login.go
│   │   ├── logout.go
│   │   └── status.go
│   └── internal/
│       ├── config/
│       │   ├── credentials.go
│       │   └── credentials_test.go
│       └── services/
│           ├── service.go
│           ├── registry.go
│           ├── trello.go
│           └── trello_test.go
├── docs/plans/
├── .claude/skills/
├── CLAUDE.md
└── .mcp.json
```

## Target Structure

```
devpilot/
├── go.mod              # module github.com/siyuqian/devpilot
├── go.sum
├── cmd/
│   └── devpilot/
│       └── main.go     # entry point, calls cli.Execute()
├── internal/
│   ├── cli/
│   │   ├── root.go     # root Cobra command + Execute()
│   │   ├── login.go
│   │   ├── logout.go
│   │   └── status.go
│   ├── config/
│   │   ├── credentials.go
│   │   └── credentials_test.go
│   └── services/
│       ├── service.go
│       ├── registry.go
│       ├── trello.go
│       └── trello_test.go
├── docs/plans/
├── .claude/skills/     # unchanged
├── CLAUDE.md           # updated paths/commands
└── .mcp.json
```

## Changes

### File Moves

| From | To |
|------|-----|
| `cli/go.mod` | `go.mod` |
| `cli/go.sum` | `go.sum` |
| `cli/main.go` | `cmd/devpilot/main.go` |
| `cli/cmd/root.go` | `internal/cli/root.go` |
| `cli/cmd/login.go` | `internal/cli/login.go` |
| `cli/cmd/logout.go` | `internal/cli/logout.go` |
| `cli/cmd/status.go` | `internal/cli/status.go` |
| `cli/internal/config/*` | `internal/config/*` |
| `cli/internal/services/*` | `internal/services/*` |

### Deleted

- `cli/devpilot` (compiled binary)
- `cli/` directory (empty after moves)

### Module Path Update

- **Old:** `github.com/siyuqian/devpilot/cli`
- **New:** `github.com/siyuqian/devpilot`

### Package Rename

- `cli/cmd/` (package `cmd`) -> `internal/cli/` (package `cli`)

### Import Path Updates

- `main.go`: `import ".../cli/cmd"` -> `import ".../devpilot/internal/cli"`
- `cmd/*.go` -> `internal/cli/*.go`: `import ".../cli/internal/config"` -> `import ".../devpilot/internal/config"`
- `cmd/*.go` -> `internal/cli/*.go`: `import ".../cli/internal/services"` -> `import ".../devpilot/internal/services"`

### Build Commands

```bash
# From repo root:
go build -o devpilot ./cmd/devpilot
go test ./...
```

### CLAUDE.md Updates

Update build commands and architecture section to reflect new paths.

## Decisions

- **Binary name:** stays `devpilot`
- **`.claude/skills/`:** unchanged (Claude Code workspace files, not part of Go project)
- **Cobra commands:** in `internal/cli/` (thin entry point in `cmd/devpilot/`)
- **No empty scaffolding:** only directories that serve the project's needs
