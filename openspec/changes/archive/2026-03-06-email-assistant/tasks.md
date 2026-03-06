## 1. Skill Scaffolding

- [x] 1.1 Use the skill-creator skill to create the email-assistant skill at `.claude/skills/email-assistant/`
- [x] 1.2 Define SKILL.md frontmatter: name, description, triggers (email-assistant, email digest, check emails)

## 2. Skill Instructions

- [x] 2.1 Write the skill body: step-by-step instructions for Claude to execute the email digest workflow
- [x] 2.2 Define the digest format template (ACTION REQUIRED / INFORMATIONAL / skipped count)
- [x] 2.3 Add email classification guidance (action required vs informational vs promotional)
- [x] 2.4 Add volume handling instructions (limit 50, truncate long emails to 2000 chars)
- [x] 2.5 Add error handling instructions (no emails, auth errors, Slack send failure)
- [x] 2.6 Add mark-as-read logic: only after successful Slack send

## 3. Skill Arguments

- [x] 3.1 Define `--channel` argument for target Slack channel
- [x] 3.2 Document default channel behavior (require explicit channel or use a configured default)

## 4. Cron Documentation

- [x] 4.1 Add cron setup instructions in skill references (crontab entries for morning + evening)
- [x] 4.2 Document macOS-specific considerations (sleep prevention, launchd alternative)
- [x] 4.3 Document the full `claude -p` invocation command

## 5. Validation

- [x] 5.1 Validate skill structure using skill-creator's quick_validate.py
- [x] 5.2 Test skill invocation manually with `/email-assistant --channel test`
