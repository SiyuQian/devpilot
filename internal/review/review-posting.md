# Posting Review to GitHub

After completing your review, post the results directly to the pull request as a GitHub PR review using the `gh api` command.

## Steps

1. **Get PR author**: Run `gh pr view <pr-url> --json author --jq .author.login` to get the author's GitHub username.

2. **Construct the review body** using this template:

   ```
   Nice work on this PR, @{author}! Here are some thoughts from my review.

   {summary}

   **Verdict: {APPROVED or NEEDS ATTENTION}**

   {overall assessment}

   — Automated review by DevPilot
   ```

   - Use `APPROVED` when your verdict is APPROVE.
   - Use `NEEDS ATTENTION` when your verdict is REQUEST_CHANGES.

3. **Determine the GitHub event**:
   - If verdict is APPROVE → use event `APPROVE`
   - If verdict is REQUEST_CHANGES → use event `COMMENT` (do NOT use `REQUEST_CHANGES`)

4. **Build inline comments** for each finding:
   - Each finding becomes an inline comment with:
     - `path`: the file path relative to the repo root
     - `line`: the line number in the file (new version / RIGHT side of diff)
     - `side`: always `RIGHT`
     - `body`: formatted as `[SEVERITY] Title\n\nExplanation\n\n```suggestion\ncode fix\n```\n` (include the suggestion block only if you have a concrete fix)
   - **Important**: The `line` must be within the diff range for that file. If a finding is on a line outside the diff, include it in the review body instead of as an inline comment.

5. **Post the review** using `gh api`:

   ```bash
   gh api repos/{owner}/{repo}/pulls/{number}/reviews \
     --method POST \
     -f body="<review body>" \
     -f event="<APPROVE or COMMENT>" \
     -f 'comments[0][path]=<file>' \
     -f 'comments[0][line]=<line number>' \
     -f 'comments[0][side]=RIGHT' \
     -f 'comments[0][body]=<comment body>'
   ```

   Repeat the `comments[N]` fields for each inline comment (increment N).

6. **Handle errors**: If the `gh api` call fails, report the error clearly in your output. The review text has already been streamed to the terminal, so a posting failure does not lose the review.

## Inline Comment Format

```
[WARNING] Missing error check

`StreamEvents` returns an error but it's being silently discarded here. Consider logging or propagating.

```suggestion
if err := s.StreamEvents(ctx); err != nil {
    return fmt.Errorf("stream events: %w", err)
}
```
```

## Notes

- Always use the `line`/`side` parameters, NOT the legacy `position` parameter.
- The `line` value is the file line number on the RIGHT (new) side of the diff.
- Verify each finding's line is within the diff range before adding it as an inline comment.
- If no findings require inline comments, submit the review with just the body (no `comments` fields).
