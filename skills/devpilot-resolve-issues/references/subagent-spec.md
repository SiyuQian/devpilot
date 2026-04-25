# Per-Task Implementer Spec

This is the prompt the controller hands to a single implementer subagent for **one task** of an issue's fix (issues are decomposed first — see `task-decomposition.md`). One implementer subagent per task. Never bundle multiple tasks into one spec; never run two implementers in parallel on the same branch.

Fill every bracketed placeholder before dispatching. A spec with unfilled brackets means you don't have enough verdict + decomposition context yet — go back to step 3 (Investigate) or `task-decomposition.md`.

## Spec template

```markdown
# Issue #<num> — Task <i>/<N>: <task title>

You are an implementer subagent. The main agent has triaged issue #<num> as **real** and decomposed the fix into <N> task(s); this dispatch covers task <i> only. Your job is to land task <i> on the current branch (`fix/issue-<num>-<slug>`), verify it, and return a status to the main agent.

## Why this task exists

Verdict: REAL.
This is task <i> of <N>. The full task list for this issue is:

1. <task 1 title>  <— marker on the current task>
2. <task 2 title>
3. <task 3 title>

You are responsible only for task <i>. Do not touch behavior owned by other tasks; if you discover something adjacent that belongs to another task, note it in your return summary instead of fixing it.

## Issue context

URL: <issue-url>

Verbatim evidence block from the issue (do not paraphrase when reasoning):

```<language>
<evidence block from issue body, with line numbers>
```

## Files you MUST read before writing any code

1. `<file>` — the primary file this task changes. Read lines `<start>-<end>` and 40 lines of context on either side.
2. `<test-file>` — the test file that covers or should cover this behavior. Read it entirely.
3. `<related-file>` — callers / related constants / docs. Read as needed; at minimum skim.

Use ripgrep (`rg`) to find additional callers before modifying any function signature.

## Acceptance criteria for THIS task only

The task is done when ALL of the following are true:

- [ ] <concrete behavior change scoped to this task>
- [ ] <new or updated test that proves it>
- [ ] <verification command for this task — usually `make test` or a targeted subset>
- [ ] <any project-specific check this task triggers, e.g. `make lint` if the task changes Go code>

These are the criteria the per-task code reviewer will check against. They must be measurable in this task's diff alone.

## Hard constraints

- **Scope:** Change only what's needed to satisfy the criteria above. No drive-by refactors, no renaming unrelated symbols, no dependency bumps. If you see something adjacent that looks wrong, note it in your return summary — don't fix it in this diff.
- **Tests first when you can.** For any bug-fix task, write the failing test that captures the Evidence case, watch it fail, then implement the fix. Follow the repo's test conventions — if you see table-driven tests, write a table-driven test.
- **No new files unless the criteria require it.** Prefer editing existing files.
- **Honor repo conventions.** Read `CLAUDE.md` / `AGENTS.md` / language style skills if they exist. In particular for Go: `devpilot-google-go-style` rules apply.
- **Commit your work on the branch before returning.** Granular commits are fine. The per-task code reviewer reads the SHA range you produced.
- **Do not push, do not open a PR, do not comment on the issue.** That's the main agent's job after all tasks land.

## Return summary format

When you're done (or stuck), return to the main agent with one of four statuses (these are mandatory; pick exactly one):

- **DONE** — All acceptance criteria met, verification commands pass, ready for code review.
- **DONE_WITH_CONCERNS** — All criteria met but you have flagged doubts (correctness, scope, repo convention) the main agent should read before reviewing.
- **NEEDS_CONTEXT** — You cannot complete the task without information that wasn't provided. Ask one question, do not guess.
- **BLOCKED** — You cannot complete the task; the spec, the codebase, or the verification command is wrong. Explain what changed your mind.

```
## Task <i>/<N> status: <DONE | DONE_WITH_CONCERNS | NEEDS_CONTEXT | BLOCKED>

Branch: fix/issue-<num>-<slug>
Commits in this task: <N> commit(s)  (sha range: <base>..<head>)

### Changes
- <file>:<range> — <one-line what changed>
- <file>:<range> — <one-line what changed>

### Acceptance criteria
- [x] <item 1>
- [x] <item 2>
- [ ] <item 3 — if not met, explain>

### Verification output
<paste final output of each verification command, or the failure if stuck>

### Concerns / notes for the reviewer
<anything adjacent you noticed but did not fix; uncertainties; doubts that earned DONE_WITH_CONCERNS>

### Question (only if NEEDS_CONTEXT)
<one-line question — do not ask multiple at once>
```

If you returned NEEDS_CONTEXT or BLOCKED, leave the branch in whatever state you reached and do not try to "tidy up" with extra commits. The main agent decides whether to re-dispatch you, change the spec, or escalate.
```

## Rules for the main agent filling this template

- **One task per dispatch.** Even if two tasks look "small enough to combine", separate dispatches keep the per-task review surface small. The skill is built around one implementer + one reviewer per task.
- **Never run implementers in parallel on the same issue's branch.** They'd race on the working tree and produce a merge mess inside a single PR. Sequential only.
- **Do not omit the Evidence block.** Even if the main agent already reasoned about it — the subagent needs the verbatim quote, every dispatch.
- **Concrete acceptance criteria, not aspirations.** "Tests pass" is not a criterion; "`make test` exits 0" is. "Handles the edge case" is not a criterion; "`TestFooFn_EmptyInput` passes" is.
- **Name specific files.** Subagents that are told "figure out which files to read" waste a full context window exploring. Hand them the starting points from your investigation in step 3.
- **Project-specific verification commands win.** `make test` / `make lint` for this repo; other repos may use `go test ./...`, `pnpm test`, `cargo test`, etc. Read the repo's `CLAUDE.md` or `Makefile` and cite the exact commands.
- **Re-dispatch policy:**
  - **Verification failed** (commands didn't pass): one re-dispatch with the failing output appended. Second failure → escalate the whole issue to `NEEDS-HUMAN`.
  - **Code review found issues** (handled in `per-task-review.md`): re-dispatch the same implementer with the reviewer's feedback verbatim. Cap at 2 review rounds per task; round 3 → `NEEDS-HUMAN`.
  - **NEEDS_CONTEXT:** answer the question, re-dispatch with the answer added to the spec.
  - **BLOCKED:** read the explanation, decide between adjusting the spec, dispatching a more capable model, or escalating `NEEDS-HUMAN`. Never re-dispatch the same model with the same spec on a BLOCKED return.
