## Context

DevPilot currently has `devpilot gmail list/read/mark-read` commands and an `email-assistant` Claude Code skill that orchestrates them. The skill does all the heavy lifting: listing, reading N emails individually, classifying, formatting, sending to Slack, and marking as read. This is slow (N+1 API calls) and couples summarization logic to the skill layer.

The goal is to move the entire workflow into a single `devpilot gmail summary` CLI command that uses `claude -p` for AI summarization.

## Goals / Non-Goals

**Goals:**
- Single CLI command to fetch, summarize, deliver, and mark-read today's unread emails
- Use `claude -p` for intelligent summarization (classification + formatting)
- Support Slack channel and DM delivery
- Support preview mode (`--no-mark-read`)
- Remove the now-redundant `email-assistant` skill

**Non-Goals:**
- Custom date range support (always uses "today")
- Multiple output formats (always plain text)
- Sending email replies from the CLI
- Supporting LLM providers other than Claude

## Decisions

### 1. Invoke `claude -p` via exec, not the API

**Decision**: Shell out to `claude -p` rather than calling the Anthropic API directly.

**Why**: DevPilot already requires Claude Code for the task runner. Using `claude -p` avoids adding an API key management system and keeps the dependency footprint small. The user's existing Claude Code auth handles everything.

**Alternative considered**: Direct Anthropic API calls from Go — rejected because it adds API key management, token counting, and model selection complexity for a feature that `claude -p` handles out of the box.

### 2. Fetch all emails concurrently, then batch to claude -p

**Decision**: Use `ListAllMessageIDs` with query `is:unread after:YYYY/MM/DD`, then fetch each message's full content **concurrently** (bounded to 10 goroutines), concatenate into a single prompt, and call `claude -p` once.

**Why**: One LLM call is faster and cheaper than per-email classification. The prompt includes all email headers + truncated bodies. Concurrent fetching keeps latency manageable even with many emails.

**Concurrency**: Use a semaphore (buffered channel, size 10) to limit parallel `GetMessage` calls. This avoids overwhelming the Gmail API while still being much faster than sequential fetching.

**Body truncation**: Each email body is truncated to 1000 characters to keep the prompt manageable. Headers (From, Subject, Date) are always included in full.

### 3. Summary command owns its own query logic

**Decision**: `gmail summary` constructs its own Gmail query (`is:unread after:YYYY/MM/DD`) internally rather than modifying the `list` command's defaults.

**Why**: Keeps the `list` command general-purpose. The `summary` command has a specific use case (today's unread) and should encode that.

### 4. Mark-read gated on summary success only

**Decision**: Mark emails as read if `claude -p` succeeds and produces output, regardless of whether Slack delivery succeeds.

**Why**: The summary has been produced and displayed to stdout — the content is not lost. If Slack fails, the user still sees the summary in the terminal. Re-processing the same emails next time would be confusing.

**Failure behavior**:
- `claude -p` fails or empty output → exit with error, do NOT mark-read
- Slack send fails → print error to stderr, still mark-read (summary was produced)

### 5. Delete email-assistant skill entirely

**Decision**: Remove `.claude/skills/email-assistant/` directory. Update CLAUDE.md with the new `gmail summary` command.

**Why**: The skill's entire purpose was orchestrating the workflow that `gmail summary` now handles. LLMs can discover `gmail summary` from CLAUDE.md. A skill that just says "run this one command" adds no value.

## Risks / Trade-offs

- **[claude CLI not on PATH]** → The command will check for `claude` on PATH before proceeding and print a clear error if missing.
- **[Very large inbox]** → If a user has hundreds of unread emails today, the prompt could be very large. Mitigated by body truncation (1000 chars per email). If total prompt exceeds a reasonable size, we can add a warning.
- **[claude -p output format unpredictable]** → We pass the summary through to stdout/Slack as-is. The prompt should be specific about desired format but we don't parse the output.
