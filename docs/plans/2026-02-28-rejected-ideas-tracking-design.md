# Rejected Ideas Tracking — Design

## Problem

When the PM skill generates feature recommendations and the user rejects some of them, there is no mechanism to remember that decision. The next time the PM skill runs, it may recommend the same features again, wasting research effort and user time.

## Solution

Integrate rejected idea tracking directly into the PM skill workflow:

1. **Record** — After Phase 4 review, automatically save rejected/deferred ideas to individual files
2. **Filter** — Before Phase 2 research, read the rejected ideas list and pass it to agent prompts as exclusions

## Storage Format

Each rejected idea is stored as `docs/rejected/{YYYY-MM-DD}-{idea-slug}.md`:

```markdown
---
status: rejected          # rejected | deferred
idea: "Real-time Collaboration"
date: 2026-02-28
score: 8                  # PM skill total score (x/12)
reason: "scope"           # scope | direction | timing | duplicate | other
---

## Reason

Does not align with current product direction. We focus on single-developer tools, not collaboration.

## Original Evidence Summary

- Competitors: Figma, Miro already dominate collaboration market
- User demand: Medium (Reddit 3 threads)
- Trend: Growing but saturated
```

### Status Values

- `rejected` — Permanently rejected. PM skill should exclude similar ideas.
- `deferred` — Not right now. PM skill may re-suggest only if market conditions have significantly changed, and must flag it as "previously deferred".

## PM Skill Changes

### Phase 1.5 (New): Load Rejected Ideas

Between Phase 1 (Scope Clarification) and Phase 2 (Research):

1. Glob scan `docs/rejected/*.md`
2. Read each file's YAML frontmatter (idea name + status + reason)
3. Build an exclusion list for Phase 2 agent prompts

### Phase 2: Agent Prompt Amendment

Append to each agent's prompt:

```
IMPORTANT: The following ideas have been previously evaluated and rejected.
Do NOT recommend features that are similar to these:
{rejected_ideas_list}

The following ideas were deferred (not right now). You may mention them
ONLY if market conditions have significantly changed:
{deferred_ideas_list}
```

### Phase 4.5 (New): Record Review Decisions

After Phase 4 discussion concludes:

1. For each recommended feature, ask: "Accept / Reject / Defer"
2. For rejected/deferred ideas, ask reason (scope / direction / timing / duplicate / other)
3. Write `docs/rejected/{date}-{slug}.md` with frontmatter + reason + evidence summary
4. Commit the file(s)

## Files Changed

| File | Change |
|------|--------|
| `.claude/skills/pm/SKILL.md` | Add Phase 1.5, amend Phase 2 prompts, add Phase 4.5 |
| `docs/rejected/` | New directory (created on first rejection) |

## Files NOT Changed

- No Go CLI code changes needed
- No Trello integration changes
- No task-executor changes
