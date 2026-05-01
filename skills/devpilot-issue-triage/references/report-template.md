# Report Template

Use this exact skeleton for `./issue-triage-<owner>-<repo>-<YYYY-MM-DD>.md`. Section order is fixed. Buckets with zero issues should still appear with "_None._" — empty buckets are signal.

```markdown
# Issue Triage — <owner>/<repo>

Date: <YYYY-MM-DD>
Open issues triaged: <N>
Filter applied: <none | label=foo | author=bar | …>

## Summary

| Bucket | Count |
|---|---|
| ready-to-fix | <n> |
| needs-info | <n> |
| needs-design | <n> |
| duplicate | <n> |
| stale | <n> |
| out-of-scope | <n> |

## ready-to-fix

### #<num> — <title>

- **Evidence:** <one line>
- **Suspected root cause:** <1–2 sentences from deep-dive>
- **Size:** XS | S | M | L
- **Suggested labels:** `triage:ready`, `<other>`
- **Handoff hint:** <one line for devpilot-resolve-issues>

(repeat per ready-to-fix issue, in ascending issue-number order)

## needs-info

### #<num> — <title>

- **Evidence:** <missing pieces, one line>
- **Suggested labels:** `triage:needs-info`, `<other>`
- **Drafted comment** (do not post):

  > <paste-ready comment from references/draft-comments.md>

(repeat)

## needs-design

### #<num> — <title>

- **Evidence:** <one line>
- **Suggested labels:** `triage:needs-design`, `<other>`
- **Drafted comment** (do not post):

  > <paste-ready comment>

(repeat)

## duplicate

### #<num> — <title>

- **Duplicates:** #<other-num>
- **Evidence:** <one-line shared root cause>
- **Suggested labels:** `triage:duplicate`
- **Drafted close-comment** (do not post):

  > <paste-ready close-comment>

(repeat)

## stale

### #<num> — <title>

- **Evidence:** <specific staleness reason>
- **Suggested labels:** `triage:stale`
- **Drafted close-comment** (do not post):

  > <paste-ready close-comment>

(repeat)

## out-of-scope

### #<num> — <title>

- **Belongs in:** <repo or product>
- **Suggested labels:** `triage:out-of-scope`
- **Drafted close-comment** (do not post):

  > <paste-ready close-comment>

(repeat)

## Next step

Hand the `ready-to-fix` list to `devpilot-resolve-issues`:

```
gh issue list --repo <owner>/<repo> --state open --label triage:ready --json number
```

Or, after applying the suggested labels manually, run `devpilot-resolve-issues` filtered to `label:triage:ready`.

_This report did not modify any GitHub state._
```

## Rules

- Section order is fixed: ready-to-fix → needs-info → needs-design → duplicate → stale → out-of-scope → Next step.
- Within each bucket, sort by ascending issue number.
- Do **not** add an "Order of Attack", "Recommended PR groupings", or any prioritization section. Out of scope.
- The trailing "_This report did not modify any GitHub state._" line is mandatory — it's how the user (and you) confirm the read-only contract held.
