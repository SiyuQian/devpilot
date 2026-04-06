---
name: pr-creator
description: >
  Use when the user wants to create or update a pull request or merge request,
  open a PR/MR, push changes for review, update a PR description, or mark a draft
  as ready. Triggers on: "create pr", "open pull request", "make a pr", "submit mr",
  "merge request", "push for review", "ready for review", "/pr", "open mr", "ship it",
  "send for review", "update the pr", "update the description", "mark as ready".
license: Complete terms in LICENSE.txt
---

# Pull Request / Merge Request Skill

**Core principle:** Read the actual diff before writing anything. Every sentence in the description
must come from code you read, not from branch names or assumptions.

## Quick Reference

| Action | GitHub | GitLab |
|--------|--------|--------|
| Create | `gh pr create --title "..." --body "..."` | `glab mr create --title "..." --description "..."` |
| Update | `gh pr edit <number> --title "..." --body "..."` | `glab mr update <number> --title "..." --description "..."` |
| Mark ready | `gh pr ready <number>` | `glab mr update <number> --draft=false` |
| Base branch | `--base <branch>` | `--target-branch <branch>` |
| Draft | `--draft` | `--draft` |
| Push | `git push -u origin HEAD` | `git push -u origin HEAD` |

## Preflight Checks

Before anything else, run these in parallel:

```bash
git remote get-url origin          # detect platform (github.com → gh, gitlab.com → glab)
git rev-parse --abbrev-ref HEAD    # current branch
git status                         # uncommitted changes?
git log <base>..HEAD --oneline     # commits ahead of base
gh pr list --head <branch>         # existing PR? (or glab mr list --source-branch)
```

**Stop and ask the user if:**
- Current branch is `main`/`master` — you cannot PR from main into main. Ask if they want to create a feature branch first. **Never** use `git reset`, `git push --force`, or any history rewriting on main — not even as an "option." If the work is already pushed to main, tell the user the PR opportunity has passed for this change.
- There are uncommitted changes — ask if they want to commit first or exclude them.
- There are no commits ahead of base — nothing to PR.
- There's already an open PR for this branch — switch to **update flow** (see below).

## Read the Diff (Required)

```bash
git diff <base>...HEAD             # full diff
git log <base>..HEAD --oneline     # commit history
```

**You MUST read and understand the actual changes before writing the description.**

From the diff, determine:
- What files were added, modified, or deleted
- What the changes do functionally (bug fix? feature? refactor?)
- Whether there are test changes
- Whether there are breaking changes

**Red flags — you're writing a bad description if:**
- You're inferring what changed from the branch name instead of the diff
- You're using generic phrases like "implements skill definition and supporting logic"
- Your test plan mentions things not visible in the diff
- Your description would be the same for a completely different set of changes

## Find the Template

Check for existing project templates first:

**GitHub:** `.github/pull_request_template.md`, `.github/PULL_REQUEST_TEMPLATE.md`, `docs/pull_request_template.md`, `PULL_REQUEST_TEMPLATE.md`, or files in `.github/PULL_REQUEST_TEMPLATE/`

**GitLab:** `.gitlab/merge_request_templates/Default.md` or files in `.gitlab/merge_request_templates/`

If a project template exists, use it. Fill it in based on the diff. Remove irrelevant sections entirely.

If no project template exists, **read ONE template** based on what you found in the diff:

| Changes touch frontend? | Bug fix or feature? | Read this file |
|---|---|---|
| Yes | Feature | [templates/frontend-feature.md](templates/frontend-feature.md) |
| Yes | Bug fix | [templates/frontend-bugfix.md](templates/frontend-bugfix.md) |
| No | Feature | [templates/backend-feature.md](templates/backend-feature.md) |
| No | Bug fix | [templates/backend-bugfix.md](templates/backend-bugfix.md) |
| Both | Either | Read both relevant templates, combine sections |
| Docs/config only | — | Use just Description + Review Guide, no template needed |

**Only read the template you need.** Do not read all four.

## Write the Description

**Title:** Under 72 characters. Use conventional commit prefix (`feat:`, `fix:`, `refactor:`, `docs:`, `chore:`). Be specific — "fix: auth redirect loop in OAuth callback" not "fix: bug".

**Review Guide (required):** Every PR must include a Review Guide section:
- Which file to start reviewing first
- Suggested review order for large diffs
- Any tricky logic or non-obvious decisions to watch for

**Body guidelines:**
- Lead with what changed and why, based on the actual diff
- Reference specific files, functions, or behaviors you saw in the code
- Run the project's lint/test commands and report results in the checklist
- Leave "For Reviewers (human)" items unchecked — those are for humans
- For bug fixes, describe the bug, root cause, and fix separately

**Show the draft to the user before creating or updating.** Let them edit the title and body.

## Create and Report

Push the branch if needed, create the PR/MR, and share the URL.

```bash
git push -u origin HEAD                    # if not already pushed
gh pr create --title "..." --body "..."    # GitHub
glab mr create --title "..." --description "..."  # GitLab
```

## Update an Existing PR/MR

When preflight finds an open PR for the current branch, or the user explicitly asks to update:

**1. Read the current PR state:**
```bash
gh pr view <number>                # GitHub — see current title, body, status
glab mr view <number>              # GitLab
```

**2. Read the full diff** (same as creating — the description must reflect ALL commits, not just new ones).

**3. Decide: preserve or rewrite.**
- **New commits added** to an existing PR → rewrite the full description to cover all commits. Don't just append — re-read the complete diff and write a cohesive description.
- **Reviewer feedback** on description → address the specific feedback while keeping the rest.
- **Draft → Ready** → update description if it was a WIP placeholder, then mark ready.

**4. Show draft to user, then execute:**
```bash
gh pr edit <number> --title "..." --body "..."    # GitHub
glab mr update <number> --title "..." --description "..."  # GitLab
```

**5. Mark ready (if transitioning from draft):**
```bash
gh pr ready <number>               # GitHub
glab mr update <number> --draft=false  # GitLab
```

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Description based on branch name, not diff | Read `git diff` first. Always. |
| Generic test plan ("verify no regressions") | Reference specific test files or manual steps from the diff |
| Auto-checking all checklist items | Run lint/test commands for verifiable items; leave human items unchecked |
| No Review Guide section | Always tell reviewers where to start and what to watch for |
| Generic bug fix description ("fix bug") | Describe the bug symptom, root cause, and fix separately |
| Leaving irrelevant sections as "N/A" | Remove the section entirely |
| Force-pushing main to create a retroactive branch | Never offer this as an option. If work is on main and pushed, the PR opportunity has passed |
| Creating duplicate PR for branch with existing PR | Check `gh pr list --head <branch>` first — update instead |
| Appending to PR description instead of rewriting | Re-read the full diff and write a cohesive description covering all commits |

## Tips

- For stacked PRs, mention the dependency chain.
- If the diff is large, highlight the most important files for reviewers.
- Respect the user's language — if they write in Chinese, write the PR in Chinese (unless the project convention is English).
