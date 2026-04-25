# Per-Task Code Review Checkpoint

After every implementer subagent returns DONE on a task, the controller invokes `superpowers:requesting-code-review` against the diff that task produced. No exceptions: no task is marked `completed` in TodoWrite without a clean review pass.

**REQUIRED SUB-SKILL:** Use `superpowers:requesting-code-review` to dispatch the `superpowers:code-reviewer` subagent. Do not write your own ad-hoc review or rely on `devpilot-pr-review` for this — that skill is for the published GitHub review at the end, not the per-task internal gate.

## Why per-task and not only at PR time

- `devpilot-pr-review` runs once, at PR creation, against the union of all task diffs. It catches what's wrong with the whole PR; it does **not** catch a bad task before the next task builds on top of it.
- A fresh-context reviewer (the `superpowers:code-reviewer` subagent) sees the task diff without the implementer's framing. Self-review is not a substitute — same context, same blind spots.
- Catching a quality issue between tasks 1 and 2 is cheap (one re-dispatch). Catching the same issue at PR time means the next task already depends on the bug.

The two reviews are complementary, not redundant. Per-task review is the internal gate; the published PR review is the artifact attached to the PR.

## How to invoke

For each task that just returned DONE, dispatch the code-reviewer following `superpowers:requesting-code-review`:

```bash
# In the controller, between implementer DONE and TodoWrite completed.
# Cwd at this point is $WORKTREE — the issue's git worktree. git resolves HEAD
# against the worktree's branch (fix/issue-<num>-<slug>), not the main checkout.

BASE_SHA=$(git rev-parse HEAD~"$commits_in_this_task")  # commits the implementer made
HEAD_SHA=$(git rev-parse HEAD)
```

The reviewer subagent inherits `$WORKTREE` as cwd, so its `git diff $BASE_SHA..$HEAD_SHA`, file reads, and any `make` invocations apply to this issue's tree — not to the main checkout. Do **not** `cd "$MAIN"` before invoking the reviewer; the SHAs above only resolve correctly inside the worktree.

Dispatch the reviewer with the template fields from `superpowers:requesting-code-review`:

- `WHAT_WAS_IMPLEMENTED` — the task's verb-first title (e.g., `"Add regression test in cmd/devpilot/run_test.go"`).
- `PLAN_OR_REQUIREMENTS` — the task's acceptance criteria from `subagent-spec.md`, verbatim.
- `BASE_SHA` / `HEAD_SHA` — the SHAs above. The reviewer reads only the diff between them.
- `DESCRIPTION` — one line: which task in the issue, what it changes.

Use `subagent_type: superpowers:code-reviewer` so the reviewer follows the `superpowers:requesting-code-review` template (`code-reviewer.md`).

## Acting on the review

The reviewer returns a verdict in the standard shape: **Strengths**, **Issues by severity (Critical / Important / Minor)**, **Recommendations**, **Assessment (Ready/With fixes/No)**.

| Verdict | Action |
|---|---|
| Ready, no issues | Mark TodoWrite `completed`. Continue to next task. |
| Ready, **Minor** only | Decide per-issue: fix small Minor items now via re-dispatch (`Approach A` below); or capture them in a note and continue. Default: fix now if it takes ≤1 line per item. |
| With fixes — **Important** | **Re-dispatch the same implementer subagent** with the verbatim review feedback (`Approach A` below). Re-review. Loop until clean. |
| With fixes — **Critical** | Same as Important, but if two re-review rounds still surface Critical issues, escalate the issue itself to `NEEDS-HUMAN` and stop work on this issue. |

### Approach A — re-dispatch the implementer

Same subagent type, same task spec, with one extra block appended:

```markdown
## Code review feedback to address

The previous attempt produced these issues from a fresh-context reviewer. Fix all of them, do not relitigate them. Push fixes as new commits on the same branch.

<paste the reviewer's Issues section verbatim>
```

Same implementer, fresh dispatch — the implementer subagent in this skill is single-turn-per-dispatch, so each fix round is a new call with the prior code already on the branch.

### Cap on review rounds

- **2 rounds maximum** of "implementer fixes → reviewer re-checks" per task.
- If round 3 would be needed, the task is in a fight with the codebase or the spec is wrong. Stop, mark the issue `NEEDS-HUMAN` with the diff so far on a pushed draft branch, and move on.

## What the per-task review does NOT cover

- **PR-level concerns** — overall PR shape, blast radius across all tasks, the published review on GitHub. That's `devpilot-pr-review` after PR creation.
- **Behavior of the issue's premise** — whether the verdict was right. Reviewing the implementation is not a re-litigation of the verdict; if you find yourself disagreeing with the verdict at this stage, push the disagreement back to step 3 of `SKILL.md` and re-investigate.
- **Cross-issue coupling** — what other open issues this fix interacts with. The loop handles those one at a time; flagging them in the reviewer's `Recommendations` is fine but does not block this task.

## Anti-shortcuts (close the loopholes)

- **"It's a one-line change, skip the review"** — One-line changes pass code review in 30 seconds. Skipping it teaches the loop to skip more.
- **"`devpilot-pr-review` will catch it later"** — That review runs once on the union diff. It cannot tell you "task 2 broke an assumption from task 1" — by then, task 1 already shipped to the branch.
- **"The implementer's self-review is enough"** — Same context. Same blind spots. The reviewer subagent has no shared history; that is the point.
- **"I'll fix the review feedback myself in the controller, faster than re-dispatching"** — Context pollution. The controller orchestrates; the implementer implements. Re-dispatch.
- **"Ready / Minor → mark complete and ignore Minors"** — The default is fix-now if each Minor is ≤1 line. Recurring Minors signal a pattern the next task will repeat.
- **"Two rounds wasn't enough, just one more"** — At round 3 the task is fighting the codebase. Escalate `NEEDS-HUMAN`. The cap exists so the loop terminates.
