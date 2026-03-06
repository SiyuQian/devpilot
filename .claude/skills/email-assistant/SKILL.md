---
name: email-assistant
description: Read unread Gmail messages, generate an intelligent summary grouped by priority, send the digest to a Slack channel, and mark processed emails as read. Use when the user wants an email digest, email summary, or to check emails. Triggers on email-assistant, email digest, check emails, summarize emails.
---

# Email Assistant

Orchestrate an email digest workflow: read unread Gmail, summarize intelligently, send to Slack, mark as read.

## Arguments

- `--channel <name>` — Target Slack channel for the digest (required). Example: `/email-assistant --channel daily-digest`

## Prerequisites

Before starting, verify both services are authenticated:

```bash
devpilot status
```

If Gmail or Slack shows as "not logged in":
- Gmail: tell the user to run `devpilot login gmail` and stop
- Slack: tell the user to run `devpilot login slack` and stop

## Workflow

### Step 1: List unread emails

```bash
devpilot gmail list --unread --limit 50
```

- If output says "No unread messages.", send a brief message to Slack: "No new emails to report." and stop
- Parse the table output to extract message IDs — you will need these for reading and marking as read
- Note the total count. If the list returns 50 messages, mention "Additional unread emails may exist beyond the 50 shown" in the digest

### Step 2: Read each email

For each message ID from Step 1:

```bash
devpilot gmail read <message-id>
```

- Collect the From, Subject, Date, and body of each email
- If an email body is longer than 2000 characters, use only the first 2000 characters for summarization

### Step 3: Classify and summarize

Classify each email into one of three categories:

**ACTION REQUIRED** — Emails that need a response, review, decision, or action from the user. Examples: review requests, meeting invites requiring RSVP, approval requests, questions directed at the user, deadlines.

**INFORMATIONAL** — Emails worth knowing about but requiring no action. Examples: status updates, deploy notifications, FYI messages, team announcements, shared documents.

**PROMOTIONAL / NOISE** — Newsletters, marketing emails, automated notifications, social media alerts, subscription digests. Do NOT summarize these individually — just count them.

### Step 4: Format the digest

Build the digest in this format:

```
Email Digest — YYYY-MM-DD Morning/Evening

ACTION REQUIRED (N):
- [Sender Name] Subject — Brief description of what action is needed
- [Sender Name] Subject — Brief description of what action is needed

INFORMATIONAL (N):
- [Sender Name] Subject — One-line summary
- [Sender Name] Subject — One-line summary

N promotional/automated emails skipped.
```

Rules:
- Use "Morning" if current time is before 12pm, "Evening" otherwise
- Each summary line should be one sentence, max ~100 characters
- If a category has zero items, omit that section entirely
- If there are more than 50 unread emails, append: "Note: Only the 50 most recent emails were processed. Additional unread emails remain."

### Step 5: Send to Slack

```bash
devpilot slack send --channel <channel> --message "<digest>"
```

- The `<channel>` comes from the `--channel` argument
- If the channel name starts with `#`, strip it (the CLI handles this)
- **If the send fails, do NOT proceed to Step 6.** Report the error and stop. Emails must remain unread so they appear in the next digest run.

### Step 6: Mark emails as read

Only after a successful Slack send:

```bash
devpilot gmail mark-read <id1> <id2> <id3> ...
```

- Pass ALL message IDs from Step 1 (including promotional emails that were skipped in the summary)
- This ensures processed emails don't appear in the next digest

## Error Handling

| Error | Action |
|-------|--------|
| Gmail not logged in | Print: "Gmail authentication required. Run: devpilot login gmail" and stop |
| Slack not logged in | Print: "Slack authentication required. Run: devpilot login slack" and stop |
| No unread emails | Send "No new emails to report." to Slack and stop |
| Slack send fails | Report the error. Do NOT mark emails as read |
| Gmail API error | Report the error with details and stop |
