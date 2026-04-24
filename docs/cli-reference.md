# DevPilot CLI Reference

Full command surface. CLAUDE.md links here; this file is read on demand.

## Build & Development

```bash
make build                         # Build binary to bin/devpilot
make test                          # Run all tests (go test ./...)
make lint                          # Run golangci-lint (must pass before commit)
make lint-fix                      # Auto-fix lint issues where possible
make run ARGS="--help"             # Build and run with arguments
make clean                         # Remove bin/
```

Run a single test:
```bash
go test ./internal/skillmgr/ -run TestInstallSkill   # Single test by name
go test ./internal/skillmgr/ -v                       # Single package, verbose
```

## Authentication & Status

```bash
devpilot login trello                # Authenticate with Trello (API key + token)
devpilot login gmail                 # Authenticate with Gmail (OAuth)
devpilot login slack                 # Authenticate with Slack (OAuth)
devpilot logout <service>            # Remove stored credentials
devpilot status                      # Show authentication status for all services
```

## Project Setup

```bash
devpilot init                        # Interactive project setup wizard
devpilot init -y                     # Accept all defaults
```

## Skills

```bash
devpilot skill add <name>                       # Install a skill (prompts for project/user level)
devpilot skill add <name>@<ref>                 # Install at specific git ref
devpilot skill add <name> --level user          # Install at user level non-interactively
devpilot skill add --all                        # Install every skill in the catalog
devpilot skill add --all --level project        # Bulk install at project level, no prompt
devpilot skill list                             # List available skills with install status
devpilot skill list --installed                 # List only installed skills
```

## Trello

```bash
devpilot push <plan.md> --board "Board Name"                 # Create Trello card from plan file
devpilot push <plan.md> --board "Board Name" --list "Ready"  # Specify target list (default: Ready)
```

## Gmail

```bash
devpilot gmail list                            # List recent emails
devpilot gmail list --unread --limit 10        # Filter
devpilot gmail read <id>                       # Display full email
devpilot gmail mark-read <id...>               # Mark as read
devpilot gmail bulk-mark-read --query "..."    # Bulk mark by Gmail query
devpilot gmail summary                         # Dry run: summarize unread emails (won't mark as read)
devpilot gmail summary --channel daily-digest  # Send summary to a Slack channel (marks as read)
devpilot gmail summary --dm U0123ABCDE         # Send summary as a DM (marks as read)
```

## Slack

```bash
devpilot slack send --channel "#general" --text "hi"   # Send a Slack message
```

## Generation

```bash
devpilot commit                             # Stage changes, generate conventional commit message, confirm, commit
devpilot commit -m "context for AI"         # Pass extra context to the model
devpilot commit --model claude-haiku-4-5    # Override model
devpilot commit --dry-run                   # Print generated message without committing
```

## Skill Helper Scripts (Python 3)

```bash
python3 .claude/skills/skill-creator/scripts/init_skill.py       # Scaffold a new skill
python3 .claude/skills/skill-creator/scripts/package_skill.py    # Package a skill for distribution
python3 .claude/skills/skill-creator/scripts/quick_validate.py   # Validate skill structure
```
