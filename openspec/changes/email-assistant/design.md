## Context

DevPilot skills are defined in `.claude/skills/<name>/SKILL.md` with YAML frontmatter and markdown body. The skill-creator skill provides scaffolding and validation tooling. The email-assistant skill will be invoked either manually (`/email-assistant`) or via `claude -p "/email-assistant"` in a cron job.

## Goals / Non-Goals

**Goals:**
- A skill that produces a well-structured email digest in Slack
- Smart summarization: group by sender/topic, flag action-required emails, skip noise
- Works unattended via `claude -p` for cron scheduling
- Handles common edge cases gracefully (no emails, auth errors)

**Non-Goals:**
- Email reply or forwarding capabilities
- Interactive email management
- Custom Slack formatting beyond mrkdwn basics
- Skill configuration UI — cron setup is documented, not automated

## Decisions

### 1. Skill workflow

```
1. Bash("devpilot gmail list --unread --limit 50")
2. For each email: Bash("devpilot gmail read <id>")
3. Claude summarizes all emails into a structured digest
4. Bash("devpilot slack send --channel <channel> --message <digest>")
5. Bash("devpilot gmail mark-read <id1> <id2> ...")
```

**Why**: Linear flow, each step uses a devpilot CLI command. Claude's summarization is the intelligent middle layer.

### 2. Digest format

```
Email Digest — 2024-01-15 Morning

ACTION REQUIRED (2):
- [Alice] Q1 Report Review — Needs your review by Friday. Attached spreadsheet.
- [PM Bot] Sprint Planning — You're assigned 3 new tickets.

INFORMATIONAL (5):
- [HR] Office closure Jan 20 — MLK Day, office closed.
- [DevOps] Deploy v2.3.1 — Successful deploy to production.
- [Newsletter] TechCrunch Daily — AI funding roundup.
...

3 promotional emails skipped.
```

**Why**: Prioritized, scannable, actionable. Claude naturally excels at this classification.

### 3. Skill creation via skill-creator

The skill MUST be created using the `skill-creator` skill, which ensures proper SKILL.md structure, validation, and best practices.

### 4. Cron scheduling

Document two crontab entries:
```
0 8 * * * claude -p "/email-assistant --channel daily-digest"
0 18 * * * claude -p "/email-assistant --channel daily-digest"
```

The skill accepts a `--channel` argument (or uses a default from skill config).

### 5. Volume handling

For large volumes (>50 unread), read only the first 50 and note "X more unread emails not shown." Reading all emails would be too slow and expensive in tokens.

## Risks / Trade-offs

- **[Token cost]** → Each email read + summarization costs API tokens. Mitigation: limit to 50 emails per run; skip promotional/notification emails when possible.
- **[Email content length]** → Long email threads can be very large. Mitigation: truncate individual emails to first 2000 chars for summarization.
- **[Cron requires machine awake]** → macOS sleep prevents cron. Mitigation: document `caffeinate` or `pmset` workaround; suggest launchd as alternative.
