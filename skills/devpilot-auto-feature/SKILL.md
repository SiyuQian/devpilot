---
name: devpilot-auto-feature
description: The complete end-to-end workflow for implementing OpenSpec changes with TDD and mandatory code review checkpoints between task chapters. This skill MUST be used instead of openspec-apply-change whenever the user wants quality gates, TDD, reviews between chapters, or a fully automated implement-review-archive-sync cycle. It wraps openspec-apply-change with superpowers:test-driven-development for every task, superpowers:requesting-code-review after each chapter, automatic openspec archival and spec syncing, and CLAUDE.md/README.md update checks. Triggers on: "auto feature", "implement with tdd", "implement with reviews", "build this feature end to end", "implement [change-name] with quality gates", "full workflow", "tdd and review", or ANY request to implement an OpenSpec change that mentions testing, reviews, quality, or thoroughness. Also triggers when resuming a partially-completed change with the auto-feature workflow. If the user just says "implement [change]" without quality keywords, use openspec-apply-change instead.
license: MIT
---

# Auto Feature

Implement OpenSpec changes end-to-end using test-driven development, with code review checkpoints after each chapter of tasks. This skill orchestrates multiple superpowers skills into a disciplined workflow that produces high-quality, reviewed code.

The workflow: for every task, write tests first, then implement. After finishing each chapter (a group of tasks sharing a major number — e.g., 1.1, 1.2, 1.3), pause and request a code review. Address all reasonable feedback before moving on. Once everything is done, archive the change, sync specs, and check if documentation needs updating.

## Why this workflow matters

Implementing features without review gates leads to compounding issues — a small design misstep in chapter 1 becomes a painful refactor in chapter 3. By reviewing after each chapter, problems surface early when they're cheap to fix. TDD ensures each task has verification built in from the start, so reviews can focus on design and intent rather than "does it work?"

## Workflow

### Phase 1: Setup

1. **Select the change** — Follow the same selection logic as `openspec-apply-change`:
   - If a name is provided, use it
   - Auto-select if only one active change exists
   - If ambiguous, run `openspec list --json` and ask the user to pick

2. **Load context** — Run `openspec instructions apply --change "<name>" --json` and read all context files (proposal, specs, design, tasks). Understand the full picture before touching code.

3. **Parse task structure** — Identify all tasks and group them by chapter (major number). For example:
   - Chapter 1: tasks 1.1, 1.2, 1.3
   - Chapter 2: tasks 2.1, 2.2
   - Chapter 3: tasks 3.1, 3.2, 3.3, 3.4

   Announce the plan: how many chapters, how many tasks per chapter, and that reviews will happen between chapters.

### Phase 2: Implement (repeat per chapter)

For each chapter:

#### Pre-check: Verify chapter completion status

Before starting a chapter, check if its tasks are already marked `[x]` in the tasks file. If all tasks in a chapter are checked off:

1. **Verify against source code** — Read the relevant source files to confirm the checked-off tasks actually correspond to real, working code. Don't trust checkboxes blindly — the code is the source of truth.
2. **If code matches tasks** — Skip this chapter entirely (including its review checkpoint, since that work was done outside this session). Move to the next chapter.
3. **If code doesn't match tasks** — Uncheck the tasks that aren't actually implemented, and treat them as pending. Proceed with TDD for those tasks.

This enables resuming partially-completed changes without redoing work, while ensuring nothing was marked done prematurely.

#### A. Implement tasks using TDD

For each task in the chapter:

1. **Invoke `superpowers:test-driven-development`** — Write failing tests that capture what the task requires, then write the minimal implementation to make them pass, then refactor.

2. **Mark the task complete** — Update `- [ ]` → `- [x]` in the tasks file.

3. **Commit** — Create a focused commit for the completed task. Use a descriptive message referencing the task number (e.g., `feat: implement user validation (task 1.2)`).

Keep changes scoped tightly to each task. If implementation reveals a design issue, pause and surface it rather than silently working around it.

#### B. Chapter review checkpoint

After completing all tasks in a chapter (and before starting the next chapter):

1. **Invoke `superpowers:requesting-code-review`** — This sends the work done in this chapter for review.

2. **Process feedback thoroughly** — Read every piece of feedback carefully. Apply all changes that are reasonable, including suggestions marked as "nice to have" or "minor." The bar is: if the feedback makes the code better and doesn't conflict with the design, do it. Only push back on feedback that would contradict the proposal/spec or introduce unnecessary complexity.

3. **Commit review fixes** — Create a separate commit for review-driven changes (e.g., `fix: address chapter 1 review feedback`).

4. **Verify everything still passes** — Run the full test suite after applying review feedback. TDD means the tests are there — use them.

Then proceed to the next chapter.

### Phase 3: Finalize

After all chapters are complete and reviewed:

#### A. Final verification

Run the full test suite one final time. Use `superpowers:verification-before-completion` to confirm everything is solid.

#### B. Documentation check

Check whether the changes require updates to project documentation:

- **CLAUDE.md** — If new commands, architecture changes, conventions, or build steps were introduced, update the relevant sections. Read the current CLAUDE.md first and compare against what changed.
- **README.md** — If user-facing features, installation steps, or usage patterns changed, update accordingly. Read the current README.md first.

Only update docs if the changes genuinely warrant it — don't add noise. If updates are needed, commit them separately (e.g., `docs: update CLAUDE.md with new X command`).

#### C. Archive and sync

1. **Archive the change** — Invoke the `openspec-archive-change` skill for this change. When prompted about syncing delta specs, choose to sync.

2. **Sync specs** — If the archive skill's sync prompt was skipped for any reason, explicitly invoke `openspec-sync-specs` to ensure main specs reflect all changes.

3. **Final status** — Display a summary:

```
## Auto Feature Complete

**Change:** <change-name>
**Tasks:** N/N complete ✓
**Chapters reviewed:** M/M ✓
**Specs:** Synced to main ✓
**Docs:** CLAUDE.md updated / README.md updated / No updates needed

All done. The feature has been implemented with TDD, reviewed chapter-by-chapter, and archived.
```

## Handling edge cases

- **Single-chapter changes** — Still do the review after the chapter. The checkpoint isn't skipped just because there's only one group.
- **Resuming partially-completed changes** — When some chapters are already done, the pre-check step verifies each completed chapter's code against its tasks. Verified chapters are skipped entirely (including their review checkpoint). Only remaining chapters get the full TDD + review treatment.
- **Tasks without chapter numbers** — If tasks use flat numbering (1, 2, 3 instead of 1.1, 1.2), treat every 3-5 tasks as a review checkpoint. Use natural boundaries in the task list (thematic groupings) when possible.
- **Review reveals design problems** — If code review feedback suggests a fundamental issue with the approach, pause and surface it to the user rather than silently patching things. The proposal/spec may need updating.
- **Test failures after review fixes** — If applying review feedback breaks existing tests, fix the tests as part of the review feedback commit. Don't leave broken tests between chapters.
- **Blocked tasks** — If a task can't be completed, stop and ask the user. Don't skip tasks or guess.

## Guardrails

- Follow the proposal and spec — don't add features beyond what's specified
- Every task gets TDD treatment, no exceptions
- Every chapter boundary gets a code review, no exceptions
- Commit frequently — at minimum once per task, plus once for review feedback
- Keep the user informed of progress (which chapter, which task, review status)
- Archive and sync are not optional — they ensure the spec stays the source of truth
