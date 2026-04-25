# Labels

Run once per repository before the first scan. Safe to re-run — `gh label create` returns a non-zero exit when a label already exists, so guard with `|| true`.

Every label carries information. There is no redundant root label: `scan/<category>` already uniquely identifies issues filed by this skill.

## Required labels

```bash
# --- categories (one per issue) ---
gh label create scan/security    --color B60205 --description "Filed by devpilot-scanning-repos: security category"      || true
gh label create scan/edge-case   --color D93F0B --description "Filed by devpilot-scanning-repos: edge-case / robustness" || true
gh label create scan/coverage    --color 0E8A16 --description "Filed by devpilot-scanning-repos: test-coverage gap"      || true

# --- security subcategories (exactly one when category=scan/security) ---
gh label create sec/injection         --color B60205 --description "SQL / shell / template / NoSQL injection"     || true
gh label create sec/authn-authz       --color B60205 --description "Auth bypass, missing checks, role tampering"  || true
gh label create sec/secrets           --color B60205 --description "Hardcoded keys, tokens, credentials"          || true
gh label create sec/crypto            --color B60205 --description "Weak hashes, bad RNG, TLS verify disabled"    || true
gh label create sec/path-traversal    --color B60205 --description "Zip-slip, ../ in user paths, archive escape"  || true
gh label create sec/ssrf-csrf         --color B60205 --description "SSRF, CORS misconfig, missing CSRF"           || true
gh label create sec/deserialization   --color B60205 --description "pickle, unsafe yaml, gob from untrusted"      || true
gh label create sec/tls-misconfig     --color B60205 --description "Cert verify off, weak ciphers, plaintext"     || true

# --- edge-case subcategories (exactly one when category=scan/edge-case) ---
gh label create edge/nil-deref         --color D93F0B --description "Nil pointer / map / slice deref"             || true
gh label create edge/bounds-overflow   --color D93F0B --description "Off-by-one, integer over/underflow"          || true
gh label create edge/error-swallowed   --color D93F0B --description "Discarded errors, returned-nil-on-error"     || true
gh label create edge/concurrency       --color D93F0B --description "Race, deadlock, leaked goroutine, double-close" || true
gh label create edge/resource-leak     --color D93F0B --description "Unclosed file/conn/row/ticker on error path" || true
gh label create edge/input-validation  --color D93F0B --description "Unbounded user input → index/alloc/length"   || true

# --- coverage subcategories (exactly one when category=scan/coverage) ---
gh label create cov/no-test-file       --color 0E8A16 --description "Exported surface with no test file at all"   || true
gh label create cov/error-paths        --color 0E8A16 --description "Happy path tested, error branches not"        || true
gh label create cov/integration-seam   --color 0E8A16 --description "Untested boundary between two packages"       || true
gh label create cov/stale-test         --color 0E8A16 --description "Production churn, test file stagnant"         || true

# --- severity (one per issue; switched : → / for namespace consistency) ---
gh label create severity/high     --color CC0000 --description "High-severity scan finding"   || true   # darker than scan/security on purpose
gh label create severity/medium   --color FBCA04 --description "Medium-severity scan finding" || true
gh label create severity/low      --color C5DEF5 --description "Low-severity scan finding"    || true

# --- confidence (one per issue, from the scoring pass) ---
gh label create confidence/75     --color BFD4F2 --description "Scoring agent gave 75/100 — real, meaningful, verified"     || true
gh label create confidence/100    --color 0052CC --description "Scoring agent gave 100/100 — certain, load-bearing, frequent" || true
```

## `area/<top-level-dir>` (auto-created on demand)

Derived from each finding's `file` path: take the first path segment, replace `/` with `-`. Example: a finding on `internal/auth/middleware.go` gets `area/internal-auth`.

The orchestrator MUST create the area label idempotently right before filing the issue:

```bash
area_label="area/$(echo "<finding-file>" | cut -d/ -f1 | tr '/' '-')"
gh label create "$area_label" --color FEF2C0 --description "Auto-derived from finding file path" || true
```

This lets the file's CODEOWNER filter `is:open label:area/internal-auth` and see only their slice.

## Per-issue label contract

Every filed issue MUST have exactly **five** labels:

1. One `scan/<category>` — `security` | `edge-case` | `coverage`
2. One subcategory matching the category — `sec/*` | `edge/*` | `cov/*`
3. One `severity/<level>` — `high` | `medium` | `low`
4. One `confidence/<score>` — `75` | `100` (matches the scoring-pass output)
5. One `area/<top-level-dir>` — auto-derived from `file`

If a scanner returns a finding whose subcategory doesn't fit any of the labels above, the orchestrator drops it and logs the mismatch — do not invent new subcategory labels at filing time. The fixed taxonomy is the contract.

## Rationale

- **`repo-scan` was deleted** because every scan-filed issue already carries `scan/*`. A label every issue has filters nothing.
- **Subcategories are mandatory**, not optional, so triage queries like `label:sec/injection` actually return a meaningful slice instead of every security issue lumped together.
- **`area/*` labels** route work to the file's owner without the orchestrator needing CODEOWNERS context — the maintainer already knows their dir.
- **`confidence/*`** lets reviewers process the `100` bucket first; mixed lists slow triage.
- **Color collision fixed**: `scan/security` keeps `#B60205`, `severity/high` is now `#CC0000` — distinguishable in the UI.

## Migration from the old taxonomy

If you previously ran the skill and have issues labeled `repo-scan`, `scan:security`, etc., do NOT mass-relabel. Run:

```bash
gh issue list --label repo-scan --state all --limit 1000 --json number,labels
```

and let the dedupe step (SKILL.md step 6) treat them as historical — old issues stay on the old labels, new issues use the new ones. The dedupe normalizer matches on title, not labels, so this is safe.
