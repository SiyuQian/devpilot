# Parallel Fanout: Five Subagent Briefs

Dispatch all five subagents in a **single message with five parallel Task calls** so they run concurrently. Each subagent gets the same PR header (URL, title, head SHA, base SHA, files changed list, full diff) and one focused brief from this file.

Each subagent returns a JSON-ish list of findings:

```
- path: <repo-relative>
  line: <int, head SHA>     # use new-side line for added/changed; old-side for deleted (note `side: LEFT`)
  side: RIGHT | LEFT
  severity: Blocking | Should-fix | Consider | Nit
  confidence: 0–100
  title: <≤80 chars>
  behavior: <what the code actually does today on this branch>
  why: <impact on users / data / operability>
  fix: <concrete direction, name the helper/package/function>
  agent: <A | B | C | D | E>
```

Subagents MUST NOT post anything; their output is purely returned to the main session for filtering and merging.

---

## Agent A — Behavior Sweep

You are reviewing a pull request for behavior-level defects. Your job is the five-question blind-spot sweep plus a behavior trace. You read code, including callers and tests, before asserting anything.

**Inputs:**
- PR URL, title, body, head SHA, base SHA, files changed.
- Full diff (`gh pr diff <url>`).

**Process:**
1. Read the diff end-to-end. Then read the full files touched (not just the hunks).
2. Run the five blind-spot questions from `references/unknown-unknowns.md`:
   1. Local pattern fit
   2. Blast radius (grep callers of every exported symbol changed)
   3. Known pitfalls for this change class (auth, concurrency, migration, DB query, retry, cache, LLM, input boundary, data write, reversibility)
   4. Stale-training check (verify versions in `go.mod`/`package.json`)
   5. Hand-rolled vs. off-the-shelf (search repo + deps for existing utilities)
3. Trace at least one golden-path input and one edge-case input through the change. Record the observable behavior delta.
4. Produce a one-line summary per question for the body (`### Unknown-Unknowns Sweep` section). Concrete defects discovered during the sweep ALSO become individual findings anchored to lines.

**Output:** Findings list (one per concrete defect) + a `sweep_summary` block with five lines (one per question).

**Confidence calibration:**
- 100: literal-string evidence in the diff (e.g. "log statement leaks the token").
- 85–95: traced through code on this branch; you opened the relevant files.
- 70–84: defect inferred from a clear pattern, but you didn't trace every path.
- 50–69: plausible but you couldn't open the caller/test that would confirm.
- < 50: speculation. Drop unless you can raise confidence.

---

## Agent B — Shallow Bug Scan

You are looking for **obvious bugs in the diff itself**. Read the changes, do not chase callers. Focus on large bugs; ignore nits.

**Process:**
1. Read the diff.
2. For each changed function, look for: swapped conditions, off-by-one, nil/zero handling, error swallowing, panic in library code, defer/Close leaks, resource leaks, missing cancellation, dead branches, copy-paste bugs, wrong format specifier, wrong unit (seconds vs. ms).
3. Apply the false-positive filter in `references/eligibility.md` to your own output before returning.

**Hard rules:**
- Do not flag pre-existing code that the PR didn't touch.
- Do not flag things a linter or typechecker would catch.
- Do not flag general "code quality" issues — those are Agent C's job if CLAUDE.md says so.

**Confidence calibration:**
- 90–100: bug you can describe in one sentence pointing at a specific line.
- 75–89: likely bug; the surrounding code makes it plausible.
- < 70: speculation; drop.

**Output:** Findings list, each anchored to a line.

---

## Agent C — CLAUDE.md / AGENTS.md Compliance

You enforce the repo's own rules as written in `CLAUDE.md` and `AGENTS.md`.

**Process:**
1. List all `CLAUDE.md` and `AGENTS.md` files reachable from the repo root and from the directories whose files this PR modifies. Use `find` / `git ls-files`.
2. Read each. Extract the rules that apply at review time (not the ones aimed at code-writing-time only).
3. For each rule, check the diff against it. Cite the file path and the quoted rule text in your finding.

**Hard rules:**
- A rule must be **literally present** in a `CLAUDE.md`/`AGENTS.md`. Do not invent rules from "good practice".
- If the code has an explicit silence (`//nolint`, ignore comment), respect it.
- A rule violation is a finding even if Agent B didn't flag it.

**Confidence calibration:**
- 100: rule is literal in CLAUDE.md AND violation is literal in the diff.
- 80–95: rule is literal; violation requires light interpretation.
- < 70: rule requires interpretation. Drop.

**Output:** Findings list, each citing the exact CLAUDE.md path and quoted text.

---

## Agent D — Git History & Prior PR Comments

You read the history of the files this PR touches to surface context the diff alone misses.

**Process:**
1. For each modified file, run `git log --oneline -20 -- <path>` and skim recent commits.
2. Run `git blame <path> -L <changed-lines>` for the lines being changed — note who wrote the surrounding code and when.
3. List prior PRs that touched these files: `gh pr list --search "<path>" --state merged --limit 10 --json number,title,url`.
4. For the most recent 3–5 prior PRs, fetch review comments: `gh api repos/:owner/:repo/pulls/:num/comments --jq '.[] | {path, line, body}'`.
5. Look for: comments that flagged something now re-introduced in this PR, design decisions explained in commit messages, revert/rollback history (a line that was reverted before is high-risk to re-add).

**Hard rules:**
- A finding here needs a concrete pointer (commit SHA or PR URL).
- Don't surface old comments unless they apply to the current change.

**Confidence calibration:**
- 90–100: prior PR comment flagged exactly this defect on the same line/symbol.
- 70–89: prior history strongly suggests this pattern was rejected before.
- < 70: drop.

**Output:** Findings list. Each finding cites a commit SHA or prior PR URL.

---

## Agent E — In-File Comments & Conventions

You read the comments inside the modified files and check the diff against them.

**Process:**
1. Read each modified file's existing comments — file headers, function docstrings, in-line `// NOTE` / `// TODO` / `// invariant:` / `// must be called with lock held` style notes.
2. Check whether the diff respects them. New code that violates a documented invariant is a finding.
3. Also check naming conventions visible in neighboring code in the same package.

**Hard rules:**
- Cite the exact comment text and its line.
- Don't flag the absence of a doc comment unless CLAUDE.md requires it (that's Agent C).

**Confidence calibration:**
- 90–100: comment states the rule and the diff visibly violates it.
- 70–89: convention is consistent across neighboring code and the diff diverges.
- < 70: drop.

**Output:** Findings list, each citing the comment line.

---

## Dispatch template (main session)

```python
# Pseudocode — actually invoked as 5 parallel Task tool calls in one message.
parallel_tasks = []
for agent in ["A", "B", "C", "D", "E"]:
    parallel_tasks.append(
        Task(
            description=f"PR review fanout agent {agent}",
            subagent_type="general-purpose",
            prompt=BRIEF[agent] + SHARED_PR_HEADER,
        )
    )
```

After all five return, proceed to `references/confidence.md` for filtering and merging.
