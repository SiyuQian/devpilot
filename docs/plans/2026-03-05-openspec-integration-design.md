# OpenSpec Integration Design

**Date:** 2026-03-05
**Status:** Implemented

## Problem Statement

DevPilot's current workflow treats plan files as one-shot artifacts: `devpilot push` sends a plan to Trello/GitHub Issues, the runner executes it, and the plan is effectively discarded. There is no living specification, no change tracking, and no ability to resume from failure. As AI-automated code submission and review become the norm, a spec-driven development (SDD) approach is needed to ensure quality and traceability.

## Target Workflow

```
PM writes requirements (Confluence / Notion)
  -> Dev reads and understands requirements
    -> Dev uses OpenSpec to plan changes locally (opsx:propose)
      -> Dev runs `devpilot sync` to push changes to Board/Issues
        -> PM reviews on Board (drag to Backlog if not ready)
          -> During downtime, `devpilot run` executes Ready cards
```

**Key principles:**
- `openspec/changes/` is the source of truth for what will be built
- Trello/GitHub Issues is a read-only visualization layer for PM review
- Sync is one-way: OpenSpec -> Board (direction A)
- PM approval is implicit: cards land in Ready, PM drags to Backlog if not approved
- Dev manually triggers `devpilot sync` when ready to share

## Design

### 1. New Command: `devpilot sync`

Replaces `devpilot push`. Scans `openspec/changes/` and syncs each change to the configured task source (Trello board or GitHub Issues).

**Mapping rules:**
- Card/issue title = change directory name (e.g., `add-user-auth`)
- Card/issue description = full content of `proposal.md` + `tasks.md`
- Default list/state = "Ready"

**Idempotency:**
- If a card/issue with the same title already exists, update its description
- Do not create duplicates

**CLI interface:**
```bash
devpilot sync                          # Sync all changes to configured board
devpilot sync --board "Board Name"     # Override board
devpilot sync --source github          # Override source
```

**Package location:** `internal/openspec/`

### 2. Runner Execution Changes

When the runner picks up a card from the Ready list:

**Current behavior:**
```bash
claude -p "Execute this plan: <card description>"
```

**New behavior:**
```bash
claude -p "/opsx:apply <card-title>"
```

The card title is the OpenSpec change name, so the runner passes it directly to `opsx:apply`. This enables:

- **Resumability:** If a task times out or fails, re-running `opsx:apply` picks up from the last unchecked task in `tasks.md`
- **Spec context:** OpenSpec automatically provides the full spec context to Claude, not just the plan text

### 3. Post-Completion Archival

After a card is marked Done:

1. Runner calls `openspec archive <change-name>` (CLI) to move the change from `openspec/changes/` to `openspec/archive/` and update specs
2. This keeps the changes directory clean and specs up-to-date

### 4. `devpilot push` Deprecation

- `devpilot push` continues to work but prints a deprecation warning
- Warning message directs users to use OpenSpec + `devpilot sync`
- Remove in a future major version

### 5. OpenSpec as Required Dependency

**Detection and installation:**
- `devpilot init` checks for Node.js and `@fission-ai/openspec`
- If missing, prompts user to install: `npm install -g @fission-ai/openspec@latest`
- `devpilot sync` and `devpilot run` validate OpenSpec is installed at startup

**Version management:**
- `.devpilot.json` gains `openspecMinVersion` field
- Runner checks installed version against minimum; prompts upgrade if outdated

### 6. Configuration Changes

`.devpilot.json` updated schema:

```json
{
  "board": "devpilot",
  "source": "github",
  "openspecMinVersion": "1.2.0"
}
```

### 7. New Package Structure

```
internal/
  openspec/
    openspec.go      # OpenSpec CLI wrapper (scan changes, validate version, archive)
    commands.go      # Cobra command: devpilot sync
    sync.go          # Sync logic: read changes/, create/update cards/issues
    sync_test.go     # Tests for sync logic
```

## What Does NOT Change

- **TaskSource interface** — Trello and GitHub Issues implementations remain the same
- **Runner state machine** — Ready -> In Progress -> Done / Failed
- **TUI dashboard** — No changes needed
- **Code review flow** — Still runs after execution if configured
- **Git branch/PR flow** — Still creates `task/{id}-{slug}` branches and PRs
- **Auth system** — Trello/GitHub credentials unchanged

## Trade-offs

**Adds Node.js as a runtime dependency:**
OpenSpec is an npm package. This means devpilot users need Node.js installed. This is acceptable because: (a) most dev environments already have Node.js, (b) the alternative (reimplementing OpenSpec in Go) would be a massive effort with ongoing maintenance burden, (c) OpenSpec's ecosystem and CLI tools provide immediate value.

**One-way sync limits PM input:**
PM can only drag cards to Backlog, not edit specs. This is intentional — specs are developer-owned artifacts. PM feedback flows through Confluence/Notion requirements, not through board card edits.

**Runner depends on local OpenSpec state:**
The runner reads `openspec/changes/` from the local repo. If the repo is not up-to-date, the runner may not find the change. This is mitigated by the fact that `devpilot run` is typically run on the same machine where `devpilot sync` was executed.
