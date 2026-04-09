## Context

The `devpilot review` command currently uses `executor.Executor` in buffered mode — no `OutputHandler` or `ClaudeEventHandler` is set, so `runBuffered` captures all output silently. The executor already supports streaming via `WithOutputHandler` and `WithClaudeEventHandler` (used by the task runner's TUI). The infrastructure is in place; the review command just doesn't use it.

## Goals / Non-Goals

**Goals:**
- Show real-time progress during `devpilot review` so users know the command is working
- Display the final review as readable text (not raw stream-json)

**Non-Goals:**
- Full TUI dashboard (Bubble Tea) for review — plain terminal output is sufficient
- Streaming output for `--dry-run` mode (it already prints the prompt and exits)

## Decisions

### 1. Use `ClaudeEventHandler` for progress display

Parse stream-json events to show meaningful progress (tool calls, thinking indicators, text output) rather than dumping raw JSON lines.

**Why over `OutputHandler`**: Raw line output is stream-json which is unreadable. Event-based parsing lets us show "Reading file X...", "Analyzing diff...", etc.

### 2. Print phases inline to stderr, final review to stdout

Progress indicators (spinner/phase text) go to stderr so they don't pollute the review output. The final structured review text goes to stdout, keeping it pipeable.

**Why**: Users may want to pipe or redirect the review output (`devpilot review <url> > review.md`). Mixing progress with output breaks this.

### 3. Extract review text from `ClaudeAssistantMsg` text blocks

Instead of printing raw `result.Stdout` (which is stream-json), accumulate text content blocks from assistant messages and print those as the final output.

**Why**: The current code prints stream-json to stdout which is not human-readable.

## Risks / Trade-offs

- [Terminal compatibility] Inline progress with `\r` may not render well in all terminals → Use simple line-based output (no cursor manipulation) as fallback when not a TTY
- [Output format change] Users relying on stream-json stdout will see different output → This is the intended fix; `--dry-run` still works for prompt inspection
