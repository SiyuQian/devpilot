# Rejected Ideas Tracking Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the PM skill remember rejected/deferred ideas so it never re-recommends them.

**Architecture:** Add two new phases to the PM skill (Phase 1.5: load exclusions, Phase 4.5: record decisions). Rejected ideas stored as individual markdown files with YAML frontmatter in `docs/rejected/`.

**Tech Stack:** Markdown/YAML files, Claude Code tools (Glob, Read, Write, AskUserQuestion)

---

### Task 1: Create the docs/rejected directory with a README

**Files:**
- Create: `docs/rejected/README.md`

**Step 1: Create the directory and README**

Write `docs/rejected/README.md` with:

```markdown
# Rejected & Deferred Ideas

This directory tracks ideas that were evaluated by the PM skill and rejected or deferred by the user.

- `rejected` — Permanently rejected. PM skill will not re-recommend similar ideas.
- `deferred` — Not right now. PM skill may re-suggest if market conditions change significantly.

## File Format

Each file uses YAML frontmatter:

\```yaml
---
status: rejected | deferred
idea: "Feature Name"
date: YYYY-MM-DD
score: N            # PM skill total score (x/12)
reason: scope | direction | timing | duplicate | other
---
\```

Followed by a markdown body explaining the rejection reason and summarizing the original evidence.
```

**Step 2: Commit**

```bash
git add docs/rejected/README.md
git commit -m "docs: add rejected ideas directory with README"
```

---

### Task 2: Add Phase 1.5 to PM skill — Load Rejected Ideas

**Files:**
- Modify: `.claude/skills/pm/SKILL.md:12-19` (after Phase 1, before Phase 2)

**Step 1: Insert Phase 1.5 between Phase 1 and Phase 2**

After the Phase 1 section (line 19) and before the Phase 2 heading (line 21), insert:

```markdown
### Phase 1.5: Load Previously Rejected Ideas

Before launching research, check for previously rejected or deferred ideas:

1. Use the Glob tool to scan `docs/rejected/*.md` (skip `README.md`)
2. Read each file's YAML frontmatter to extract: `idea`, `status`, `reason`
3. Build two lists:
   - **Rejected list**: Ideas with `status: rejected` — these MUST be excluded from recommendations
   - **Deferred list**: Ideas with `status: deferred` — these may only be re-suggested if there is strong new evidence of changed market conditions

If no files exist in `docs/rejected/`, skip this phase and proceed to Phase 2.

Store these lists for use in Phase 2 agent prompts.
```

**Step 2: Verify the edit**

Read `.claude/skills/pm/SKILL.md` and confirm Phase 1.5 appears between Phase 1 and Phase 2 with correct indentation and heading level.

**Step 3: Commit**

```bash
git add .claude/skills/pm/SKILL.md
git commit -m "feat(pm): add Phase 1.5 — load rejected ideas before research"
```

---

### Task 3: Amend Phase 2 agent prompts with exclusion instructions

**Files:**
- Modify: `.claude/skills/pm/SKILL.md` (Phase 2 agent prompt templates)

**Step 1: Add exclusion block to Agent 1 (Competitor Analyst) prompt**

In the Agent 1 prompt template, before the final "Keep your response under 800 words" line, insert:

```
{rejected_ideas_block}
```

Where `{rejected_ideas_block}` is defined earlier in Phase 2 as:

```markdown
**IMPORTANT**: Before each agent prompt, if Phase 1.5 produced any rejected/deferred ideas, append this block before the 800-word constraint line:

\```
IMPORTANT: The following ideas have been previously evaluated and REJECTED by our team. Do NOT recommend features similar to these:
{for each rejected idea: "- {idea_name} (reason: {reason})"}

The following ideas were DEFERRED (not right now). Only mention them if you find strong evidence that market conditions have significantly changed since {date}:
{for each deferred idea: "- {idea_name} (deferred on: {date}, reason: {reason})"}
\```

If there are no rejected or deferred ideas, omit this block entirely.
```

Add this instruction block right after the "**IMPORTANT**: Launch all 3 agents in a single message" paragraph and before the agent subsections.

**Step 2: Verify the edit**

Read the Phase 2 section and confirm:
- The exclusion block instruction appears once, before the agent subsections
- It references `{rejected_ideas_block}` as a template variable
- Each agent's existing prompt template is NOT individually modified (the instruction applies to all three)

**Step 3: Commit**

```bash
git add .claude/skills/pm/SKILL.md
git commit -m "feat(pm): amend Phase 2 agent prompts with rejected ideas exclusion"
```

---

### Task 4: Add Phase 4.5 — Record Review Decisions

**Files:**
- Modify: `.claude/skills/pm/SKILL.md` (after Phase 4, before Key Rules)

**Step 1: Insert Phase 4.5 after Phase 4**

After the Phase 4 section (the "Then engage in discussion" bullet list, before "## Key Rules"), insert:

```markdown
### Phase 4.5: Record Review Decisions

After the user has made their decisions on which features to pursue, record rejected and deferred ideas:

1. **Ask for each recommended feature** — Use AskUserQuestion to ask the user's decision for each feature that they did NOT choose to pursue:
   - Options: "Reject permanently" / "Defer for later" / "Already accepted"
   - For rejected/deferred: ask reason with options: "scope" / "direction" / "timing" / "duplicate" / "other"
   - If "other", ask for a brief explanation

2. **Write rejection files** — For each rejected or deferred idea, use the Write tool to create `docs/rejected/{YYYY-MM-DD}-{idea-slug}.md`:

   ```markdown
   ---
   status: {rejected|deferred}
   idea: "{Feature Name}"
   date: {YYYY-MM-DD}
   score: {total_score}
   reason: "{reason}"
   ---

   ## Reason

   {User's explanation or generated summary based on reason category}

   ## Original Evidence Summary

   {2-3 bullet points from the Phase 3 synthesis for this feature}
   ```

   The `{idea-slug}` should be the feature name in lowercase-kebab-case (e.g., "Real-time Collaboration" → "real-time-collaboration").

3. **Commit** — Stage and commit all new rejection files:

   ```
   git add docs/rejected/
   git commit -m "docs: record rejected/deferred ideas from PM research"
   ```

4. **Confirm** — Tell the user which ideas were recorded and where the files are saved.
```

**Step 2: Verify the edit**

Read the end of SKILL.md and confirm Phase 4.5 appears between Phase 4 and Key Rules.

**Step 3: Commit**

```bash
git add .claude/skills/pm/SKILL.md
git commit -m "feat(pm): add Phase 4.5 — record rejected/deferred ideas after review"
```

---

### Task 5: Update Key Rules section

**Files:**
- Modify: `.claude/skills/pm/SKILL.md` (Key Rules section at the bottom)

**Step 1: Add two new rules**

Append these to the Key Rules bullet list:

```markdown
- **Check rejected ideas first** — Always run Phase 1.5 before research; never skip it
- **Record every decision** — Every idea that isn't accepted must be recorded in Phase 4.5 as rejected or deferred
```

**Step 2: Commit**

```bash
git add .claude/skills/pm/SKILL.md
git commit -m "feat(pm): add key rules for rejected ideas tracking"
```

---

### Task 6: Final verification

**Step 1: Read the complete SKILL.md**

Read the entire `.claude/skills/pm/SKILL.md` and verify:
- Phase order is: 1 → 1.5 → 2 → 3 → 4 → 4.5
- Phase 2 includes the exclusion block instruction
- Phase 4.5 includes file creation and commit steps
- Key Rules include the two new entries
- No broken markdown formatting

**Step 2: Verify docs/rejected/ exists**

```bash
ls docs/rejected/
```

Expected: `README.md`

**Step 3: Run existing tests to ensure nothing is broken**

```bash
make test
```

Expected: All tests pass (this change is skill-only, no Go code modified).
