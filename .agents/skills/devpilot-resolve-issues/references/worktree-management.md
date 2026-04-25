# Worktree Management

Every REAL issue gets its own `git worktree`. The main checkout never sees the fix branch — that is the whole point. This file is the operational reference: path scheme, create / remove commands, preflight pruning, failure modes.

**Scope:** This file is loaded at step 0 (preflight pruning) and step 5 (per-issue create) of `SKILL.md`, and again on the cleanup path after step 8 / on escalation. It is not loaded by implementer or reviewer subagents — they inherit cwd from the controller and don't run worktree commands themselves.

## Path scheme

```
<repo-root>.worktrees/issue-<num>-<slug>/
```

Concretely, if the main checkout is `/Users/me/Works/foo` and the issue is `#42` with slug `sanitize-shell-input`:

```
/Users/me/Works/foo                                  <- main checkout (default branch, never the fix branch)
/Users/me/Works/foo.worktrees/issue-42-sanitize-shell-input  <- the per-issue worktree
```

Why this layout:

- **Sibling of repo, hidden by `.worktrees` suffix.** Discoverable when listing the parent directory; clearly transient by name.
- **One parent dir per repo.** `rm -rf <repo>.worktrees` (or `git worktree prune`) cleans every dangling worktree for that repo at once.
- **No collision with other repos.** The dir name embeds the repo name, so multiple repos under the same parent each get their own bucket.

`<slug>` is the same kebab-case slug used for the branch name (3–5 ASCII words, derived from the issue title). The branch in the worktree is `fix/issue-<num>-<slug>` — same naming you would have used pre-worktree.

**Do not put the worktree inside the main checkout.** `<repo-root>/.worktrees/issue-N` looks tidier but breaks: the main checkout's tooling (linters, test runners, IDEs, file watchers) will recurse into it and confuse itself, and `git status` from the main checkout will surface it as untracked. Sibling-only.

## Step 0 — preflight pruning

Run once per loop, before selecting the first issue. Three things, in order:

```bash
# 1. Confirm git is new enough to support worktrees (it always is on supported platforms).
git worktree --help >/dev/null 2>&1 || { echo "git worktree unavailable"; exit 1; }

# 2. Prune metadata for worktrees whose directory was deleted out from under git.
git worktree prune

# 3. List remaining worktrees and decide what to do with stragglers.
git worktree list
```

If `git worktree list` shows worktrees from a prior `/resolve-issues` run that did NOT clean up:

- **Their branch is already pushed and the PR is open / merged** — safe to remove. `git worktree remove <path>` (add `--force` only if uncommitted changes exist; investigate first).
- **Their branch has unpushed commits** — STOP. Ask the user. This is somebody's in-progress work; the safe default is leave it alone and pick a different issue, not blow it away.

Never remove a worktree without first inspecting its branch. The whole point of worktrees is that they keep work isolated, so a stale worktree is more often "in-progress work nobody has come back to" than "leftover garbage."

## Step 5 — create the worktree for one issue

Pre-conditions you should have already established:

- Verdict on `#<num>` is REAL (steps 3–4 of `SKILL.md`).
- Default branch name is in `$DEFAULT_BRANCH` (from preflight).
- Slug is in `$SLUG` (kebab-case, 3–5 words).

```bash
# Refresh the default branch ref so the worktree branches off the latest commit.
git fetch origin "$DEFAULT_BRANCH"

WORKTREE_PATH="$(git rev-parse --show-toplevel).worktrees/issue-${NUM}-${SLUG}"
BRANCH="fix/issue-${NUM}-${SLUG}"

# Create the worktree on a fresh branch off origin/<default>.
git worktree add -b "$BRANCH" "$WORKTREE_PATH" "origin/${DEFAULT_BRANCH}"

# Move the controller into the worktree. Subagents dispatched from here will inherit this cwd.
cd "$WORKTREE_PATH"
```

After this point, every subsequent step (6 implementers, 6c review, 7 verify, 8 PR creation, 9 issue comment) runs with `cwd = $WORKTREE_PATH`. The main checkout is untouched until cleanup.

### Failure modes at create time

| Failure | What to do |
|---|---|
| `fatal: '<path>' already exists` | A previous run for this issue did not clean up. Run `git worktree list` to confirm; if the branch is unpushed, treat as in-progress (ask the user). If the branch matches `fix/issue-<num>-<slug>` and has been pushed, remove it (`git worktree remove "$WORKTREE_PATH"`) and retry. |
| `fatal: '<branch>' is already checked out at '<other-path>'` | The branch is live in another worktree. Same diagnosis: somebody (you in a prior run, or another agent) is mid-fix. Stop and ask. |
| `<branch>` already exists locally | Either reuse it (`git worktree add "$WORKTREE_PATH" "$BRANCH"` — no `-b`) if it was a clean prior attempt, or rename your slug. Default to renaming. |
| `error: invalid reference: origin/<default>` | `git fetch origin` first; then retry. If the fetch fails, that's a network / auth issue, escalate to the user before continuing the loop. |
| Disk full / parent dir not writable | Stop the loop. The user has to fix the filesystem before any issue can be processed. |

Never `--force` your way past these. A forced worktree create on top of in-progress work is exactly the kind of "hard-to-reverse" action that belongs in a confirmation prompt, not in an autonomous loop.

## Cleanup — after PR creation, after escalation, or after FALSE-POSITIVE retry-with-worktree

The controller is currently at `cwd = $WORKTREE_PATH`. Get back to the main checkout and remove the worktree:

```bash
MAIN="$(git rev-parse --path-format=absolute --git-common-dir)"
MAIN="$(dirname "$MAIN")"   # .git/common-dir → repo root of the main checkout
cd "$MAIN"

git worktree remove "$WORKTREE_PATH"
```

**Cleanup policy — try in this order:**

1. **Plain remove first:**

   ```bash
   git worktree remove "$WORKTREE_PATH"
   ```

   Often fails on Go / Node / Rust repos with "contains modified or untracked files" because `bin/` or `node_modules/` or `target/` got built inside the worktree.

2. **If plain remove failed because of *gitignored* artifacts** (build outputs, dep caches): drop the ignored files first, then re-run the plain remove. This avoids `--force` entirely and is the preferred path:

   ```bash
   git -C "$WORKTREE_PATH" clean -fdX     # -X = ignored files only; -fd = force, dirs
   git worktree remove "$WORKTREE_PATH"   # should succeed now
   ```

3. **`--force` is the last resort, only when both of these are true:**
   - You have **verified** with `git status` and `git log origin/$BRANCH..HEAD` that all real work is on origin (or that you're intentionally discarding it).
   - `clean -fdX` didn't help — i.e., the dirty files aren't gitignored, they're untracked but not ignored.

   Then: `git worktree remove --force "$WORKTREE_PATH"`. Reach for this only after step 2 didn't work.

**Common cases:**

- After `devpilot-pr-creator` succeeded → the branch is on origin → step 2 (`clean -fdX` + plain remove) handles ~all build-artifact cases. Skip straight to `--force` only if `clean -fdX` left modified-but-not-ignored files behind, which suggests something tracked got modified — investigate before forcing.
- After escalation pushed a draft branch → same: try step 2 first.
- Implementer crashed mid-task and the worktree has uncommitted changes to **tracked** files → STOP. Do not `--force`. Ask the user. Those changes are not on origin.

After remove, run `git worktree prune` once more to clean the metadata side and confirm nothing is left:

```bash
git worktree prune
git worktree list   # should not show the issue worktree anymore
```

### Cleanup branches

Cleanup is required at exactly three points in the loop:

1. **PR opened cleanly (step 8 success)** — branch is pushed, PR is up; remove the worktree, the branch lives on origin.
2. **Escalation mid-fix (BLOCKED, round-3 review, second verification fail)** — push the branch as a draft (`git push -u origin "$BRANCH"`), comment on the issue with the branch URL, *then* remove the worktree. The escalation is documented on the issue; the work-so-far is preserved on origin.
3. **Final verify failed at step 7** — same as escalation: push as draft, comment, then remove.

You never reach cleanup on the FALSE-POSITIVE or NEEDS-HUMAN paths because **no worktree was ever created** for those — the verdict happens at step 4, before step 5.

## Why this matters (and what it forbids)

- **The main checkout is sacred.** While the loop runs, the user's `cd` into the repo still shows the default branch with no surprise edits. They can keep working — open editors, run dev servers — without colliding with the loop.
- **Multiple issues in flight is now physically possible.** Each issue lives at a separate path on disk, on a separate branch, with separate `make` build outputs. The current `SKILL.md` still keeps issues sequential, but the constraint that previously *forced* sequence (one branch checked out at a time) is gone — when we relax sequencing in the future, the worktree layout is what makes it safe.
- **One implementer per task on a given branch is still required.** Worktrees buy you isolation between *issues*, not between tasks of the same issue. Two implementers in the same worktree race on the same working tree.

## Anti-shortcuts

- **"Just branch in the main checkout for this one — worktree feels like overhead."** Worktree creation is two commands and ~50ms. The "overhead" is the user being able to keep working in the main checkout. Skip it once and the loop forgets it forever.
- **"`git stash` before switching branches is the same thing."** It is not. Stash leaves the main checkout's HEAD pointing at the fix branch with an unrelated stash on top — exactly the polluted state worktrees exist to avoid.
- **"I'll create the worktree but stay in the main checkout's cwd and pass paths."** Then every shell invocation needs `-C "$WORKTREE_PATH"`, every subagent gets a different cwd than git expects, and the implementer trips on its own paths. `cd` into the worktree once at step 5 and stay there until cleanup.
- **"`git worktree remove --force` is fine, the work is on origin."** Only if you actually pushed it. Verify with `git status` and `git log origin/<branch>..HEAD` first; "I think I pushed" is not verified.
- **"Cleanup can wait until the end of the loop, batch it."** No. Each iteration cleans up its own worktree. A loop that processes 10 issues should never have 10 worktrees alive simultaneously — that is unprocessed cleanup, not parallelism.
