# Stream-JSON Dashboard Design

**Date:** 2026-03-01
**Goal:** Show real-time Claude Code progress in the `devpilot run` TUI dashboard by parsing `--output-format stream-json` output.

## Background

The current TUI streams raw `claude -p` stdout/stderr line-by-line into a scrollable log viewport. This shows unstructured text with `[stdout]`/`[stderr]` prefixes — no visibility into what Claude is doing (which tools it's calling, what files it's touching, how many tokens it's using).

Claude Code's `--output-format stream-json` flag emits NDJSON with structured events for every tool call, tool result, text block, and session summary. We can parse this to build a rich multi-panel dashboard.

## Design

### 1. Executor — stream-json Parsing

Switch executor default args from `["-p", "--allowedTools=*"]` to `["-p", "--output-format", "stream-json", "--allowedTools=*"]`.

Add a `StreamParser` that reads NDJSON lines and produces typed Go structs:

```go
// Parsed from stream-json NDJSON lines
type ClaudeSystemMsg struct {
    SessionID string
    Model     string
    Tools     []string
}

type ClaudeAssistantMsg struct {
    Content []ContentBlock // text + tool_use blocks
}

type ClaudeUserMsg struct {
    ToolResults []ToolResult
}

type ClaudeResultMsg struct {
    Turns        int
    DurationMs   int
    InputTokens  int
    OutputTokens int
    CacheReadTokens int
}

// ContentBlock variants
type TextBlock struct {
    Text string
}

type ToolUseBlock struct {
    ID    string
    Name  string
    Input map[string]any
}

type ToolResult struct {
    ToolUseID  string
    Content    string
    DurationMs int
    Truncated  bool
}
```

The `OutputHandler` callback changes from `func(OutputLine)` to `func(ClaudeEvent)` where `ClaudeEvent` is an interface over the message types above.

The executor still accumulates raw stdout for log saving. `ExecuteResult` stays the same.

Non-JSON lines (e.g. from test commands) are treated as raw text output gracefully.

### 2. New Event Types

Replace `CardOutputEvent` with richer events:

```go
ToolStartEvent   { ToolName string, Input map[string]any }
ToolResultEvent  { ToolName string, DurationMs int, Truncated bool }
TextOutputEvent  { Text string }
StatsUpdateEvent { InputTokens, OutputTokens, CacheReadTokens, Turns int }
```

**Mapping from stream-json:**
- `assistant` message with `tool_use` block → `ToolStartEvent`
- `assistant` message with `text` block → `TextOutputEvent`
- `user` message (tool result) → `ToolResultEvent`
- `result` message → `StatsUpdateEvent` (also emitted incrementally from per-turn `usage`)

All existing events (`CardStartedEvent`, `CardDoneEvent`, `CardFailedEvent`, `ReviewStartedEvent`, `ReviewDoneEvent`, `RunnerStartedEvent`, `PollingEvent`, etc.) remain unchanged.

### 3. TUI Model — Structured State

Replace `logLines []string` with structured state:

```go
type TUIModel struct {
    // ... existing fields (boardName, phase, lists, activeCard, history, etc.)

    // Replaces logLines []string
    toolCalls    []toolCallEntry   // history of completed tool calls
    activeCall   *toolCallEntry    // currently executing tool (nil if none)
    textLines    []string          // Claude's prose output (scrollable)
    stats        sessionStats      // running totals

    // File tracking
    filesRead    []string          // unique files Read/Grep'd
    filesEdited  []string          // unique files Edit/Write'd

    // Viewports (replaces single viewport)
    toolViewport viewport.Model    // scrollable tool call history
    textViewport viewport.Model    // scrollable text output

    // Which panel has focus for scrolling
    focusedPane  string            // "tools" or "text"
}

type toolCallEntry struct {
    toolName   string
    summary    string      // e.g. "main.go" for Read, "go test ./..." for Bash
    durationMs int         // -1 while in progress
    timestamp  time.Time
}

type sessionStats struct {
    inputTokens     int
    outputTokens    int
    cacheReadTokens int
    turns           int
}
```

**Event handling:**
- `ToolStartEvent` → set `activeCall`, extract file paths to `filesRead`/`filesEdited` based on tool name
- `ToolResultEvent` → move `activeCall` to `toolCalls` history, update duration
- `TextOutputEvent` → append to `textLines`, update text viewport
- `StatsUpdateEvent` → update `stats` fields

**Key bindings:**
- `Tab` — switch focus between tool pane and text pane
- `g`/`G`/`j`/`k` — scroll the focused pane
- `q`/`Ctrl+C` — quit (unchanged)

### 4. TUI Layout — Multi-Panel Dashboard

```
┌─────────────────────────────────────────────────────────────────────┐
│ devpilot run   Board: devpilot   [▶ running] ↑12k ↓3k  T:7 [q]│
├──────────────────────┬──────────────────────────────────────────────┤
│  Lists               │  ▶ "Add auth middleware"                    │
│  Ready          2    │    Branch: task/abc123-add-auth             │
│  In Progress    1    │    Duration: 2m34s                          │
│  Done           3    │    ⚡ Edit internal/server/middleware.go     │
│  Failed         0    │                                             │
├──────────────────────┴──────────────────────────────────────────────┤
│  Tool Calls                                        Files           │
│  ✓ Read  main.go                          12ms     R main.go      │
│  ✓ Grep  "AuthMiddleware"                 45ms     R server.go    │
│  ✓ Read  internal/server/server.go        8ms      E middleware.go│
│  ✓ Edit  internal/server/middleware.go    102ms     E server.go    │
│  ✓ Bash  go test ./...                  3400ms                    │
│  ⚡ Edit  internal/server/routes.go         ...                    │
│                                                                    │
├────────────────────────────────────────────────────────────────────┤
│  Claude Output                                                     │
│  I'll add authentication middleware to protect the API endpoints.  │
│  First, let me read the existing server setup...                   │
│  Now I'll create the middleware function...                         │
│                                                                    │
├────────────────────────────────────────────────────────────────────┤
│ ✅ "Fix login bug" (1m12s) | ✅ "Update deps" (45s)               │
└────────────────────────────────────────────────────────────────────┘
```

**Panels (top to bottom):**

| Panel | Content | Height |
|-------|---------|--------|
| Header | Title, board, phase, token usage (↑in ↓out), turn count, quit hint | 1 line |
| Status + Active | Left: list card counts. Right: card name, branch, elapsed, current tool (⚡) | ~5 lines |
| Tool Calls + Files | Left: scrollable tool history (✓ done / ⚡ active, name, summary, duration). Right: deduplicated file list (R=read, E=edited) | flex ~40% |
| Claude Output | Scrollable prose from Claude's text blocks | flex ~30% |
| Footer | Last 5 completed tasks with ✅/❌ icons | 1-2 lines |

**Header token display:** `↑12k` = input tokens, `↓3k` = output tokens (cumulative, abbreviated). No cost display — subscription model, not API billing.

Minimum terminal size: **80x20** (up from 60x15).

### 5. Plain Text Mode

When `--no-tui` or non-TTY, print a simplified single-line feed using the same parsed events:

```
[tool] Read main.go (12ms)
[tool] Bash go test ./... (3400ms)
[text] I'll add the middleware...
[stats] ↑12k ↓3k turns:7
```

### 6. Reviewer — Unchanged

The reviewer runs a separate `claude -p` for code review. It stays as-is (plain text, no stream-json) since we don't display its progress in the dashboard.

### 7. Log Saving — Unchanged

Raw stdout (now NDJSON) saved to `~/.config/devpilot/logs/{card-id}.log`. The full stream-json is useful for debugging.
