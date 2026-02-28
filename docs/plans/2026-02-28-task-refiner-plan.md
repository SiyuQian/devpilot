# Task Refiner Skill Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a Claude Code skill that reads a Trello card, analyzes the codebase, and interactively improves the card's plan content before writing it back.

**Architecture:** A pure skill (no Go code changes) in `.claude/skills/task-refiner/` with a SKILL.md defining the workflow and a `references/` directory for the quality checklist. Reuses Trello API patterns from the existing trello skill (curl + credentials from `~/.config/devkit/credentials.json`).

**Tech Stack:** Markdown (SKILL.md), Bash/curl (Trello API), Python 3 one-liners (JSON parsing)

---

### Task 1: Create Skill Directory Structure

**Files:**
- Create: `.claude/skills/task-refiner/SKILL.md`
- Create: `.claude/skills/task-refiner/references/quality-checklist.md`

**Step 1: Create the directory**

Run:
```bash
mkdir -p .claude/skills/task-refiner/references
```

**Step 2: Verify directory exists**

Run:
```bash
ls -la .claude/skills/task-refiner/
```
Expected: Empty directory with `references/` subdirectory.

**Step 3: Commit**

```bash
git add .claude/skills/task-refiner/
git commit -m "chore: scaffold task-refiner skill directory"
```

---

### Task 2: Write the Quality Checklist Reference

This file defines the plan quality dimensions used by both Refine and Expand modes. Separated from SKILL.md to keep the main file concise (progressive disclosure).

**Files:**
- Create: `.claude/skills/task-refiner/references/quality-checklist.md`

**Step 1: Write the quality checklist**

Create `.claude/skills/task-refiner/references/quality-checklist.md` with the following content:

```markdown
# Plan Quality Checklist

Use this checklist to evaluate and improve Trello card plans. Read this file when performing refinement.

## Required Elements

Every plan must have:

- [ ] Clear title as `# Heading` (becomes Trello card name)
- [ ] Goal statement: one sentence describing what this builds
- [ ] Ordered implementation steps (numbered list)
- [ ] Each step specifies: what to do, which files to touch, how to do it
- [ ] Verification steps at the end (test commands, build checks)

## Quality Dimensions

### 1. Specificity

- [ ] Steps reference concrete file paths (e.g., `internal/config/config.go`)
- [ ] Steps include actual code snippets or describe exact changes
- [ ] No vague language: "update config" → "add `TimeoutField` to `Config` struct in `internal/config/config.go`"

### 2. Executability

- [ ] Each step can be executed by Claude without human judgment
- [ ] No steps that say "decide how to..." or "figure out..."
- [ ] External dependencies are documented (APIs, tools, libraries)

### 3. Test Strategy

- [ ] Unit tests specified for new functions/methods
- [ ] Test file paths included (e.g., `internal/foo/foo_test.go`)
- [ ] Verification commands listed (e.g., `go test ./internal/foo/...`)
- [ ] Build check included (e.g., `go build ./...`)

### 4. Architecture Consistency

- [ ] Uses patterns already in the codebase (check via codebase analysis)
- [ ] Follows existing naming conventions
- [ ] Respects package boundaries
- [ ] Does not contradict decisions in `docs/plans/`

### 5. Edge Cases

- [ ] Error handling considered for each step
- [ ] Input validation at system boundaries
- [ ] Graceful degradation where appropriate

### 6. Dependency Order

- [ ] Steps are ordered so dependencies come first
- [ ] No circular dependencies between steps
- [ ] Shared utilities created before code that uses them

## Plan Template

When expanding a vague idea, generate a plan following this structure:

```
# [Feature Name]

**Goal:** [One sentence]

**Architecture:** [2-3 sentences about approach]

## Steps

### 1. [First component/change]

**Files:**
- Create: `path/to/new/file.go`
- Modify: `path/to/existing/file.go`
- Test: `path/to/file_test.go`

**What to do:**
[Concrete description with code snippets]

**Verification:**
- Run: `go test ./path/to/...`
- Expected: All tests pass

### 2. [Next component/change]
...

## Final Verification

- Run: `go test ./...`
- Run: `go build ./...`
- Manual check: [any manual verification needed]
```
```

**Step 2: Verify the file reads well**

Run:
```bash
wc -l .claude/skills/task-refiner/references/quality-checklist.md
```
Expected: Approximately 80-90 lines.

**Step 3: Commit**

```bash
git add .claude/skills/task-refiner/references/quality-checklist.md
git commit -m "docs: add plan quality checklist for task-refiner skill"
```

---

### Task 3: Write SKILL.md — Frontmatter and Overview

**Files:**
- Create: `.claude/skills/task-refiner/SKILL.md`

**Step 1: Write the SKILL.md file**

Create `.claude/skills/task-refiner/SKILL.md` with the complete skill definition. The file must include:

**Frontmatter:**
```yaml
---
name: task-refiner
description: Improve Trello card task plans for the devkit runner. Use when user wants to refine, improve, or expand a Trello task/card plan. Triggers on /refine-task, "refine task", "improve card", "改进任务".
---
```

**Body sections (in order):**

1. **Overview** — What the skill does, one-card-at-a-time constraint
2. **Usage** — `/refine-task <card-url-or-name>`, argument parsing
3. **Process** — The 5-step numbered workflow:
   - Step 1: Fetch Card Content (credential reading pattern, curl commands for card by URL and by search)
   - Step 2: Analyze Codebase (read CLAUDE.md, explore with Glob/Grep, check docs/rejected/ and docs/plans/)
   - Step 3: Detect Mode (structured plan → Refine, vague → Expand)
   - Step 4: Improve the Plan
     - Refine: Read `references/quality-checklist.md`, evaluate each dimension, fix gaps
     - Expand: Read `references/quality-checklist.md`, use template, generate full plan
   - Step 5: Confirm and Update (show diff, get user approval, PUT to Trello, add comment)
4. **Credential Reading** — Exact bash pattern:
   ```bash
   TRELLO_KEY=$(cat ~/.config/devkit/credentials.json | python3 -c "import sys,json; print(json.load(sys.stdin)['trello']['api_key'])")
   TRELLO_TOKEN=$(cat ~/.config/devkit/credentials.json | python3 -c "import sys,json; print(json.load(sys.stdin)['trello']['token'])")
   ```
5. **API Reference** — Card fetch, search, update, comment curl commands
6. **Mode Detection** — Heuristic: if card description has `#` headings AND numbered steps → Refine; otherwise → Expand
7. **Important Rules** — One card at a time, always confirm before updating, always add comment logging refinement

The full SKILL.md should be under 200 lines (well within the 500-line guideline). The quality checklist stays in `references/` for progressive disclosure.

**Step 2: Count lines to verify conciseness**

Run:
```bash
wc -l .claude/skills/task-refiner/SKILL.md
```
Expected: Under 200 lines.

**Step 3: Verify frontmatter is valid YAML**

Run:
```bash
head -4 .claude/skills/task-refiner/SKILL.md
```
Expected: `---`, `name: task-refiner`, `description: ...`, `---`

**Step 4: Commit**

```bash
git add .claude/skills/task-refiner/SKILL.md
git commit -m "feat: add task-refiner skill for improving Trello card plans"
```

---

### Task 4: Manual Validation

**Step 1: Verify skill structure matches conventions**

Compare with existing skills:
```bash
ls -la .claude/skills/trello/
ls -la .claude/skills/task-executor/
ls -la .claude/skills/task-refiner/
```
Expected: Similar structure — `SKILL.md` at root, optional subdirectories.

**Step 2: Verify credential reading pattern matches trello skill**

Search for the credential pattern in both skills to confirm consistency:
```bash
grep -n "credentials.json" .claude/skills/task-refiner/SKILL.md
grep -n "credentials.json" .claude/skills/trello/SKILL.md
```
Expected: Same `~/.config/devkit/credentials.json` path and similar Python one-liner pattern.

**Step 3: Verify references are mentioned in SKILL.md**

```bash
grep -n "quality-checklist" .claude/skills/task-refiner/SKILL.md
```
Expected: At least one reference to `references/quality-checklist.md` with instruction on when to read it.

**Step 4: Test skill invocation trigger**

Verify the description includes the right trigger words:
```bash
head -5 .claude/skills/task-refiner/SKILL.md | grep -i "refine-task\|improve\|改进"
```
Expected: Trigger keywords present in the description line.

**Step 5: Final commit if any fixes needed**

If any issues were found and fixed:
```bash
git add .claude/skills/task-refiner/
git commit -m "fix: address validation issues in task-refiner skill"
```

---

## Final Verification

- Run: `find .claude/skills/task-refiner/ -type f` — all expected files exist
- Run: `wc -l .claude/skills/task-refiner/SKILL.md` — under 200 lines
- Run: `wc -l .claude/skills/task-refiner/references/quality-checklist.md` — under 100 lines
- Manual check: Read SKILL.md end-to-end and confirm the workflow is clear and complete
