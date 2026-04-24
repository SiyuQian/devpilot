# Subagent fix spec

This is the prompt you hand to `superpowers:subagent-driven-development` (via its dispatch mechanism) when a verdict is REAL. Fill every bracketed placeholder before dispatching. A spec with unfilled brackets means you don't have enough verdict context yet — go back to step 3 (Investigate).

## Spec template

```markdown
# Fix issue #<num>: <issue title>

You are a coding subagent. The main context has already triaged this issue as **real** and handed it to you for implementation. Your job is to land a focused fix on the current branch (`fix/issue-<num>-<slug>`), verify it, and return a summary to the main agent.

## Issue

URL: <issue-url>

Verbatim evidence block from the issue (do not paraphrase when reasoning):

```<language>
<evidence block from issue body, with line numbers>
```

## Files you MUST read before writing any code

1. `<file>` — the primary file to change. Read lines `<start>-<end>` and 40 lines of context on either side.
2. `<test-file>` — the test file that covers or should cover this behavior. Read it entirely.
3. `<related-file>` — callers / related constants / docs. Read as needed; at minimum skim.

Use ripgrep (`rg`) to find additional callers before modifying any function signature.

## Acceptance criteria

The fix is done when ALL of the following are true:

- [ ] <concrete behavior change — e.g., "passing empty string to `FooFn` returns `ErrEmptyInput` instead of panicking">
- [ ] <new test or regression test — e.g., "`TestFooFn_EmptyInput` passes, covers the case from the Evidence block">
- [ ] <verification command 1 passes — e.g., "`make test` exits 0">
- [ ] <verification command 2 passes — e.g., "`make lint` exits 0">
- [ ] <any additional project-specific check>

## Hard constraints

- **Scope:** Change only what's needed to satisfy the criteria above. No drive-by refactors, no renaming unrelated symbols, no dependency bumps. If you see something adjacent that looks wrong, note it in your return summary — don't fix it in this diff.
- **Tests first when you can.** For any bug, write the failing test that captures the Evidence case, watch it fail, then implement the fix. Follow the repo's test conventions — if you see table-driven tests, write a table-driven test.
- **No new files unless the criteria require it.** Prefer editing existing files.
- **Honor repo conventions.** Read `CLAUDE.md` / `AGENTS.md` / language style skills if they exist. In particular for Go: `devpilot-google-go-style` rules apply.
- **Do not push, do not open a PR, do not comment on the issue.** That's the main agent's job after verification.

## Return summary format

When you're done (or stuck), return to the main agent with:

```
## Fix summary for issue #<num>

Branch: fix/issue-<num>-<slug>
Commits: <N> commit(s)

### Changes
- <file>:<range> — <one-line what changed>
- <file>:<range> — <one-line what changed>

### Acceptance criteria
- [x] <item 1>
- [x] <item 2>
- [ ] <item 3 — if not met, explain>

### Verification output
<paste final output of each verification command, or the failure if stuck>

### Notes for the reviewer
<anything adjacent you noticed but did not fix, or uncertainties>
```

If you could not satisfy the criteria, return with as much as you got done and the verbatim failure output. The main agent will decide whether to retry you once or escalate to NEEDS-HUMAN.
```

## Rules for the main agent filling this template

- **Do not omit the Evidence block.** Even if the main agent already reasoned about it — the subagent needs the verbatim quote.
- **Concrete acceptance criteria, not aspirations.** "Tests pass" is not a criterion; "`make test` exits 0" is. "Handles the edge case" is not a criterion; "`TestFooFn_EmptyInput` passes" is.
- **Name specific files.** Subagents that are told "figure out which files to read" waste a full context window exploring. Hand them the starting points from your investigation in step 3.
- **Project-specific verification commands win.** `make test` / `make lint` for this repo; other repos may use `go test ./...`, `pnpm test`, `cargo test`, etc. Read the repo's `CLAUDE.md` or `Makefile` and cite the exact commands.
- **Cap retries at one.** If the first subagent returns with failing verification, re-dispatch ONCE with the failure output appended. If the second attempt still fails, escalate to NEEDS-HUMAN per step 7 of the skill.
