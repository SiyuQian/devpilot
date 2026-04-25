# Decomposing an Issue into Tasks

After a REAL verdict on issue `#N`, the controller decomposes the fix into a small ordered list of tasks before dispatching any implementer. Each task gets its own implementer subagent and its own code-review checkpoint (`per-task-review.md`).

**Why decompose at all.** The resolve-issues loop runs many issues; the controller's context window is the scarce resource. A single big subagent call hides all the work behind one summary, leaves no review checkpoint until the end, and produces diffs the per-task quality gate cannot catch incrementally. Splitting into tasks is the same `superpowers:subagent-driven-development` pattern: fresh implementer per task, code review after each.

## Sizing rules

- **1 task** when the fix is one localized change in one file plus its test (most repo-scan findings).
- **2–3 tasks** when the fix splits into clearly separable pieces: e.g. *(a) write the failing regression test, (b) fix the bug, (c) update the docs/CHANGELOG*. Or when the fix touches one production file and one test file with non-trivial restructuring in each.
- **More than 3 tasks** is a smell: the issue is likely too coarse and should have been multiple issues. Stop, escalate as `NEEDS-HUMAN`, ask the user to split the issue.

The cap matters. A four-task fix burns four implementer dispatches plus four code-review dispatches plus four re-dispatch loops on review feedback — that is most of an iteration's budget in one issue.

## Signals that justify >1 task

- The Evidence block points at two distinct call sites or two distinct symptoms.
- The fix changes a public API surface and the test suite needs to be migrated separately.
- A regression test must land before the production fix to demonstrate the failing case (TDD on the issue itself).
- Documentation, CLAUDE.md, or generated artifacts must be updated alongside the code change.

If none of those apply, ship as 1 task.

## Output: a TodoWrite list scoped to the issue

The decomposition lives in the controller's TodoWrite for *this* issue iteration only. Subjects use the form `#<num>: <verb-first short title>`, so the list reads top-to-bottom as the fix plan.

```
TodoWrite (issue #142):
  - "#142 task 1/2: Add regression test in cmd/devpilot/run_test.go"
  - "#142 task 2/2: Sanitize user-supplied path in cmd/devpilot/run.go"
```

Each todo's `description` field carries the per-task implementer-spec hand-off (see `subagent-spec.md`). When all of an issue's tasks are completed, run the final verify step in `SKILL.md` and proceed to PR creation.

## What goes in each task

A task is the smallest unit that:

1. Compiles and passes `make test` / `make lint` on its own (a half-applied fix that breaks the build is not a task).
2. Has a clear acceptance criterion the implementer can self-check (a test name, a behavior change, a file edit).
3. Can be reviewed in isolation by `superpowers:requesting-code-review` — the reviewer can tell whether *this* task is correct without needing later tasks for context.

If a task can't satisfy all three, merge it back into its neighbor.

## Hand-off into the per-task loop

For each task, in the order the controller listed them:

1. Mark the task `in_progress` in TodoWrite.
2. Dispatch the implementer subagent with the per-task spec from `subagent-spec.md`.
3. Run the per-task review (`per-task-review.md`).
4. On clean review, mark the task `completed` and continue to the next.

**Tasks are sequential, never parallel.** They share a single branch and a single PR; parallel implementers stomp on each other's diffs. This is explicit in `superpowers:subagent-driven-development` — do not dispatch multiple implementer subagents at once.

## When decomposition itself fails

- **You can't write concrete acceptance criteria for a task** → the verdict step skipped enough investigation. Go back to step 3 (Investigate) on the issue. Don't make up criteria.
- **Every task you draft says "fix the bug" with no measurable check** → not enough verdict context. Re-investigate, re-decompose, or escalate `NEEDS-HUMAN`.
- **The user asked for "just fix it, don't bother with tasks"** → the loop still requires at least one task (one implementer + one review). The minimum is 1, not 0. The per-task review is non-negotiable; see `per-task-review.md`.
