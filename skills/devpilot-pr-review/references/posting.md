# Posting the Review

Default flow: show the user the drafted body and the inline comments, then post **everything in a single combined POST** to GitHub's review API. Body + inline comments + event in one call so they show up grouped under one review (not as standalone PR comments).

## GitHub — single combined POST

Build a JSON payload and pipe it to `gh api --input -`:

```bash
owner=...   # e.g. SiyuQian
repo=...    # e.g. devpilot
num=...     # PR number

# $body holds the rendered review body (template.md → "Review body template").
# $event is REQUEST_CHANGES | COMMENT | APPROVE (see mapping below).

jq -n \
  --arg event "$event" \
  --arg body  "$body" \
  --argjson comments "$comments_json" \
  '{event: $event, body: $body, comments: $comments}' \
| gh api -X POST "repos/$owner/$repo/pulls/$num/reviews" --input -
```

`$comments_json` is a JSON array, one object per inline finding. Build it with `jq` so quoting and newlines in markdown bodies are handled safely:

```bash
comments_json=$(jq -n '
  [
    { path: "internal/auth/client.go", line: 72, side: "RIGHT", body: $c1 },
    { path: "internal/auth/client.go", line: 68, side: "RIGHT", body: $c2 },
    { path: "internal/auth/client.go", line: 83, side: "RIGHT", body: $c3 }
  ]
' --arg c1 "$comment_blocking" \
  --arg c2 "$comment_should_fix" \
  --arg c3 "$comment_consider")
```

Each comment body uses the inline-comment template from `template.md`. Severity tag goes inside the comment text; the API does not have a per-comment severity field.

### Event mapping

`$event` is derived from the highest-severity inline finding:

| Highest severity | event |
|---|---|
| Any Blocking | `REQUEST_CHANGES` |
| Should-fix / Consider / Nit only | `COMMENT` |
| No findings | `APPROVE` |

### Anchor fields

- **`path`** — repo-relative path of the changed file.
- **`line`** — line in the file at the head SHA. Use the new line for added or changed lines.
- **`side`** — `RIGHT` for added / changed (default), `LEFT` for deleted lines (then `line` refers to the base file).
- **Multi-line comments** — add `start_line` (and `start_side` when commenting across both sides). `start_line` ≤ `line`.
- **Avoid `position`** — deprecated; `line` + `side` is the supported form.

### Resolving `owner`, `repo`, `num`

```bash
gh pr view "$url" --json url,number,baseRepository \
  -q '"\(.baseRepository.owner.login) \(.baseRepository.name) \(.number)"'
```

…or split the URL `https://github.com/<owner>/<repo>/pull/<num>`.

### Pre-post sanity check

Before the POST:

- Every inline comment's `(path, line)` exists in the diff at `head_sha` (`gh pr diff` output). Posting against a non-diff line returns 422.
- The combined body length is well under GitHub's review-body limit (65 KB). Trim if needed; per-finding detail lives inline anyway.
- The event matches the highest-severity finding (table above).

## GitLab — discussions API

GitLab merge requests use `glab api` for inline (positional) discussions and `glab mr note` for the body summary. There is no `request-changes` state; severity stays in the body.

```bash
# One request per inline finding
glab api -X POST "projects/:id/merge_requests/:iid/discussions" \
  -F "body=$comment_body" \
  -F "position[base_sha]=$BASE_SHA" \
  -F "position[head_sha]=$HEAD_SHA" \
  -F "position[start_sha]=$START_SHA" \
  -F "position[position_type]=text" \
  -F "position[new_path]=$path" \
  -F "position[new_line]=$line"

# Summary body
glab mr note <iid> --message "$body"
```

Resolve `BASE_SHA`, `HEAD_SHA`, `START_SHA` from `glab mr view --json diff_refs`.

## Skip posting and say so

Skip posting and tell the user explicitly that the review is local-only when any of these hold:

- The user opted out ("don't post", "dry run", "local only", "just draft").
- The review is on a patch pasted into chat with no real PR behind it.
- The PR is already merged or closed.

In any of those cases, render the body and the inline comments in chat (each comment prefixed with its `path:line` so the user can read it without the API anchor).

## Anti-shortcuts

- **Don't dump findings into the body** to avoid building the comments array. Findings tied to a line go inline.
- **Don't post inline comments via `gh pr comment`** — those are PR conversation comments, not review comments. They show up in a different pane and can't be resolved as part of a review.
- **Don't post inline comments outside a review** (`POST .../pulls/:num/comments` directly). Always route through `POST .../pulls/:num/reviews` so they land grouped under one review with the right event.
- **Don't split into multiple reviews** ("one for blockers, one for nits"). One review per pass; the author sees one notification, one set of comments, one verdict.
