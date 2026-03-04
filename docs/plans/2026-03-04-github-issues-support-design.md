# GitHub Issues Support Design

Date: 2026-03-04

## Overview

Add GitHub Issues as an alternative task source alongside Trello. Users can choose either backend to manage features and tasks for the devpilot autonomous runner.

## State Mapping

GitHub Issues maps to the Trello state machine as follows:

| State | Issue Status | Labels |
|-------|-------------|--------|
| Ready | open | `devpilot` only |
| In Progress | open | `devpilot` + `in-progress` |
| Done | closed | — |
| Failed | open | `devpilot` + `failed` |

Only issues with the `devpilot` label are managed by the runner. Failed tasks remain open so users can retry by removing the `failed` label.

## Architecture

### TaskSource Interface

New file `internal/taskrunner/source.go` defines the abstraction:

```go
type Task struct {
    ID          string
    Name        string
    Description string
    URL         string
    Priority    int // parsed from P0/P1/P2 labels
}

type TaskSource interface {
    FetchReady() ([]Task, error)
    MarkInProgress(id string) error
    MarkDone(id, prURL, duration string) error
    MarkFailed(id, errMsg string) error
    AddComment(id, comment string) error
}
```

### Runner Refactor

`Runner` replaces `*trello.Client` with `TaskSource`. The core task execution logic (git operations, claude invocation, PR creation) is unchanged. Only the task lifecycle calls (fetch, state transitions, comments) go through the interface.

### Trello Adapter

`internal/trello/source.go` wraps the existing `trello.Client` to implement `TaskSource`. All existing behavior is preserved.

### GitHub Adapter

New `internal/github` package implements `TaskSource` using the `gh` CLI — no additional SDK required. The `gh` CLI automatically infers the repository from the current directory's `origin` remote.

```
FetchReady    → gh issue list --label devpilot --state open --json number,title,body,url,labels
MarkInProgress → gh issue edit <id> --add-label in-progress
MarkFailed    → gh issue edit <id> --remove-label in-progress --add-label failed
MarkDone      → gh issue close <id>
AddComment    → gh issue comment <id> --body "..."
```

## Authentication

No new `devpilot login github` command. The GitHub adapter checks `gh auth status` at startup and returns a friendly error if not authenticated. Users authenticate via the existing `gh auth login` flow.

## Configuration

### `.devpilot.json`

New `source` field:

```json
{
  "board": "My Board",
  "source": "github",
  "models": {}
}
```

Default is `"trello"` for backward compatibility.

### `devpilot run`

New `--source` flag overrides config:

```bash
devpilot run --board "Board Name"              # uses config, defaults to trello
devpilot run --source github                   # use GitHub Issues
devpilot run --source trello --board "..."     # explicit Trello
```

### `devpilot push`

Extends to support GitHub Issues:

```bash
devpilot push <plan.md>                        # auto-detects source from config
devpilot push <plan.md> --source github        # create GitHub Issue
```

Issue title = first `# Heading` line of the plan file. Issue body = full file content. The `devpilot` label is added automatically.

### `devpilot init`

The interactive wizard gains a new step: "Choose task source (trello/github)".

## File Changes

- `internal/taskrunner/source.go` — new: `Task` struct + `TaskSource` interface
- `internal/taskrunner/runner.go` — refactor: use `TaskSource` instead of `*trello.Client`
- `internal/trello/source.go` — new: Trello adapter implementing `TaskSource`
- `internal/github/` — new package: GitHub Issues adapter + `gh` CLI wrapper
- `internal/github/source.go` — `TaskSource` implementation
- `internal/github/client.go` — `gh` CLI wrapper (issue list, edit, comment, close)
- `internal/taskrunner/commands.go` — add `--source` flag, wire up correct source
- `internal/trello/commands.go` — add `--source` flag to push command
- `internal/project/config.go` — add `Source` field to `Config`
- `internal/initcmd/` — add source selection step to wizard
