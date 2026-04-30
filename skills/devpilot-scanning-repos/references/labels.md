# Labels

The skill defines a canonical taxonomy below. **Every label name uses `{type}:{value}` (colon, not slash).** Before creating any of these labels, the orchestrator MUST first snapshot the repo's existing labels and reuse any suitable ones — see SKILL.md step 2. Only create the canonical label when the repo has no semantically equivalent label.

## Reconciliation procedure (summary; full version in SKILL.md step 2)

1. `gh label list --limit 200 --json name,description > /tmp/devpilot-existing-labels.json` (raise `--limit` if it returns full).
2. For each canonical label below, look in the snapshot for a label whose **name OR description** clearly maps to the same purpose. If found, reuse it. If not, create the canonical `{type}:{value}` label with the `gh label create` line below.
3. When uncertain, do NOT reuse — false reuse poisons triage queries (`label:scan:security` returning unrelated `security` issues is worse than two parallel labels).

## Canonical taxonomy and create commands

Each line is the **fallback** used when no suitable repo label exists. Run only the lines whose canonical label is missing from the repo and has no suitable existing equivalent.

```bash
# --- categories (one per issue) ---
gh label create scan:security    --color B60205 --description "Filed by devpilot-scanning-repos: security category"
gh label create scan:edge-case   --color D93F0B --description "Filed by devpilot-scanning-repos: edge-case / robustness"
gh label create scan:coverage    --color 0E8A16 --description "Filed by devpilot-scanning-repos: test-coverage gap"

# --- security subcategories (exactly one when category=scan:security) ---
gh label create sec:injection         --color B60205 --description "SQL / shell / template / NoSQL injection"
gh label create sec:authn-authz       --color B60205 --description "Auth bypass, missing checks, role tampering"
gh label create sec:secrets           --color B60205 --description "Hardcoded keys, tokens, credentials"
gh label create sec:crypto            --color B60205 --description "Weak hashes, bad RNG, TLS verify disabled"
gh label create sec:path-traversal    --color B60205 --description "Zip-slip, ../ in user paths, archive escape"
gh label create sec:ssrf-csrf         --color B60205 --description "SSRF, CORS misconfig, missing CSRF"
gh label create sec:deserialization   --color B60205 --description "pickle, unsafe yaml, gob from untrusted"
gh label create sec:tls-misconfig     --color B60205 --description "Cert verify off, weak ciphers, plaintext"

# --- edge-case subcategories (exactly one when category=scan:edge-case) ---
gh label create edge:nil-deref         --color D93F0B --description "Nil pointer / map / slice deref"
gh label create edge:bounds-overflow   --color D93F0B --description "Off-by-one, integer over/underflow"
gh label create edge:error-swallowed   --color D93F0B --description "Discarded errors, returned-nil-on-error"
gh label create edge:concurrency       --color D93F0B --description "Race, deadlock, leaked goroutine, double-close"
gh label create edge:resource-leak     --color D93F0B --description "Unclosed file/conn/row/ticker on error path"
gh label create edge:input-validation  --color D93F0B --description "Unbounded user input → index/alloc/length"

# --- coverage subcategories (exactly one when category=scan:coverage) ---
gh label create cov:no-test-file       --color 0E8A16 --description "Exported surface with no test file at all"
gh label create cov:error-paths        --color 0E8A16 --description "Happy path tested, error branches not"
gh label create cov:integration-seam   --color 0E8A16 --description "Untested boundary between two packages"
gh label create cov:stale-test         --color 0E8A16 --description "Production churn, test file stagnant"

# --- severity (one per issue) ---
gh label create severity:high     --color CC0000 --description "High-severity scan finding"
gh label create severity:medium   --color FBCA04 --description "Medium-severity scan finding"
gh label create severity:low      --color C5DEF5 --description "Low-severity scan finding"

# --- confidence (one per issue, from the scoring pass) ---
gh label create confidence:75     --color BFD4F2 --description "Scoring agent gave 75/100 — real, meaningful, verified"
gh label create confidence:100    --color 0052CC --description "Scoring agent gave 100/100 — certain, load-bearing, frequent"
```

Note: no `|| true` guards. The reconciliation step has already proven the label is missing — a failure here is a real signal, not noise to swallow.

## Suitable-match guidance

The orchestrator should treat these as suitable reuses (examples — not exhaustive):

| Canonical | Suitable existing label (examples) | Not suitable |
|---|---|---|
| `scan:security` | `security`, `type:security`, `kind/security` | `bug` (too broad) |
| `scan:coverage` | `testing`, `tests`, `coverage` | `enhancement` |
| `severity:high` | `priority:high`, `p0`, `severity-1`, `critical` | `urgent` (priority ≠ severity) |
| `severity:low` | `priority:low`, `p3`, `nice-to-have` | `wontfix` |
| `confidence:75` / `confidence:100` | (almost never present) | anything ranking-shaped — confidence is internal to the scoring pass |
| `sec:injection` | `vulnerability:injection`, `security:injection` | `security` (loses subcategory signal) |
| `area:cmd` | `cmd`, `component:cmd`, `area-cmd` | `module` (too generic) |

Rule of thumb: **subcategory labels (`sec:*`, `edge:*`, `cov:*`) and `confidence:*` rarely have a suitable equivalent — almost always create.** Top-level category, severity, and area labels are the ones most worth reusing.

## `area:{top-level-dir}` (resolved lazily at filing time)

Derived from each finding's `file` path: take the first path segment. Example: `internal/auth/middleware.go` → canonical label `area:internal`.

At filing time, the orchestrator checks `/tmp/devpilot-existing-labels.json` for a suitable area-ish label (e.g. the repo already has `internal` or `area-internal`). If suitable, reuse it. Otherwise:

```bash
area_label="area:$(echo "<finding-file>" | cut -d/ -f1)"
gh label create "$area_label" --color FEF2C0 --description "Auto-derived from finding file path"
```

This lets a CODEOWNER filter `is:open label:area:internal` (or whichever label was reused) and see only their slice.

## Per-issue label contract

Every filed issue MUST have exactly **five** labels (using whichever name the step-2 mapping resolved to — canonical or reused):

1. One category — canonical: `scan:security` | `scan:edge-case` | `scan:coverage`
2. One subcategory matching the category — canonical: `sec:*` | `edge:*` | `cov:*`
3. One severity — canonical: `severity:high` | `severity:medium` | `severity:low`
4. One confidence — canonical: `confidence:75` | `confidence:100`
5. One area — canonical: `area:{top-level-dir}`

If a scanner returns a finding whose subcategory doesn't match the fixed enum, the orchestrator drops it and logs the mismatch — do not invent new subcategory values at filing time. The fixed taxonomy is the contract; reuse only swaps the *label that carries* the canonical meaning, not the meaning itself.

## Rationale

- **`{type}:{value}` everywhere** matches GitHub's increasingly common convention and reads as a key/value pair (`type:security`, `severity:high`) rather than a path.
- **Reuse before create** keeps the repo's label space clean. A repo that already has `security` and `priority:high` does not need `scan:security` and `severity:high` cluttering the sidebar — but the canonical names are still the fallback when nothing suitable exists.
- **Subcategories are mandatory**, not optional, so triage queries like `label:sec:injection` actually return a meaningful slice instead of every security issue lumped together.
- **`confidence:*`** lets reviewers process the `100` bucket first; mixed lists slow triage.

## Migration from the old taxonomy

If you previously ran the skill and have issues labeled `repo-scan`, `scan/security`, etc., do NOT mass-relabel. The dedupe step (SKILL.md step 6) matches on title, not labels — old issues stay on the old labels, new issues use the new colon-form (or whatever the reconciliation resolved to).
