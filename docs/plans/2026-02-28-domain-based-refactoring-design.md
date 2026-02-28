# Domain-Based Package Refactoring Design

## Context

The `internal/` directory is currently organized by technical layer (`cli/`, `config/`, `services/`, `trello/`, `runner/`). This design reorganizes it by business capability into three domain packages: `auth/`, `trello/`, `taskrunner/`.

## Target Structure

```
internal/
├── auth/                    # Authentication & credential management
│   ├── credentials.go       # Save/Load/Remove/ListServices
│   ├── credentials_test.go
│   ├── service.go           # Service interface + Registry
│   ├── trello.go            # TrelloService implementation
│   ├── trello_test.go
│   └── commands.go          # login, logout, status CLI commands
│
├── trello/                  # Trello API client + push command
│   ├── client.go            # HTTP client
│   ├── client_test.go
│   ├── types.go             # Board, List, Card
│   ├── commands.go          # push CLI command
│   └── commands_test.go
│
└── taskrunner/              # Task execution orchestration
    ├── executor.go          # Subprocess execution
    ├── executor_test.go
    ├── git.go               # Git/GitHub operations
    ├── git_test.go
    ├── reviewer.go          # Automated code review
    ├── reviewer_test.go
    ├── runner.go            # Main polling loop
    └── commands.go          # run CLI command
```

## Migration Map

| Current File | New Location | Notes |
|---|---|---|
| `cli/root.go` | `cmd/devkit/main.go` | Root command inlined into main |
| `cli/login.go` | `auth/commands.go` | |
| `cli/logout.go` | `auth/commands.go` | |
| `cli/status.go` | `auth/commands.go` | |
| `cli/run.go` | `taskrunner/commands.go` | |
| `cli/push.go` | `trello/commands.go` | |
| `cli/push_test.go` | `trello/commands_test.go` | |
| `config/credentials.go` | `auth/credentials.go` | Package: `config` → `auth` |
| `config/credentials_test.go` | `auth/credentials_test.go` | |
| `services/service.go` | `auth/service.go` | |
| `services/registry.go` | `auth/service.go` | Merged with service.go |
| `services/trello.go` | `auth/trello.go` | |
| `services/trello_test.go` | `auth/trello_test.go` | |
| `runner/executor.go` | `taskrunner/executor.go` | Package: `runner` → `taskrunner` |
| `runner/executor_test.go` | `taskrunner/executor_test.go` | |
| `runner/git.go` | `taskrunner/git.go` | |
| `runner/git_test.go` | `taskrunner/git_test.go` | |
| `runner/reviewer.go` | `taskrunner/reviewer.go` | |
| `runner/reviewer_test.go` | `taskrunner/reviewer_test.go` | |
| `runner/runner.go` | `taskrunner/runner.go` | |

## Wiring Pattern

Each domain exports `RegisterCommands(parent *cobra.Command)`. The entry point becomes:

```go
// cmd/devkit/main.go
func main() {
    rootCmd := &cobra.Command{Use: "devkit", Short: "Developer kit CLI"}
    auth.RegisterCommands(rootCmd)
    trello.RegisterCommands(rootCmd)
    taskrunner.RegisterCommands(rootCmd)
    rootCmd.Execute()
}
```

## Dependency Flow

```
cmd/devkit/main.go
├── auth       (no internal deps)
├── trello     (depends on: auth)
└── taskrunner (depends on: trello, auth)
```

No circular dependencies.

## What Stays the Same

- All external CLI behavior (commands, flags, output)
- All tests pass with same coverage
- `trello/client.go` and `trello/types.go` content unchanged
- Credential storage path (`~/.config/devkit/credentials.json`)

## Deleted Packages

- `internal/cli/` — commands distributed to domain packages
- `internal/config/` — absorbed into `auth/`
- `internal/services/` — absorbed into `auth/`
- `internal/runner/` — renamed to `taskrunner/`
