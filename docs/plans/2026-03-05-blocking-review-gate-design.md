# Blocking Review Gate Design

**Date**: 2026-03-05
**Status**: Approved

## Problem

The task runner's code review step is non-blocking — the review result is ignored and the PR is always auto-merged. This means AI-generated code with bugs or quality issues gets merged without intervention.

## Solution

Make code review a blocking gate with a self-heal loop. When the review finds issues, Claude automatically attempts to fix them and re-reviews, up to a hardcoded retry limit. The PR only merges after a clean review.

## Design

### Current Flow

```
Execute task → Push → Create PR → Review (non-blocking) → Merge → Done
```

### New Flow

```
Execute task → Push → Create PR → Review Loop → Merge → Done

Review Loop:
  for attempt 0..MaxReviewRetries:
    Review (code-review skill)
    if "No issues found" in output → approved, break
    if retries remaining → Fix attempt → push → continue
    else → fail card
```

### Approval Detection

The code-review skill does not use `gh pr review --approve`. It posts a comment with one of two formats:

- **Clean**: `"No issues found. Checked for bugs and CLAUDE.md compliance."`
- **Issues**: `"Found N issues: ..."`

Detection: check if review `claude -p` stdout contains `"No issues found"`. If yes → approved. Otherwise → needs fix.

### Prompts

**Review prompt**: `"Code review: ${PR_URL}"` — triggers the installed code-review skill.

**Fix prompt**: `"Fix the code review comments on ${PR_URL}. Read the review with gh pr view and address all requested changes."`

### Timeouts

Both review and fix attempts use the existing `--review-timeout` flag (default 10 minutes). No new timeout flags.

### Constants

New file `internal/taskrunner/config.go` to centralize package constants:

```go
const MaxReviewRetries = 5
```

### Events

New events for TUI/plain-text display:

- `FixStartedEvent{PRURL string, Attempt int}`
- `FixDoneEvent{PRURL string, Attempt int, ExitCode int}`

### Files Changed

| File | Change |
|------|--------|
| `internal/taskrunner/config.go` | New file — centralized constants |
| `internal/taskrunner/runner.go` | Replace non-blocking review block with retry loop |
| `internal/taskrunner/reviewer.go` | Change review prompt, add `FixPrompt()`, add `IsApproved()` helper |
| `internal/taskrunner/events.go` | Add `FixStartedEvent`, `FixDoneEvent` |
| `internal/taskrunner/tui.go` | Handle new events |
| `internal/taskrunner/tui_view.go` | Display fix attempt info |
| `internal/taskrunner/commands.go` | Handle new events in plain-text handler |

### Backward Compatibility

- `--review-timeout 0` still disables review entirely (no change)
- No new CLI flags
- When review is enabled, the only behavioral change is that merge is now blocked on review approval

### What This Does NOT Include

- Configurable quality gates (lint, security scan) — future work
- New CLI flags for retry count — hardcoded at 5
- Changes to Config struct
