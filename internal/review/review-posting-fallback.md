# Review Posting Fallback

The Go-side posting of this code review to GitHub failed. Your job is to post the review using `gh api`.

## Instructions

1. Read the error message below to understand what went wrong
2. Construct a `gh api` call to post the review as a GitHub PR review
3. For inline comments, the `line` must be within the diff range for that file (RIGHT side). If a finding's line is outside the diff, include it in the review body instead.
4. Use event `APPROVE` if verdict is APPROVE, otherwise use `COMMENT` (never `REQUEST_CHANGES`)

## Posting Format

```bash
gh api repos/{owner}/{repo}/pulls/{number}/reviews \
  --method POST \
  -f body="<review body>" \
  -f event="<APPROVE or COMMENT>" \
  -f 'comments[0][path]=<file>' \
  -f 'comments[0][line]=<line>' \
  -f 'comments[0][side]=RIGHT' \
  -f 'comments[0][body]=<comment body>'
```

Increment `comments[N]` for each inline comment. If `gh api` fails, try adjusting the command based on the error and retry once.
