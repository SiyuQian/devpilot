## 1. Streaming Event Handler

- [x] 1.1 Create `internal/review/streamer.go` with a `reviewStreamer` struct that implements `ClaudeEventHandler` — prints tool calls and thinking indicators to stderr, streams text content to stdout
- [x] 1.2 Add TTY detection: suppress stderr progress when stdout is not a terminal
- [x] 1.3 Add tests for `reviewStreamer` verifying it extracts text from assistant messages and formats tool/thinking indicators correctly

## 2. Wire Up Streaming in Review Command

- [x] 2.1 Update `newReviewExecutor` in `review.go` to accept and wire a `ClaudeEventHandler` via `WithClaudeEventHandler`
- [x] 2.2 Update `commands.go` `Run` function to create a `reviewStreamer`, pass it to `Review()`, and skip the final `fmt.Print(result.Stdout)` since text is already streamed
- [x] 2.3 Ensure `--dry-run` path remains unchanged (no streaming, just print prompt)

## 3. Verification

- [x] 3.1 Run `make test` and `make lint` — all pass
- [x] 3.2 Manual test: `devpilot review <pr-url>` shows real-time progress and readable output
- [x] 3.3 Manual test: `devpilot review <pr-url> > review.md` captures only review text, no progress noise
