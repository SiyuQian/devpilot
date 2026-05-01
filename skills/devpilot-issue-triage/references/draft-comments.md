# Draft Comments

These are paste-ready comment templates. Fill in the bracketed placeholders. Each draft goes into the report under the issue's section — never posted directly.

## `needs-info`

```
Thanks for the report. To make this actionable, could you share:

- [Specific question 1, e.g. exact steps that reproduce the issue]
- [Specific question 2, e.g. browser / Node version / OS]
- [Specific question 3, e.g. expected vs actual behavior]

Without [the most critical missing piece], we can't reliably reproduce or scope a fix.
```

Rule: ask for the **specific** missing pieces, not a generic "please provide more info". Maximum 3 questions — pick the ones that unblock the most.

## `needs-design`

```
This is a real issue, but the fix requires a decision before we write code:

- [Trade-off 1, e.g. "do we change the public response shape (breaking) or add a new field (clutter)"]
- [Trade-off 2, e.g. "is performance or correctness the priority for this path"]

Tagging for design discussion. Once we pick a direction, this can move to `ready-to-fix`.
```

Rule: list the trade-offs as concrete either/or choices, not vague "we should think about this".

## `duplicate`

```
Closing as duplicate of #[N] (same root cause: [one-line reason]). Please follow that issue for updates.
```

Rule: always cite the canonical issue number. The 1-line reason proves it's actually the same root cause, not just similar symptoms.

## `stale`

```
Closing as stale: [specific reason — last activity 18 months ago / references the removed `XYZ` module / superseded by the rewrite in #N].

If this is still affecting you on the current version, please open a new issue with a fresh repro.
```

Rule: name the specific reason. "Stale because old" isn't a reason; "stale because the file it references was deleted in #N" is.

## `out-of-scope`

```
This belongs in [other-repo / other-product] rather than here. Closing — please re-file at [link or repo name] with the same details.
```

Rule: name the right destination. If you don't know it, demote the bucket to `needs-design` instead.

## `ready-to-fix` (handoff hint, not a comment)

This isn't a comment to post — it's a 1-line hint that goes into the report so `devpilot-resolve-issues` (or a human) has a head start:

```
Handoff: [size XS/S/M/L] — [root cause in one sentence] — [primary file(s) to edit].
```

Example: `Handoff: S — catch block returns undefined; client crashes on missing summary — src/app/api/quotation/calculate/route.ts.`
