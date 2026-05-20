# Devpilot Graph — Design

**Date:** 2026-05-19
**Status:** Brainstorm spec, pending plan
**Owner:** Siyu Qian

## 1. Motivation

`devpilot-pr-review` currently relies on a five-agent fanout that reads diffs, greps for callers, and reasons about test coverage from textual signals. This works, but the same structural questions (who calls this function, is it on a critical path, does it have tests, what does it implement) get re-derived by every agent on every PR. The signal quality is uneven and the token cost is high.

The user's other project, `code-review-graph` (Python), shows that a persistent code graph delivers these answers cheaply and accurately. However, taking a hard dependency on `code-review-graph` for `devpilot-pr-review` creates a coupling between two projects with diverging roadmaps (general graph tool vs. PR review enhancement) and forces users to install a Python toolchain.

This document specifies **`devpilot graph`**: a Go-native graph subsystem built into the `devpilot` CLI, owned and maintained inside the devpilot repo, distributed via the existing single-binary install. It serves `devpilot-pr-review` first but is designed as a general subsystem that future skills (`devpilot-dead-code-cleanup`, `devpilot-resolve-issues`, `devpilot-scanning-repos`) can adopt.

## 2. Scope

### In scope (v1)

- Tree-sitter based parsing for **Go, TypeScript, JavaScript, Rust**
- Persistent SQLite-backed graph cache under `~/.devpilot/`
- Incremental updates driven by git diff
- Node types: `file`, `function`, `method`, `class`, `struct`, `interface`, `type`
- Edge types: `contains`, `calls`, `imports`, `tests`, `implements`, `extends`
- Derived queries: `callers_of`, `callees_of`, `tests_for`, `impact_radius`, `hub_nodes`, `implementors_of`, `cross_dir_edges`
- CLI subcommand surface (see §4)
- `preflight` composite query producing JSON for `devpilot-pr-review` fanout
- Grep fallback when graph is unavailable
- Skill integration: new step 1.5 in `devpilot-pr-review` SKILL.md

### Out of scope (v1; deferred)

- Other languages (Python, Java, C/C++, etc.) — language scope is intentionally narrow
- Vector / semantic search — no embeddings layer in v1
- Leiden community detection — replaced by directory-based grouping
- Flow detection and criticality scoring — replaced by in-degree hub detection
- Wiki / visualization / docs generation
- Multi-repo registry / cross-repo search
- General-purpose MCP server (CLI is the only interface in v1)
- End-to-end PR replay benchmark (deferred to post-v1.0)

## 3. Architecture

```
devpilot/
├── cmd/devpilot/
│   └── graph.go                       # cobra subcommand registration
├── internal/graph/
│   ├── parser/                        # go-tree-sitter wrappers per language
│   │   ├── parser.go                  # common Parser interface
│   │   ├── go.go
│   │   ├── typescript.go
│   │   ├── javascript.go
│   │   └── rust.go
│   ├── resolver/                      # cross-file symbol resolution
│   │   ├── imports.go                 # generic import edge construction
│   │   ├── tsconfig.go                # TypeScript path-alias resolution
│   │   ├── gomod.go                   # Go module path resolution
│   │   └── interfaces.go              # implements / extends inference
│   ├── store/                         # SQLite persistence
│   │   ├── schema.go                  # tables, indexes
│   │   ├── migrations.go              # forward-only migrations
│   │   └── store.go                   # CRUD + transactional batch ops
│   ├── cache/                         # ~/.devpilot/ management
│   │   ├── paths.go                   # repo-key hashing, dir layout
│   │   ├── flock.go                   # build/update concurrency lock
│   │   └── ttl.go                     # preflight JSON garbage collection
│   ├── query/                         # read-side API
│   │   ├── callers.go
│   │   ├── callees.go
│   │   ├── tests.go
│   │   ├── impact.go
│   │   ├── hubs.go
│   │   ├── implementors.go
│   │   ├── context.go                 # source-snippet pull
│   │   ├── detect_changes.go
│   │   └── preflight.go               # composite for pr-review skill
│   ├── envelope/                      # uniform JSON output envelope
│   │   └── envelope.go
│   ├── grep_fallback/                 # diff-driven grep when graph absent
│   │   └── fallback.go
│   └── lsp/                           # test-only: drive gopls/tsc/rust-analyzer
│       ├── gopls.go
│       ├── tsc.go
│       └── rust_analyzer.go
└── skills/devpilot-pr-review/
    ├── SKILL.md                       # adds step 1.5
    ├── references/
    │   ├── preflight.md               # new — preflight contract + schema
    │   ├── fanout.md                  # modified — SHARED_PR_HEADER includes preflight
    │   ├── unknown-unknowns.md        # modified — Agent A consumes preflight first
    │   └── template.md                # modified — Architecture Impact body section
    └── scripts/
        ├── preflight.sh               # entry: invokes `devpilot graph preflight`
        └── grep_fallback.sh           # entry: degraded mode when CLI fails
```

### Data flow

1. Skill `step 1.5` invokes `scripts/preflight.sh <repo> <base> <head>`.
2. `preflight.sh` checks the cache at `~/.devpilot/graphs/<repo-key>/graph.db`. If missing or stale relative to `base`, it triggers `devpilot graph build <repo>`.
3. `devpilot graph preflight --base <sha> --head <sha>` reads the cache and emits a JSON document (§6) to a temp file under `~/.devpilot/preflight/`.
4. The skill includes the JSON path in `SHARED_PR_HEADER`; subagents consume `changed_symbols` directly instead of grepping.
5. On any failure, `preflight.sh` exits with code `10` and writes a degraded JSON via `grep_fallback.sh`; the skill still runs but with `mode=fallback_grep`.

## 4. CLI surface

All commands output JSON to stdout (no human-readable mode). Errors are reported inside the JSON envelope, not via exit code (except for catastrophic failure → exit 20).

### Maintenance

```
devpilot graph build <repo>              # full build or auto-incremental update
devpilot graph status <repo>             # cache stats, freshness, covered languages
devpilot graph clean [--repo X | --all]  # remove cached graphs
```

`build` auto-decides full vs. incremental based on cache freshness and presence of base SHA in the graph. There is no separate `update` verb.

### Generic queries

```
devpilot graph query <pattern> <target> [--depth N] [--limit N]
```

Patterns supported in v1: `callers_of`, `callees_of`, `tests_for`, `imports_of`, `implementors_of`, `implements`.

### Named shortcuts

These are semantically clear aliases over `query` and underlying composite queries:

```
devpilot graph impact <file...> [--depth N]
devpilot graph hubs [--threshold N]
devpilot graph context <symbol> [--depth N]
```

### PR-review composites

```
devpilot graph detect-changes --base <sha> --head <sha>
devpilot graph preflight --base <sha> --head <sha>
```

### Envelope

Every command returns:

```json
{
  "version": "1",
  "command": "<verb>",
  "ok": true,
  "data": { ... },
  "warnings": [],
  "error": null,
  "next_tool_suggestions": ["..."]
}
```

`next_tool_suggestions` is borrowed from `code-review-graph`: each handler returns a short list of likely useful follow-up commands tailored to the result, reducing skill-side trial-and-error.

`ok: false` with a populated `error` block represents a *handled* failure. The CLI itself exits 0 for handled failures so skills can read structured error info; the CLI exits nonzero only on unrecoverable crashes. (The exit-code tiers in §8 — 0/10/20 — refer to the **`scripts/preflight.sh` wrapper**, not the underlying `devpilot graph` binary.)

### Node ID format

Aligned with `code-review-graph`:

```
<repo-relative-path>::<container>.<name>
```

Examples:
```
internal/payment/processor.go::PaymentProcessor.Charge
src/billing/charge.ts::PaymentProcessor#charge
```

The `#` vs `.` separator preserves language idioms; downstream consumers treat them as opaque IDs.

## 5. Storage and cache layout

### On-disk layout

```
~/.devpilot/
├── graphs/
│   └── <repo-key>/                    # repo-key = sha1(absolute repo root)[:12]
│       ├── graph.db                   # SQLite database
│       ├── meta.json                  # last-build HEAD SHA, languages, parser version, sizes
│       └── build.lock                 # flock file for concurrent build coordination
└── preflight/
    └── <repo-key>-<timestamp>.json    # TTL 7 days, lazy-cleaned on next preflight
```

### Why all under `~/.devpilot/` (not repo-local)

Devpilot's existing convention puts state in `<repo>/.devpilot/` (e.g., `logs/`). Graphs intentionally break this convention to enable cache sharing across git worktrees of the same repo — a common developer setup. The break is recorded here so future maintainers do not "fix" it by drifting back to repo-local storage.

The repo-key uses `repo_root` absolute path rather than the origin URL so that different clones of the same repo do not share a potentially-stale graph; the cost is small (one extra build per clone) compared to the debugging pain of cross-clone staleness.

### SQLite driver

`modernc.org/sqlite` (pure Go transpilation). Performance is 30–50% slower than the cgo `mattn/go-sqlite3` driver, but the project's queries are bounded by single-digit ms in typical operation and the cgo-free build preserves single-binary cross-compilation — critical for devpilot's existing install model.

### Schema (simplified)

```sql
CREATE TABLE nodes (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,                    -- file | function | method | class | struct | interface | type
  path TEXT NOT NULL,
  name TEXT NOT NULL,
  container TEXT,                        -- parent symbol id, NULL for top-level
  language TEXT NOT NULL,
  start_line INTEGER,
  end_line INTEGER,
  is_exported INTEGER NOT NULL DEFAULT 0,
  signature_hash TEXT                    -- detect signature_changed in incremental update
);

CREATE TABLE edges (
  src TEXT NOT NULL,
  dst TEXT NOT NULL,
  kind TEXT NOT NULL,                    -- contains | calls | imports | tests | implements | extends
  PRIMARY KEY (src, dst, kind)
);

CREATE INDEX idx_edges_dst_kind ON edges(dst, kind);  -- for callers_of
CREATE INDEX idx_edges_src_kind ON edges(src, kind);  -- for callees_of
CREATE INDEX idx_nodes_path ON nodes(path);
```

Migrations are forward-only. The cache directory carries its schema version in `meta.json`; on mismatch the cache is rebuilt from scratch (acceptable trade for v1).

### Incremental update

On `build` with an existing cache:

1. Compute `git diff --name-status <last_built_sha> HEAD` to list changed files.
2. Delete all nodes whose `path` is in the changed set; cascade-delete dependent edges.
3. Re-parse changed files; insert new nodes and edges.
4. Update `meta.json` with new HEAD SHA.

If `last_built_sha` is not an ancestor of HEAD (force-push, branch switch), fall back to full rebuild.

### Concurrency

- `build.lock` flock guards write operations; wait up to 60s.
- Reads use SQLite WAL mode and require no lock; concurrent preflight queries are safe.
- If the lock is held longer than 60s, the caller proceeds with the (possibly stale) existing database and emits a warning rather than blocking the review.

## 6. Preflight JSON schema

The `preflight` composite produces a single JSON document that the PR-review skill includes wholesale in `SHARED_PR_HEADER`. Standard archive (Q6 "B") sized at ≤4000 tokens per typical PR; large PRs are truncated to top-50 changed symbols by risk score.

### Schema

```json
{
  "version": "1",
  "command": "preflight",
  "ok": true,
  "data": {
    "mode": "cached | cached_updated | built | built_ephemeral | fallback_grep",
    "graph": {
      "freshness": { "covers_base_sha": true, "stale_files": 0 },
      "languages": ["go", "ts"],
      "skipped_files": []
    },
    "changed_symbols": [
      {
        "id": "internal/payment/processor.go::PaymentProcessor.Charge",
        "kind": "method",
        "is_exported": true,
        "is_new": false,
        "change_type": "modified",
        "callers": {
          "count": 23,
          "in_hub": true,
          "sample": ["api/checkout.go:42::handleCheckout"]
        },
        "callees_changed": ["internal/auth/Session.Validate"],
        "tests": {
          "has_tests": true,
          "test_symbols": ["internal/payment/processor_test.go::TestCharge"]
        },
        "implementors_of": null,
        "implements": ["Payable"],
        "community": "internal/payment",
        "risk_factors": ["hub", "critical_signature_change"]
      }
    ],
    "cross_community_edges": [
      {
        "from": "internal/billing",
        "to": "internal/auth",
        "count_added": 2,
        "samples": ["internal/billing/Charge → internal/auth/Session"]
      }
    ],
    "risk_summary": {
      "hub_nodes_modified": 1,
      "untested_public_changes": 3,
      "interface_changes": 1,
      "new_cross_community_edges": 2
    },
    "truncated_symbols": []
  },
  "warnings": [],
  "error": null,
  "next_tool_suggestions": [
    "devpilot graph context <symbol> --depth 1   # pull source for high-risk symbols",
    "devpilot graph query callers_of <symbol> --depth 2   # trace deeper"
  ]
}
```

### Sizing rules

- `callers.sample`: top 10 by `is_exported` then alphabetical
- `tests.test_symbols`: include all
- `callees_changed`: names only, no signatures
- `cross_community_edges.samples`: top 5
- Source snippets are never included; consumers call `devpilot graph context <symbol>` on demand
- `changed_symbols` is truncated to N=50 by risk score descending; remainder IDs go in `truncated_symbols`

### Risk score

Computed inside `preflight` to rank `changed_symbols`:

```
risk = is_exported * 2
     + in_hub * 3
     + interface_change * 3
     + untested * 1
```

Symbols with `risk == 0` still appear if there is room within the 50-symbol budget.

### "Community" definition (v1)

The `community` field is the **shallowest directory containing the symbol that has more than one child directory or file**, capped at depth 3. This is a deliberate degradation of `code-review-graph`'s Leiden-based communities; it is cheap, deterministic, and adequate for "does this PR cross module boundaries" detection.

## 7. Skill integration

### Changes to `SKILL.md`

Insert step 1.5 in the workflow:

```
0. Eligibility gate
1. Load PR
1.5 Graph preflight (best-effort)
    Run: bash scripts/preflight.sh <repo> <base> <head>
    On exit 0 or 10: include the JSON path in SHARED_PR_HEADER
    On exit 20: omit and note in the review body
2. Parallel fanout
3. Filter + merge
4. Draft review
5. Post
```

### Changes to `references/fanout.md`

Modify `SHARED_PR_HEADER` to declare:

> If `$PREFLIGHT_JSON` exists and `mode != fallback_grep`, prefer `data.changed_symbols[].callers` over manual grep for blast-radius questions. Treat `data.risk_summary` fields as deterministic hints (no LLM judgment required).

### Changes to `references/unknown-unknowns.md`

Agent A's question 2 (Blast radius) gets a fast-path: consume preflight `callers` data first, only fall back to grep when `mode == fallback_grep` or when the symbol is not in `changed_symbols` (e.g. cross-cutting concerns).

### Changes to `references/template.md`

Add an "Architecture Impact" section to the review body template, populated from preflight data:

```markdown
### Architecture Impact

- Touched communities: <data.community values, distinct>
- Hub nodes modified: <id (caller_count)>
- New cross-module edges: <from> → <to> (×<count>)
- Test coverage: <covered>/<total> changed public symbols have tests
- Interface changes: <implementor counts>

_Source: devpilot graph (mode=<data.mode>)_
```

When `mode == fallback_grep` the section is replaced with a single line: `_Architecture Impact: unavailable (graph fallback)_`.

### New file: `references/preflight.md`

Documents the JSON contract, mode values, and how each subagent should consume the data. Acts as the source of truth for skill-side expectations; mirrors §6 of this document.

### New scripts

- `scripts/preflight.sh` — repo+base+head → JSON path or exit 10/20 (see §8)
- `scripts/grep_fallback.sh` — minimal viable JSON when the CLI is unavailable (see §8)

## 8. Failure modes and exit codes

### Exit code semantics

| Exit | Meaning | Skill action |
|---|---|---|
| 0 | Success (may include warnings) | Use JSON, proceed with fanout |
| 10 | Soft failure — fell back to grep | Use JSON (`mode=fallback_grep`); fanout agents notice degraded mode |
| 20 | Hard failure — no JSON produced | Skip step 1.5; original fanout behavior |

### Failure mode matrix

| Scenario | Behavior | Exit | mode |
|---|---|---|---|
| Cache hit + base SHA covered | Direct query | 0 | `cached` |
| Cache hit but stale | Auto `build` (incremental) | 0 | `cached_updated` |
| No cache, build succeeds | Build + query | 0 | `built` |
| Build timeout (>300s) | Kill build → grep fallback | 10 | `fallback_grep_timeout` |
| Build crash (panic/segv) | Grep fallback | 10 | `fallback_grep_crash` |
| Repo languages unsupported (PHP, Ruby, …) | Grep fallback | 10 | `fallback_grep_unsupported` |
| Partial file parse failures | Continue, skipped_files populated | 0 | `cached` + warnings |
| Base SHA missing (force-push) | Use merge-base as fallback reference | 0 | `cached` + warnings |
| `devpilot` binary missing | Grep fallback | 10 | `fallback_grep_no_binary` |
| Cache dir not writable | In-memory build | 0 | `built_ephemeral` |
| All paths exhausted | No JSON | 20 | — |

### Timeouts

- Build phase: 300s
- Query phase: 30s
- Total `preflight.sh`: 360s

### Version compatibility

`version: "1"` is the contract version. The skill validates this field and refuses to consume mismatched versions (exit 20). Devpilot CLI guarantees v1 output for at least one major release after v2 is introduced.

### Grep fallback minimum viable set

When `mode == fallback_grep` the JSON includes only:

- `changed_symbols[].id` (parsed from diff)
- `changed_symbols[].kind` (best-effort from filename/regex)
- `changed_symbols[].is_exported` (capitalization heuristic for Go/Rust, `export` keyword for TS/JS)
- `changed_symbols[].callers.{count, sample}` (grep-based; flagged with `_fallback_caveats`)
- `warnings: ["graph unavailable, using grep fallback"]`

Fields not derivable from grep (`tests`, `implements`, `community`, `risk_factors`) are `null` or empty.

## 9. Testing strategy

### Layer 1 — Graph correctness (LSP cross-check, Q8.1 "B")

Drive `gopls`, `tsc --noEmit --listFiles`, and `rust-analyzer` programmatically against fixture repos. For each `find references` and `goto definition` result, assert the corresponding edge exists in our graph (or that its absence is justified, e.g., dynamic dispatch). Acceptance target:

- ≥ 90% precision and ≥ 90% recall on `callers_of`, `tests_for`, `implementors_of`
- Measured on devpilot itself, code-review-graph (as an OSS fixture), and one each for TS/JS/Rust

The LSP test layer is tagged `//go:build lsp_check` and runs in a nightly CI job, not on every PR (the LSP servers are heavy and flaky).

### Layer 2 — CLI output contract

- JSON schema files under `internal/graph/envelope/schemas/`
- One e2e test per CLI verb: invoke binary → parse stdout → schema validation + field-level assertions
- Run on every PR

### Layer 3 — Skill integration

v1 ships with unit-level fanout assertions only:

- Given a mock preflight JSON with specific risk factors, assert that the relevant agent brief in `references/fanout.md` triggers the corresponding heuristic
- 5+ such assertions in the v1 release gate

End-to-end PR replay (running the skill against historical PRs and measuring recall/precision against known-defect ground truth) is deferred to post-v1.0. The fixture-collection effort is sized as a separate project once the skill is stable.

### Performance benchmarks

CI runs build/update/preflight on a ~10k LOC fixture and a real ~50k LOC repo. Regressions exceeding 25% relative to the baseline fail the build. Baselines (per Q8.3):

| Operation | Target (~10k LOC, ~500 files) |
|---|---|
| Full build | < 60s |
| Incremental update | < 5s |
| Preflight query | < 10s |

## 10. Release gate (v1)

A release of `devpilot graph v1` requires all of the following:

1. Four-language fixture snapshot tests pass.
2. LSP cross-check on devpilot + one OSS fixture per language achieves ≥ 90% precision and recall on three core edge types.
3. All nine CLI subcommands (`build`, `status`, `clean`, `query`, `impact`, `hubs`, `context`, `detect-changes`, `preflight`) plus all six `query` patterns have e2e tests; JSON schema validation 100%.
4. Five or more skill-side unit assertions on fanout prompt content pass.
5. Performance baselines met on devpilot itself and one external ~10k LOC OSS repo.
6. Manual walkthrough of every failure mode in §8 produces the expected exit code and JSON shape.
7. Three real PRs reviewed manually with preflight enabled produce findings of equal or better quality vs. the pre-preflight baseline.

## 11. Risks and mitigations

| Risk | Severity | Mitigation |
|---|---|---|
| `go-tree-sitter` upstream maintenance is uneven | M | Pin to a specific commit in `go.mod`; budget for a fork if upstream stalls |
| `modernc.org/sqlite` performance hits | L | Baseline allows 10s preflight; 30–50% overhead leaves headroom. cgo driver remains a v1.x emergency lever |
| TypeScript path-alias resolution complexity | H | Borrow design from `code-review-graph`'s `tsconfig_resolver.py`; ship a fixture suite covering nested tsconfigs and project references |
| Go implicit interface satisfaction is expensive to compute | M | v1 restricts `implements` inference to exact method-set match; document that subset / structural matches are deferred |
| LSP-driven test layer is slow and flaky in CI | M | Nightly job, not per-PR; tagged `//go:build lsp_check`; mark known-flaky cases with documented bypasses |
| `is_exported && !has_tests` auto-finding produces noise | M | Ship the rule flag-gated, default off; enable only after one week of internal dogfooding |
| Graph cache divergence from working tree (manual edits without commit) | L | `build` recomputes per-file hashes; stale uncommitted files trigger reparsing on next `build` invocation |
| Devpilot binary version mismatch with skill expectations | M | Envelope `version` field plus skill-side check; document v1 support window publicly |

## 12. Timeline (rough)

| Workstream | Estimate |
|---|---|
| Parser layer (4 languages + cross-file resolve) | 3 weeks |
| Store + cache + incremental update | 2 weeks |
| Query layer (callers / tests / impact / hubs / implementors / preflight composite) | 2 weeks |
| CLI surface (11 subcommands + envelope + JSON schemas) | 1.5 weeks |
| LSP cross-check test infrastructure | 2 weeks |
| Skill integration (preflight.sh + SKILL.md edits + grep fallback) | 1 week |
| Buffer / debug / fixture collection / dogfooding | 1.5 weeks |
| **Total** | **~13 weeks** |

## 13. Open follow-ups (post-v1)

- End-to-end PR replay benchmark suite
- Vector / semantic search layer
- Flow detection and criticality scoring
- Leiden community detection
- Python / Java / C++ language support
- MCP server interface for direct Claude consumption
- Cross-repo registry

## 14. Decision log

Key choices reached during brainstorming (2026-05-19), each with the alternative rejected:

| # | Choice | Alternative | Reason |
|---|---|---|---|
| Q1 | General subcommand surface | PR-review-only verb | Multiple skills will consume |
| Q2 | Persistent cached graph | Ephemeral per-review build | Unblocks full-graph queries; enables incremental updates |
| Q3 | SQLite (`modernc.org/sqlite`) | Custom binary format, JSON | Incremental updates demand a real database; pure-Go preserves single-binary distribution |
| Q4 | Full v1 incl. `implements`/`extends` | Minimal v1, edges deferred | User accepted scope expansion for richer interface-change signals |
| Q5 | Medium CLI surface | Minimal or full | Covers foreseeable consumers without speculative verbs |
| Q5b | JSON-only output, uniform envelope | Dual human/JSON output | CLI consumers are always AI/code |
| Q6 | Standard (~4000 token) preflight, top-50 truncation | Compact or thick | Balance between completeness and fanout token budget |
| Q7 | 3-tier exit codes (0/10/20) with grep fallback | Hard fail or always succeed | Preflight must never block review |
| Cache layout | All under `~/.devpilot/` | Repo-local `.devpilot/` | Shared across worktrees |
| Q8.1 | LSP cross-check for accuracy | Snapshot-only or human-annotated | User selected highest-rigor option |
| Q8.2 | Defer end-to-end PR replay to post-v1.0 | Include in v1 | Fixture collection too large for v1 scope |
| Q8.3 | build<60s / update<5s / preflight<10s | Other baselines | Default targets accepted |
