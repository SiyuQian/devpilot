---
name: developerkit:task-refiner
description: Improve Trello card task plans for the devkit runner. Use when user wants to refine, improve, or expand a Trello task/card plan. Triggers on /refine-task, "refine task", "improve card", "改进任务".
---

# Task Refiner

Improve Trello card task plans so they execute reliably with `devkit run`. Works one card at a time in two modes:

- **Refine**: Improve an existing structured plan (has headings and numbered steps)
- **Expand**: Generate a complete plan from a vague idea or short description

## Usage

```
/refine-task <card-url-or-name>
```

Argument is required. If a Trello URL (contains `trello.com/c/`), extract the card ID from the path. If a name/keyword, search for the card via the API.

## Process

### Step 1: Fetch Card Content

Read credentials and fetch the card:

```bash
TRELLO_KEY=$(cat ~/.config/devkit/credentials.json | python3 -c "import sys,json; print(json.load(sys.stdin)['trello']['api_key'])")
TRELLO_TOKEN=$(cat ~/.config/devkit/credentials.json | python3 -c "import sys,json; print(json.load(sys.stdin)['trello']['token'])")
```

If credentials file is missing or fields are absent, tell user to run `devkit login trello` and stop.

Fetch the card:

```bash
curl -s "https://api.trello.com/1/cards/{id}?fields=name,desc&key=$TRELLO_KEY&token=$TRELLO_TOKEN"
```

If argument was a name, search first:

```bash
curl -s "https://api.trello.com/1/search?query={name}&modelTypes=cards&key=$TRELLO_KEY&token=$TRELLO_TOKEN"
```

Pick the best match and confirm with the user before proceeding.

### Step 2: Analyze Codebase

Build context for writing a good plan:

1. Read `CLAUDE.md` for project conventions and structure
2. Explore relevant code with Glob and Grep based on the card topic
3. Check `docs/rejected/` for previously rejected approaches to avoid
4. Check `docs/plans/` for existing design decisions to stay consistent with

### Step 3: Detect Mode

Apply this heuristic to the card description:

- **Refine mode**: Description contains markdown headings (`#`, `##`, `###`) AND numbered steps (`1.`, `### 1.`, ordered list items)
- **Expand mode**: Everything else — short descriptions, unstructured bullets, feature requests, vague ideas, or empty descriptions

### Step 4: Propose Directions (Expand mode only)

**Skip this step entirely if in Refine mode — go straight to Step 5.**

Before committing to a full plan, propose 2-3 implementation directions for the user to choose from. Each direction should:

1. Have a short name (e.g., "A: Interactive wizard", "B: Subcommand-per-step")
2. Describe the approach in 2-3 sentences
3. List key trade-offs: complexity, flexibility, consistency with codebase patterns
4. Note which existing patterns it follows or departs from

Present the directions and **wait for the user to choose** before proceeding. The user may also combine ideas from multiple directions or suggest a different approach entirely.

### Step 5: Improve the Plan

Read the file `references/quality-checklist.md` (relative to this skill).

**If Refine mode:**

1. Evaluate the existing plan against each quality dimension (Specificity, Executability, Test Strategy, Architecture Consistency, Edge Cases, Dependency Order)
2. Identify gaps — missing file paths, vague steps, missing tests, wrong ordering
3. Fix each gap while preserving the plan's intent and structure
4. Ensure verification steps exist at the end

**If Expand mode:**

1. Use the plan template from the quality checklist
2. Based on the direction the user chose in Step 4, explore the codebase to determine the right files, packages, and patterns
3. Generate a complete plan with concrete file paths, code approach, and test strategy
4. Ensure every step is executable by Claude without human judgment

### Step 6: Confirm and Update

1. Show the improved plan to the user in full
2. Wait for explicit approval before updating
3. Update the card description:

```bash
curl -s -X PUT "https://api.trello.com/1/cards/{id}?key=$TRELLO_KEY&token=$TRELLO_TOKEN" \
  --data-urlencode "desc={improved-plan}"
```

4. Add a comment logging the refinement:

```bash
curl -s -X POST "https://api.trello.com/1/cards/{id}/actions/comments?key=$TRELLO_KEY&token=$TRELLO_TOKEN" \
  --data-urlencode "text=Plan refined by task-refiner skill (mode: {refine|expand})"
```

## API Reference

Base URL: `https://api.trello.com/1`

All requests include `key=$TRELLO_KEY&token=$TRELLO_TOKEN` as query parameters.

| Operation | Method | Endpoint | Key params |
|-----------|--------|----------|------------|
| Get card | GET | `/cards/{id}?fields=name,desc` | card ID |
| Search cards | GET | `/search?query={q}&modelTypes=cards` | query string |
| Update card | PUT | `/cards/{id}` | `desc` (URL-encoded) |
| Add comment | POST | `/cards/{id}/actions/comments` | `text` (URL-encoded) |

## Important Rules

- Always process one card at a time
- Always confirm with the user before updating the card
- Always add a comment after updating to log the refinement
- URL-encode the description when updating via PUT (use `--data-urlencode`)
- Check `docs/rejected/` to avoid recommending previously rejected approaches
- If the card cannot be found or fetched, report the error clearly and stop
