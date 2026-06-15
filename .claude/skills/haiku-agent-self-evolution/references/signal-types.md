# Signal Types

Every mutation must be grounded in at least one signal from this taxonomy. Speculative changes are rejected.

## Signal Taxonomy

| Signal Type | Source | Detection Method | Cost | Reliability |
|---|---|---|---|---|
| **lint_failure** | CI logs, `make lint` output | Parse exit code + grep for rule name in stderr | Cheapest | Highest — deterministic |
| **test_failure** | `make test` / `go test ./...` output | Parse `FAIL` lines, read test name and message | Cheap | Highest — deterministic |
| **git_hook_rejection** | Pre-commit hook stderr | Captured before push attempt | Cheap | Highest — deterministic |
| **repeated_violation** | Git history across ≥2 PRs/commits | Same rule appears in multiple CI runs or diffs | Medium | High — pattern requires N≥2 |
| **review_comment** | GitHub PR review comments | Fetch via GitHub MCP; grep for imperative language ("should", "always", "never", "use X instead") | Medium | Medium — needs interpretation |
| **architecture_violation** | Import analyzer, structural tests | Run `go build ./...`, check for unexpected cross-package imports | Cheap | High |
| **golden_principle_regression** | GOLDEN_PRINCIPLES sweep output | Compare current grade vs previous run | Medium | Medium — requires baseline |

## Minimum Evidence Threshold

| Signal Type | Minimum Before Proposing Mutation |
|---|---|
| lint_failure | 1 occurrence (deterministic rule) |
| test_failure | 1 occurrence (clear failure message) |
| git_hook_rejection | 1 occurrence |
| repeated_violation | ≥2 occurrences across different PRs/commits |
| review_comment | ≥2 occurrences of substantially the same comment, OR 1 occurrence that is clearly a structural rule |
| architecture_violation | 1 occurrence |
| golden_principle_regression | Regression across ≥2 consecutive sweeps |

Single review comments about stylistic preference (not structural) go into a "watching" list, not a mutation.

## Signal Collection Commands

**Scan recent git history:**
```bash
git log --oneline -20
git log --oneline --since="7 days ago"
```

**Check for repeated lint rule violations:**
```bash
make lint 2>&1 | grep -oP '(?<=\[)[a-z-]+(?=\])' | sort | uniq -c | sort -rn
```

**Read CI failure from specific commit:**
```bash
git show <hash> --stat
git diff <hash>^..<hash>
```

**Count CLAUDE.md lines (before any addition):**
```bash
wc -l CLAUDE.md
```

## Signal Record Format

Each signal collected during Phase 1 must be recorded as:

```json
{
  "type": "lint_failure",
  "rule": "unused-variable",
  "occurrences": 3,
  "source": "ci",
  "evidence": [
    "commit abc1234: internal/auth/commands.go:45:3: unused variable 'err'",
    "PR #51 CI log: same rule, internal/slack/client.go:12:2",
    "PR #45 CI log: same rule, internal/gmail/fetch.go:88:5"
  ],
  "first_seen": "2026-06-10",
  "last_seen": "2026-06-15"
}
```
