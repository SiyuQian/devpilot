# The Unknown-Unknowns Sweep

Five questions that push a review past naming/formatting nits and onto the behavior the PR introduces. Answer all five before writing findings. "N/A, because X" is a valid answer; a silently skipped question is not.

## 1. Local pattern fit

How is this kind of change done elsewhere in *this* codebase? Is the PR matching convention, diverging on purpose, or diverging by accident?

**How to check:** grep for the same concept (error type, helper name, middleware, migration shape). Read one or two neighboring files in the same package.

**What this surfaces:** accidental divergence — the single most common silent defect in PR review. The author may not know the helper exists, or may have written a parallel implementation because the existing one was hard to find.

## 2. Blast radius

Beyond the diff, who depends on the changed behavior?

**How to check:** grep for callers of every exported symbol in the diff; look at tests, configs, feature flags, migrations, docs, and any downstream service or client (mobile SDK, CLI, cron, webhook consumers).

**What this surfaces:** behavior changes that are invisible in the diff but touch code the author didn't open. "The diff shows what changed; the blast radius is usually larger."

## 3. Known pitfalls for this change class

Name the class of change first, then check the class-specific pitfalls against the diff. Bring security, data integrity, and reversibility into this question.

| Class | Pitfalls to check |
|---|---|
| Auth | Recursion on 401, session invalidation, token leakage in logs, replay, concurrent-refresh race |
| Concurrency | Races, deadlocks, lost updates, missing singleflight, unbounded goroutines |
| Migration | Backfill lock duration, rollback path, long transactions, ordering with deploys |
| DB query | N+1, missing index, unbounded scan, lost ordering, pagination off-by-one |
| Retry / rate-limit | Recursion, thundering herd, missing jitter, retry on non-idempotent write |
| Cache | Stale reads, invalidation on writes, cold-start, thundering herd on miss |
| Prompt / LLM | Cache invalidation, token-cost regression, prompt injection, non-deterministic output into a deterministic pipeline |
| Input boundary | Unsanitized input into shell / SQL / HTML / template / prompt, path traversal |
| Data write | Non-idempotent writes, missing unique constraint, delete without tombstone |
| Reversibility | Irreversible migration, external API call, published message, cache poisoning |

If the PR touches several classes, check each. If the class isn't in this table, name it explicitly and list the pitfalls you checked.

## 4. Stale-training check

Before asserting "the right way to do X is Y," ask whether that claim might be 6–18 months stale.

**How to check:** verify against `go.mod` / `package.json` / lockfiles for library versions. For anything fast-moving (framework APIs, SDK versions, model IDs, security advisories, deprecations), pull a recent source.

**What this surfaces:** confident-sounding advice based on training that is no longer correct.

## 5. Hand-rolled vs. off-the-shelf

What in this PR is being hand-rolled that has a mature option already in the repo or its dependencies?

**Common offenders:** retry loops, rate limiters, backoff, cache layers, diff parsers, auth flows, date math, slug generation, singleflight, YAML/JSON parsing, concurrency primitives.

**How to check:** search the repo and `go.mod` / `package.json` for existing utilities. If the dependency is already present, the hand-rolled version is a finding.

## Output format

Each question gets one line in the review's `### Unknown-Unknowns Sweep` section:

```
1. Local pattern fit: <finding or "matches convention in X">
2. Blast radius: <finding>
3. Known pitfalls (incl. security/data/reversibility): <finding>
4. Stale-training check: <finding or "N/A">
5. Hand-rolled vs. off-the-shelf: <finding or "N/A">
```
