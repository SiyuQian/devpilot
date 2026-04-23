# Labels

Run once per repository before the first scan. Safe to re-run — `gh label create` returns a non-zero exit when a label already exists, so guard with `|| true`.

```bash
gh label create repo-scan        --color FBCA04 --description "Filed by devpilot-scanning-repos"       || true
gh label create scan:security    --color B60205 --description "Security category"                 || true
gh label create scan:edge-case   --color D93F0B --description "Edge-case / robustness category"   || true
gh label create scan:coverage    --color 0E8A16 --description "Testing-coverage category"         || true
gh label create severity:high    --color B60205 --description "High-severity scan finding"        || true
gh label create severity:medium  --color FBCA04 --description "Medium-severity scan finding"      || true
gh label create severity:low     --color C5DEF5 --description "Low-severity scan finding"         || true
```

## Rationale

- `repo-scan` is the root label — filter to this to see the full scan backlog.
- `scan:<category>` lets the maintainer route the work (security → security team, coverage → the file's owner, etc.).
- `severity:<severity>` maps cleanly to the 3-tier triage bucket most teams already use.

Do not add more labels. A longer taxonomy slows triage more than it helps.
