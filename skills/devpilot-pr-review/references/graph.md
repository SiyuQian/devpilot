# Graph Enrichment (`devpilot graph preflight`)

Graph is the **fact bed** under the fanout. It does not produce findings on its own. It tells every subagent, before they read code, exactly which symbols this PR changed, who calls them, whether any of them are hubs, and which public surface lacks tests. Subagents stop guessing about blast radius; the main session can corroborate or contradict findings against ground truth.

Pre-fanout, run the preflight once. Inject the result into the shared header that every Agent A–E brief sees. Subagents do **not** call `devpilot graph` themselves — they consume the structured payload.

## When to use

Run preflight whenever the PR touches code in a language the graph supports (Go, TypeScript/JavaScript, Rust at time of writing). Skip the optional follow-ups (`graph context`, `graph impact`) unless a specific finding requires deeper trace.

## Command

```bash
bin/devpilot graph preflight --base <base-sha> --head <head-sha>
```

For an incremental re-review (see eligibility.md), pass `--base <last_reviewed_sha> --head <head_sha>` instead of PR base.

Cache the JSON to `/tmp/pr_review_graph.json`.

## Fallback triggers (skip graph, fall through to grep)

Skip preflight and tell the body why if **any** of:

- `data.mode != "built"` in the payload (graph cache missing, build failed, language unsupported).
- Repo has no graph cache and `devpilot graph status` returns `ok:false`.
- Preflight exits non-zero or takes > 30 s.
- PR touches only unsupported languages (e.g. Python-only, shell-only, docs-only).

Do **not** auto-run `devpilot graph build` — the build can be slow and the user may not want it triggered by a review. Note the fallback in the body's sweep summary: `Behavior trace: grep-only (graph unavailable: <reason>)`.

## Payload — what each field means for the review

```jsonc
{
  "data": {
    "mode": "built",                       // or "fallback" → skip graph; see above
    "graph": { "freshness": { "covers_base_sha": true, "stale_files": 0 } },
    "changed_symbols": [
      {
        "id": "internal/auth/oauth.go::StartFlow",
        "kind": "function",
        "is_exported": true,
        "change_type": "modified",
        "callers": { "count": 2, "in_hub": false,
                     "sample": ["internal/gmail/service.go::Service.Login", ...] },
        "tests": { "has_tests": false, "test_symbols": [] },
        "community": "internal/auth",
        "risk_factors": ["untested_public"]
      }
    ],
    "cross_community_edges": [
      { "from": "internal/gmail", "to": "internal/auth",
        "count_added": 7, "samples": ["..."] }
    ],
    "risk_summary": {
      "hub_nodes_modified": 0,
      "untested_public_changes": 15,
      "interface_changes": 0,
      "new_cross_community_edges": 8
    }
  }
}
```

| Field | What Agent A–E does with it |
|---|---|
| `changed_symbols[].callers` | Agent A's blast-radius answer. Authoritative. Skip the grep step. |
| `changed_symbols[].callers.in_hub` | If `true`, Agent A escalates that symbol's review and notes it in `sweep_summary.blast_radius`. |
| `changed_symbols[].tests` | Agent A and Agent B both consult. `has_tests:false` on an exported behavior change is a Should-fix finding. |
| `changed_symbols[].risk_factors` | `untested_public`, `hub`, `interface_change` — each gates a specific finding pattern. |
| `cross_community_edges` | Agent A's "is this PR widening the contract between two packages?" question. New cross-community edges in an internal-only PR are a Consider-level finding. |
| `risk_summary.untested_public_changes` | Aggregate count for the body's sweep summary. |
| `risk_summary.interface_changes` | If > 0, Agent A traces implementors via `graph context --id <iface>`. |

## Confidence weighting (consumed by `confidence.md` merge step)

After the fanout returns, the main session reconciles each finding against the graph payload:

- **Corroborated** — the finding names a symbol whose callers/risk_factors match the defect. Confidence floor raised to 85. (Cap stays at 95 unless literal-string evidence pushes it to 100.)
- **Contradicted** — the finding asserts "X calls Y" but graph shows Y has zero callers, or asserts "this is a hub" but `in_hub:false`. Confidence capped at 50, which drops it under the default threshold.
- **Unsupported** — finding sits outside the graph's coverage (no symbol match, or graph in fallback mode). No adjustment. Original score stands.

A finding can be both corroborated on one dimension and contradicted on another — take the more conservative outcome.

## Optional follow-ups

Used only when a specific surviving finding needs a deeper trace; not run by default.

```bash
bin/devpilot graph context --id <symbol-id> --depth 1   # source + immediate callers/callees
bin/devpilot graph impact  --files <path,path>          # caller union for symbols defined in files
```

Output is appended to the inline comment as evidence, not posted as a separate finding.

## Known noise patterns

- `change_type: "modified"` is line-overlap based. A struct or function can be marked modified when only neighboring lines shifted; its body may not have changed. Confirm against the diff before treating as a behavior change.
- `callers.count: 0` means "no static caller in indexed languages". Reflection, codegen, RPC, CLI dispatch tables, and test main files are invisible. Spot-check with one grep before calling something dead code.
- `risk_factors: ["untested_public"]` is a useful trigger but does not by itself constitute a finding — combine with a concrete contract change in the diff.

## What graph does NOT do

- It does not judge style, naming, comments, or CLAUDE.md compliance — those stay with Agents C and E.
- It does not catch logic bugs inside a function body — that is Agent B.
- It does not know dynamic dispatch beyond what the static graph records. Reflection, codegen, RPC, and string-keyed dispatch are invisible. Cross-reference Agent A's grep sweep when the change involves any of those.
- It does not replace reading the code. It removes the "did I miss a caller?" anxiety so subagents spend their tokens on judgment, not enumeration.
