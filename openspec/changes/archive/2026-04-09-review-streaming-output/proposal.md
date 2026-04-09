## Why

The `devpilot review` command runs Claude in buffered mode — no output appears until the entire review completes (potentially several minutes). Users see a blank terminal and can't tell if the command is working, stuck, or failed.

## What Changes

- Add real-time streaming output to the `devpilot review` command so users see progress as Claude works
- Display a progress indicator (phase/status) while the review runs
- Print the final review result as readable text (not raw stream-json)

## Capabilities

### New Capabilities
- `review-streaming`: Real-time progress display during `devpilot review` execution, including phase indicators and streamed text output

### Modified Capabilities
- `review-command`: The review command switches from buffered to streaming execution mode

## Impact

- `internal/review/review.go` — wire up `WithClaudeEventHandler` or `WithOutputHandler` in `newReviewExecutor`
- `internal/review/commands.go` — add streaming display logic (progress spinner/phases, final output formatting)
- No API or dependency changes; uses existing `executor` streaming infrastructure
