# Posting the Review

## Default flow

Show the user the drafted review, then post.

**GitHub** — post as a PR review so the review state is visible to GitHub:

```bash
printf '%s' "$review" | gh pr review <url> <mode> --body-file -
```

`<mode>` is derived from the highest-severity finding:

| Highest severity | Mode |
|---|---|
| Any Blocking | `--request-changes` |
| Should-fix / Consider / Nit only | `--comment` |
| No findings | `--approve` |

Pipe the body via stdin (`--body-file -`) to avoid shell-quoting issues with the review's markdown.

**GitLab** — `glab mr note <id> --message "$review"`. GitLab has no request-changes state; severity lives inside the body.

## Skip posting and say so

Skip posting, and tell the user explicitly that the review is local-only, when any of these hold:

- The user opted out ("don't post", "dry run", "local only", "just draft").
- The review is on a patch pasted into chat with no real PR behind it.
- The PR is already merged or closed.

## Inline comments (opt-in)

For line-level feedback, use the GitHub review API directly:

```bash
gh api -X POST /repos/{owner}/{repo}/pulls/{num}/reviews \
  -f event=<event> \
  -f body="$review_summary" \
  -F 'comments[]={"path":"...","line":42,"side":"RIGHT","body":"..."}'
```

Use this only when the user asks for inline comments. Default to the summary review above.
