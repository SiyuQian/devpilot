# devkit push — Design Document

**Date:** 2026-02-28
**Status:** Approved

## Summary

A `devkit push` CLI command that reads a superpowers implementation plan markdown file and creates a Trello card from it. The card name is extracted from the file's `# Heading` and the full file contents become the card description. This feeds directly into the `devkit run` workflow, which picks up cards from the "Ready" list.

## CLI Interface

```
devkit push <plan-file> [flags]

Arguments:
  plan-file              Path to a plan markdown file (required)

Flags:
  --board <name>         Trello board name (required)
  --list <name>          Target list name (default: "Ready")
```

### Example

```bash
devkit push docs/plans/2026-02-28-task-runner-plan.md --board "Sprint Board"
devkit push docs/plans/2026-02-28-task-runner-plan.md --board "Sprint Board" --list "Backlog"
```

## Card Mapping

| Card field      | Source                                          |
|-----------------|-------------------------------------------------|
| **name**        | First `# Heading` line from the file            |
| **description** | Full file contents                              |

## Flow

1. Validate args: exactly one positional arg (file path)
2. Read the file, extract the `# Title` from line 1
3. Load Trello credentials from `~/.config/devkit/credentials.json`
4. Resolve board name → ID via `FindBoardByName`
5. Resolve list name → ID via `FindListByName`
6. Call `CreateCard(listID, name, desc)` — new method on Trello client
7. Print the card URL on success

## Code Changes

| File                              | Change                                                        |
|-----------------------------------|---------------------------------------------------------------|
| `internal/trello/client.go`      | Add `CreateCard(listID, name, desc string) (*Card, error)`    |
| `internal/trello/types.go`       | Add `URL` field to `Card` struct (from `shortUrl` JSON)       |
| `internal/trello/client_test.go` | Test for `CreateCard`                                         |
| `internal/cli/push.go`           | New Cobra command                                             |

## Design Decisions

### Reuse existing Trello client

The Trello client already has `FindBoardByName`, `FindListByName`, and the `post` helper. Adding `CreateCard` is a one-method extension — no new packages needed.

### Card name from heading

Plans always start with `# Feature Name Implementation Plan`. Extracting this gives a readable card title automatically, with no extra user input.

### Default list = "Ready"

This matches the `devkit run` convention where the runner polls the "Ready" list. Pushing a plan to "Ready" means it's immediately eligible for autonomous execution.
