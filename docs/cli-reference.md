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
go test ./internal/taskrunner/ -v                     # Single package, verbose
```

## Authentication & Status

```bash
devpilot login trello                # Authenticate with Trello (API key + token)
devpilot logout trello               # Remove stored credentials
devpilot status                      # Show authentication status for all services
```

## Project Setup

```bash
devpilot init                        # Interactive project setup wizard
devpilot init -y                     # Accept all defaults
```

## Queueing Work

```bash
devpilot push <plan.md> --board "Board Name"                 # Create Trello card from plan file
devpilot push <plan.md> --board "Board Name" --list "Ready"  # Specify target list (default: Ready)
```

## Running the Autonomous Runner

```bash
devpilot run --board "Board Name"                          # Start autonomous task runner (TUI mode)
devpilot run --board "Board Name" --no-tui                 # Plain text output (no dashboard)
devpilot run --board "Board Name" --once --dry-run         # Test with one card, no execution
devpilot run --board "Board Name" --interval 60            # Poll every 60s (default: 300)
devpilot run --board "Board Name" --timeout 45             # 45min per-task timeout (default: 30)
devpilot run --board "Board Name" --review-timeout 0       # Disable auto code review
```

## OpenSpec Sync

```bash
devpilot sync                        # Sync OpenSpec changes to board/issues
devpilot sync --board "Board Name"   # Override board
devpilot sync --source github        # Override source
```

## Gmail Summary

```bash
devpilot gmail summary                         # Dry run: summarize unread emails (won't mark as read)
devpilot gmail summary --channel daily-digest  # Send summary to a Slack channel (marks as read)
devpilot gmail summary --dm U0123ABCDE         # Send summary as a DM (marks as read)
devpilot gmail summary --no-mark-read=false    # Explicitly mark emails as read without sending
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

## Code Review

```bash
devpilot review <pr-url>                                     # AI-powered code review, posts to PR
devpilot review <pr-url> --no-post                           # Review without posting to PR
devpilot review <pr-url> --model claude-sonnet-4-6-20250514  # Review with custom model
devpilot review <pr-url> --dry-run                           # Print assembled prompt without executing
```

## Misc

```bash
devpilot commit     # Generate commit message from staged changes
devpilot readme     # Generate or improve README.md
```

## Skill Helper Scripts (Python 3)

```bash
python3 .claude/skills/skill-creator/scripts/init_skill.py       # Scaffold a new skill
python3 .claude/skills/skill-creator/scripts/package_skill.py    # Package a skill for distribution
python3 .claude/skills/skill-creator/scripts/quick_validate.py   # Validate skill structure
```
