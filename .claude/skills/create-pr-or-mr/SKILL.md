---
name: create-pr-or-mr
description: >
  Create pull requests (GitHub) or merge requests (GitLab) with well-structured descriptions
  that reflect the actual changes. Automatically detects the hosting platform and uses `gh` or
  `glab` accordingly. Uses existing PR/MR templates when available, otherwise generates a
  structured description with only the sections relevant to the change.
  Use this skill whenever the user wants to create a pull request, merge request, open a PR,
  submit a MR, or push changes for review. Triggers on: "create pr", "open pull request",
  "make a pr", "submit mr", "merge request", "push for review", "ready for review",
  "/pr", "create pull request", "open mr", "创建PR", "提交合并请求".
---

# Pull Request / Merge Request Skill

Create high-quality pull requests or merge requests by analyzing the actual code changes and
generating descriptions that help reviewers understand what changed and why.

## Workflow

### Step 1: Detect Platform

Determine whether this is a GitHub or GitLab repository:

```bash
git remote get-url origin
```

- If the remote URL contains `github.com` -> use `gh`
- If the remote URL contains `gitlab.com` or a known GitLab instance -> use `glab`
- If ambiguous, check which CLI is available (`gh --version` / `glab --version`) and ask the user

### Step 2: Gather Context

Understand the full scope of changes before writing anything.

1. **Identify the base branch:**
   ```bash
   git rev-parse --abbrev-ref HEAD
   ```
   Determine the target branch (usually `main` or `master`). If the user specified a target, use that.

2. **Get the full diff and commit history from the base branch:**
   ```bash
   git log <base>..HEAD --oneline
   git diff <base>...HEAD
   ```
   Read ALL commits, not just the latest one. The PR description must reflect the entire branch.

3. **Check for untracked/uncommitted changes:**
   ```bash
   git status
   ```
   If there are uncommitted changes, let the user know and ask if they want to commit first.

4. **Understand the changes:**
   - What files were modified, added, or deleted
   - What the changes do functionally (bug fix? new feature? refactor?)
   - Whether changes touch frontend, backend, infrastructure, docs, or a mix
   - Whether there are test changes
   - Whether there are breaking changes

### Step 3: Find or Build the Template

Check for existing PR/MR templates in this order:

**GitHub:**
1. `.github/pull_request_template.md`
2. `.github/PULL_REQUEST_TEMPLATE.md`
3. `docs/pull_request_template.md`
4. `PULL_REQUEST_TEMPLATE.md`
5. Files inside `.github/PULL_REQUEST_TEMPLATE/` (if multiple, ask user which to use)

**GitLab:**
1. `.gitlab/merge_request_templates/Default.md`
2. Files inside `.gitlab/merge_request_templates/` (if multiple, ask user which to use)

If a template exists, use it as the structure and fill it in based on the changes.

If no template exists, use the default template below — but only include sections that are
relevant to this specific change.

### Default Template

```markdown
## Description

{Concise explanation of what this change does and why}

Closes: #{issue_number}

---

## Type of Change

{Only the applicable items, checked}

* [x] Bug fix
* [ ] New feature
* [ ] Refactor
* [ ] Performance improvement
* [ ] Documentation update
* [ ] Breaking change

---

## How Has This Been Tested?

{Specific steps to verify, or mention of automated tests added/passing}

---

## Screenshots / Demo (if applicable)

Before:

After:

---

## Checklist

* [x] I have performed a self-review of my code
* [x] My code follows the project's coding standards
* [x] I have added tests where necessary
* [x] All tests pass locally
* [x] Documentation has been updated if required
* [x] This PR is ready for review

---

## Additional Notes

{Context reviewers should know}
```

### Section Inclusion Rules

The point of this template is to help reviewers — not to create busywork. Only include sections
that carry useful information for the specific change:

| Section | Include when... | Omit when... |
|---------|----------------|--------------|
| Description | Always | Never omit |
| Closes: line | There's a related issue | No linked issue |
| Type of Change | Always (helps reviewers set expectations) | Never omit |
| How Has This Been Tested? | There are functional changes | Docs-only or trivial config changes |
| Screenshots / Demo | Changes affect UI, visual output, or CLI output | Backend-only, library, infra, docs changes |
| Checklist | Always (useful self-check) | Never omit |
| Additional Notes | There's migration steps, deployment notes, or context that doesn't fit elsewhere | Nothing extra to say |

When omitting a section, remove it entirely — don't leave empty headers or placeholder comments.

### Step 4: Draft the PR Description

Write the description by filling in the template based on your analysis from Step 2.

**Writing guidelines:**
- **Description**: Lead with what the change does, then why. Be specific — "Fix race condition
  in task runner polling loop" is better than "Fix bug". If the commits tell a clear story,
  summarize the narrative.
- **Type of Change**: Check the boxes that apply. If it's a mix, check multiple.
- **How Has This Been Tested?**: Mention specific test files added/modified, or manual steps.
  If CI covers it, say so.
- **Screenshots**: Only for visual changes. If included, describe what's shown.
- **Checklist**: Pre-check items you've verified. Leave unchecked items that the user should
  confirm.
- **Additional Notes**: Migration steps, feature flags, deploy dependencies, follow-up work.

### Step 5: Create the PR/MR

Present the draft title and description to the user for review before creating.

**GitHub:**
```bash
gh pr create --title "<title>" --body "<body>" [--base <target-branch>] [--draft]
```

**GitLab:**
```bash
glab mr create --title "<title>" --description "<body>" [--target-branch <target>] [--draft]
```

Use `--draft` if the user indicates the PR isn't ready for review yet.

If the branch hasn't been pushed yet, push it first:
```bash
git push -u origin HEAD
```

### Step 6: Report Back

After creating, share the PR/MR URL with the user.

## Tips

- Keep PR titles under 72 characters. Use imperative mood ("Add feature" not "Added feature").
- If the diff is very large, mention which files are most important to review first.
- For stacked PRs, mention the dependency chain in Additional Notes.
- Respect the user's language — if they write in Chinese, write the PR in Chinese (unless the
  project convention is English).
