# Rejected & Deferred Ideas

This directory tracks ideas that were evaluated by the PM skill and rejected or deferred by the user.

- `rejected` — Permanently rejected. PM skill will not re-recommend similar ideas.
- `deferred` — Not right now. PM skill may re-suggest if market conditions change significantly.

## File Format

Each file uses YAML frontmatter:

```yaml
---
status: rejected | deferred
idea: "Feature Name"
date: YYYY-MM-DD
score: N            # PM skill total score (x/12)
reason: scope | direction | timing | duplicate | other
---
```

Followed by a markdown body explaining the rejection reason and summarizing the original evidence.
