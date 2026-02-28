# Task Refiner Skill Design

**Status:** Approved
**Date:** 2026-02-28

## Overview

A Claude Code skill (`task-refiner`) that reads a Trello card, analyzes the current codebase, and interactively improves the card's plan content — either refining an existing structured plan or expanding a vague idea into a complete executable plan. The improved content is written back to Trello after user confirmation.

## Motivation

The current workflow (`devkit push` → Trello → `devkit run`) depends on high-quality execution plans in Trello cards. Plans with vague steps, missing file paths, or no test strategy cause the task-executor to fail or produce poor results. Currently there's no tool to bridge the gap between a rough idea and a runner-ready plan.

## Skill Definition

- **Name:** `task-refiner`
- **Location:** `.claude/skills/task-refiner/SKILL.md`
- **Trigger:** `/refine-task`
- **Scope:** One card at a time (human review is the bottleneck)

### Usage

```
/refine-task <card-url>       # Via Trello card URL (preferred)
/refine-task <card-name>      # Via card name search
```

## Work Modes

The skill auto-detects which mode to use based on card content:

### Refine Mode

Triggered when the card already contains a structured plan (headings, numbered steps, file paths).

Reviews and improves the plan along these dimensions:

1. **Specificity** — Steps must have concrete file paths and commands. "Update config" → "Edit `internal/config/config.go`, add `XxxField` to the `Config` struct"
2. **Executability** — Each step must be directly executable by Claude without human judgment calls
3. **Test strategy** — Must include unit tests, verification commands, and build checks
4. **Architecture consistency** — Plan must align with existing codebase patterns (informed by codebase analysis)
5. **Edge cases** — Error handling and boundary conditions considered
6. **Dependency order** — Steps ordered correctly with clear dependencies

### Expand Mode

Triggered when the card contains a vague idea, feature request, or brief requirement description.

Generates a complete implementation plan by:

1. Understanding the intent behind the idea
2. Exploring the codebase to determine the right implementation approach
3. Checking `docs/rejected/` to avoid recommending previously rejected approaches
4. Checking `docs/plans/` for relevant existing design decisions
5. Generating a step-by-step plan with file paths, commands, and test strategy
6. Ensuring output format is compatible with task-executor expectations

## Core Flow

### Step 1 — Fetch Card Content

- Read Trello credentials from `~/.config/devkit/credentials.json`
- Extract card ID from URL, or search by name via Trello search API
- Fetch card name and description via `GET /cards/{id}`

### Step 2 — Analyze Codebase Context

- Read `CLAUDE.md` for project architecture overview
- Use Glob/Grep to explore relevant files and patterns
- Read `docs/rejected/` to avoid re-recommending rejected ideas
- Read `docs/plans/` for existing design decisions

### Step 3 — Detect Mode and Improve

Auto-detect based on card content structure:
- Has headings + numbered steps + file references → **Refine mode**
- Otherwise → **Expand mode**

Apply the appropriate improvement strategy (see Work Modes above).

### Step 4 — Interactive Confirmation

- Present the improved plan alongside the original for comparison
- Allow the user to request further changes
- Only proceed to update after explicit user confirmation

### Step 5 — Update Trello Card

- Update card description via `PUT /cards/{id}` with the improved plan
- Add a comment recording the refinement: "Refined by task-refiner skill"

## Trello API Integration

Reuses the existing Trello skill's curl-based API pattern:

```bash
# Read credentials
KEY=$(jq -r '.trello.apiKey' ~/.config/devkit/credentials.json)
TOKEN=$(jq -r '.trello.token' ~/.config/devkit/credentials.json)

# Fetch card
curl -s "https://api.trello.com/1/cards/{cardId}?key=$KEY&token=$TOKEN"

# Search card by name
curl -s "https://api.trello.com/1/search?query={name}&modelTypes=cards&key=$KEY&token=$TOKEN"

# Update card description
curl -s -X PUT "https://api.trello.com/1/cards/{cardId}" \
  -d "key=$KEY&token=$TOKEN&desc={newDesc}"

# Add comment
curl -s -X POST "https://api.trello.com/1/cards/{cardId}/actions/comments" \
  -d "key=$KEY&token=$TOKEN&text=Refined by task-refiner skill"
```

## Plan Quality Standard

Output plans must be compatible with the task-executor skill. A high-quality plan includes:

**Required elements:**
- Clear title (`# Heading`)
- Ordered implementation steps
- Each step specifies: what to do, which files to touch, how to do it
- Verification steps (test commands, build checks)

**Format:** Markdown with `# Title`, `## Section` headings, and numbered steps under each section.

## Non-Goals

- Batch processing of multiple cards (human review is the bottleneck)
- Auto-refinement without human confirmation
- Changes to Go code or CLI commands
- Integration into the `devkit run` pipeline

## Dependencies

- Trello API credentials configured via `devkit login trello`
- `jq` available on PATH for JSON parsing
- `curl` available on PATH for API calls
