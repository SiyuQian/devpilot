## Why

With Gmail and Slack integrations in place, we need a Claude Code skill that ties them together: read unread emails, summarize them intelligently, send the digest to Slack, and mark emails as read. This is the user-facing workflow that makes the infrastructure useful. Designed to run via `claude -p "/email-assistant"` on a cron schedule (morning + evening).

## What Changes

- Create an `email-assistant` Claude Code skill using the skill-creator skill
- The skill orchestrates: `devpilot gmail list` → `devpilot gmail read` → Claude summarization → `devpilot slack send` → `devpilot gmail mark-read`
- Include cron setup documentation for scheduled daily digests
- Skill handles edge cases: no unread emails, login errors, large volumes

## Capabilities

### New Capabilities
- `email-assistant-skill`: Claude Code skill that reads unread Gmail messages, generates an intelligent summary grouped by priority/topic, sends the digest to a configured Slack channel, and marks processed emails as read. Supports cron-based scheduling.

### Modified Capabilities

## Impact

- `.claude/skills/email-assistant/` — new skill directory with SKILL.md
- Depends on `gmail-integration` (devpilot gmail commands) and `slack-integration` (devpilot slack send)
- No code changes to existing packages — this is purely a skill (prompt + instructions)
- Skill MUST be created using the skill-creator skill
