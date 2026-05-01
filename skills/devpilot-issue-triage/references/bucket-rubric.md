# Bucket Rubric

Use this when you're torn between two buckets. Pick the **first** matching rule.

## Decision order

1. **Is there an exact-match open or closed issue with the same root cause and files?**
   → `duplicate`. Cite the canonical issue number.

2. **Is the issue about something this repo doesn't own (different product, different service, vendor bug)?**
   → `out-of-scope`.

3. **Last activity > 12 months AND no recent comments AND references code that no longer exists?**
   → `stale`.

4. **Reporter didn't say what they did, what they expected, or what they got — AND the body has no cited file/line/log?**
   → `needs-info`. Even if you can guess, you'd be guessing.

5. **There's a real bug or missing feature, but the fix requires picking between ≥ 2 reasonable approaches, or changes a public API, or touches > 5 files?**
   → `needs-design`.

6. **Otherwise** — issue cites code, has a proposed fix or obvious one, fits in a single PR:
   → `ready-to-fix`.

## Counter-examples (common mistakes)

- "Issue from a scanner with severity:high" — does **not** automatically mean `ready-to-fix`. Severity is about impact, not scope. Apply rule 5 first.
- "Old issue but reporter is still active" — not `stale`. Stale means the *issue* has rotted, not the reporter.
- "Suggested fix is in the issue body" — doesn't override rule 5. If the suggested fix changes a public API, it's still `needs-design`.
- "I personally know the answer to the missing-info question" — doesn't override rule 4. The skill's job is to classify what the issue *contains*, not what you happen to know. Bucket as `needs-info`; your knowledge goes into the drafted question comment as context.

## Tie-breakers

- `needs-info` vs `needs-design` → if asking the reporter 1–2 questions could plausibly resolve it, pick `needs-info`. If even with full info the team would still need to debate, pick `needs-design`.
- `duplicate` vs `stale` → `duplicate` wins. Linking is more useful than closing as stale.
- `ready-to-fix` vs `needs-design` → if the deep-dive (step 3) shows multiple plausible fix paths, demote to `needs-design`.

## Bucket size limit

There is no quota. If 90% of the backlog is `ready-to-fix`, that's fine. If 90% is `needs-info`, that's a reporting-quality signal, not a bug in the rubric.
