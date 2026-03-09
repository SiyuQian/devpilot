## Why

The current email workflow requires a Claude Code skill to orchestrate multiple CLI calls (list, read, classify, send, mark-read) to produce an email summary. This is fragile, slow (N API calls per email), and tightly couples summarization logic to the skill layer. Users should be able to get an email digest with a single CLI command, with optional Slack delivery built in.

## What Changes

- **New `devpilot gmail summary` command**: Fetches today's unread emails, calls `claude -p` to generate an intelligent summary, optionally sends to a Slack channel or DM, and marks processed emails as read.
  - `--channel <name>`: Send summary to a Slack channel
  - `--dm <user-id>`: Send summary as a DM
  - `--no-mark-read`: Skip marking emails as read (preview mode)
  - Uses today's date as the default filter (`after:YYYY/MM/DD`)
  - No arbitrary limit on email count (fetches all unread for today)
- **Remove `email-assistant` skill**: The skill's orchestration logic is replaced by the CLI command. LLMs can discover `gmail summary` from CLAUDE.md.
- **Update CLAUDE.md**: Add `gmail summary` to the CLI commands section.

## Capabilities

### New Capabilities
- `gmail-summary`: CLI command that fetches, summarizes (via claude -p), optionally delivers to Slack, and marks emails as read.

### Modified Capabilities
- `gmail-commands`: The `list` behavior is unchanged, but `summary` is added as a new subcommand that internally constructs its own query with today's date filter.

## Impact

- `internal/gmail/`: New `summary` command + supporting logic for invoking `claude -p`
- `.claude/skills/email-assistant/`: Deleted entirely
- `CLAUDE.md`: Updated CLI commands section
- Runtime dependency: Requires `claude` CLI to be available on PATH for the summary command
