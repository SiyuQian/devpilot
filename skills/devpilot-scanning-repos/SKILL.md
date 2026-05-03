---
name: devpilot-scanning-repos
description: Use when the user asks to scan, audit, or sweep an entire GitHub repository for issues and file them as tickets — "scan this repo", "audit the codebase", "find bugs/security holes/missing tests", "check the docs are still accurate", "/repo-scan", "open issues for all the problems you find". Scans security, edge cases, testing coverage, and doc/code drift (CLAUDE.md, AGENTS.md, README.md and the docs they link to) without assuming business logic. Do NOT use for reviewing a single PR (use devpilot-pr-review) or language-specific style review (use devpilot-google-go-style).
---

# Repo Scan (Security / Edge Cases / Coverage → GitHub Issues)

## Files in this skill

| File | When to load |
|---|---|
| `agents/security-scanner.md` | Step 3 — sub-agent prompt for the security scanner. |
| `agents/edge-case-hunter.md` | Step 3 — sub-agent prompt for edge-case hunting (no business logic). |
| `agents/coverage-auditor.md` | Step 3 — sub-agent prompt for test-coverage gap detection. |
| `agents/doc-consistency-auditor.md` | Step 3 — sub-agent prompt for doc/code drift detection (CLAUDE.md, AGENTS.md, README.md and linked docs). |
| `references/scoring.md` | Step 4 — full 0/25/50/75/100 rubric + false-positive classes. |
| `references/issue-template.md` | Step 7 — exact `gh issue create` body and label contract. |
| `references/labels.md` | Step 2 — one-shot `gh label create` commands. |
| `scripts/check-findings.py` | Step 3.5 — validates each scanner's JSON output against the schema. |
| `evals/evals.json` | Test scenarios for skill behavior (not loaded at runtime). |

## Overview

A whole-repo sweep that dispatches **four parallel specialist sub-agents** (security, edge-case, coverage, doc-drift), scores every finding 0–100 for confidence, filters below threshold, then files each surviving finding as a labeled GitHub issue. Business logic is out of scope — scanners only catch mistakes a reasonable reader could flag without domain knowledge. The doc-drift scanner audits the entry-point docs (`CLAUDE.md`, `AGENTS.md`, `README.md`) and every doc file they link to, checking falsifiable claims against the current code.

**Core principle:** coverage during scan, filtering during scoring, noise-free issues at the end. The sub-agents are told to surface everything they notice; a separate scoring pass kills the noise so the human only sees load-bearing issues.

## When NOT to Use

- Single PR / diff review → `devpilot-pr-review`.
- Pure style / lint / formatting → the relevant style skill (`devpilot-google-go-style`, etc.) + the project's linter.
- Business-logic correctness ("does this function compute the right tax rate?") → a human with domain context.
- Repo without a `.github`-style issue tracker, or user doesn't want issues created → ask first; print findings to terminal instead.

## Workflow

1. **Resolve target.** Accept `owner/repo`, a clone URL, or "this repo" (use `gh repo view --json nameWithOwner`). Verify with `gh repo view`.
2. **Reconcile labels — rename non-canonical matches, reuse exact matches, create what's missing.** Do NOT blindly paste a `gh label create` block. The procedure:
   1. **Snapshot existing labels:** `gh label list --limit 200 --json name,description > /tmp/devpilot-existing-labels.json`. If the count returned equals the limit, raise `--limit` and re-run until the result is shorter than the limit.
   2. **For each label the skill needs** (the full taxonomy lives in `references/labels.md`): classify the closest existing repo label into one of three buckets:
      - **Exact match** — the repo label name already equals the canonical `type:value` (e.g. repo has `scan:security`). **Reuse as-is.**
      - **Semantic match, non-canonical name** — same intent, different name (e.g. repo has `security` or `type-security` for `scan:security`; `nil-deref` for `edge:nil-deref`). **Rename to the canonical name** with `gh label edit "<old-name>" --name "<canonical-name>" --description "<canonical-desc>"`. Renaming preserves all existing issue associations, so this is safe and brings the repo onto our taxonomy. Confirm the rename plan with the user before executing if more than 3 labels are being renamed at once.
      - **Too generic / wrong intent** — e.g. `bug` for `edge:nil-deref`, or `enhancement` for anything scan-related. **Do NOT rename and do NOT reuse.** Create the canonical label fresh; leave the generic label alone.
   3. **Build a name-mapping table** for this run: `canonical_name → label_to_apply`. After renames, every entry should be the canonical `type:value` name. The orchestrator uses this mapping when filing issues in step 7.
   4. **Create the labels that had no match.** Use the canonical `type:value` form. See `references/labels.md` for the exact `gh label create` invocation per label (colors and descriptions).
   5. **Print the reconciliation summary** to the user before continuing: `N reused, R renamed (old → new), M created` — so they can spot a wrong rename or reuse before issues get filed.

   `area:*` labels follow the same three-bucket rule but are reconciled lazily at filing time (step 7): for each finding, check the snapshot for an existing area-ish label covering the same top-level dir; rename it to `area:<dir>` if it's a semantic match with a non-canonical name, otherwise create.
2.5. **Build the file manifest AND the doc manifest.**
   - **Source manifest** (`/tmp/devpilot-scan-manifest.txt`): one walk, sampled path list of production source files for the security / edge-case / coverage scanners. They MUST NOT re-walk — they may only read paths in this manifest. See "Scaling for large repos" below. Default cap: **800 files** (`--full` raises to 2000, `--scope <dir>` constrains to a subtree).
   - **Doc manifest** (`/tmp/devpilot-doc-manifest.txt`): built independently for the doc-drift scanner. Steps:
     1. Find every entry-point file, case-insensitive, anywhere in the repo: `fd -HI -t f -i '^(claude|agents|readme)\.md$'` (fallback: `find . -type f -iregex '.*/\(claude\|agents\|readme\)\.md'`).
     2. Parse markdown links from each — both inline `[text](path)` and reference `[text]: path` forms — keep only relative targets that resolve to existing files with doc-ish extensions (`.md`, `.mdx`, `.txt`, `.rst`) or living under a `docs/`-style directory. Strip `#anchor` for resolution but keep it for the scanner's reporting.
     3. Recurse one hop at a time, dedupe by absolute path, cap at depth 3 and at 200 doc files total. Write the resulting list to `/tmp/devpilot-doc-manifest.txt`. Print to the user: total entry points found, total docs in the manifest, and a tree showing which entry point pulled in which linked doc.
3. **Dispatch scanners in parallel.** In ONE message, launch four sub-agents using the prompts in `agents/`:
   - `agents/security-scanner.md` — pass `/tmp/devpilot-scan-manifest.txt`
   - `agents/edge-case-hunter.md` — pass `/tmp/devpilot-scan-manifest.txt`
   - `agents/coverage-auditor.md` — pass `/tmp/devpilot-scan-manifest.txt`
   - `agents/doc-consistency-auditor.md` — pass `/tmp/devpilot-doc-manifest.txt`
   Each returns a list of `Finding` objects (see format below). Scanners are told to emit everything they notice — including low-severity — because filtering happens in step 4, not in the scanner.
3.5. **Validate scanner output.** Pipe each scanner's JSON array through `python3 scripts/check-findings.py`. The `--manifest` flag is mandatory and chooses the manifest the scanner was dispatched against:
   - security / edge-case / coverage → `--manifest /tmp/devpilot-scan-manifest.txt`
   - doc-drift → `--manifest /tmp/devpilot-doc-manifest.txt`
   The script rejects findings whose `file` is not on the relevant manifest, missing required fields, invalid `category`/`subcategory`/`severity` enums, and empty `evidence`. Fix (or ask the scanner to re-emit) before scoring.
4. **Score every finding, in batches.** Group findings by category and dispatch ONE scoring sub-agent per batch of up to **25 findings** (not one per finding — the per-finding fan-out doesn't scale past ~50). The scoring agent returns a JSON array of `{index, score, reason}` aligned with the input order. See `references/scoring.md` for the rubric and the batched prompt.
5. **Filter.** Drop every finding with score `< 75`. If zero survive, stop — report "no high-confidence issues found" to the user and do not create issues.
6. **Deduplicate against existing issues.** Before filing, query existing scan issues. Use a search that covers BOTH the new taxonomy and the legacy `repo-scan` label (so re-runs against repos scanned under the old label set still dedupe correctly):
   ```bash
   gh issue list --search 'label:scan:security,scan:edge-case,scan:coverage,scan:doc-drift,repo-scan in:title' \
     --state all --limit 1000 --json title,number,state
   ```
   Normalize titles by lower-casing, stripping the `[scan:<category>]` / `[repo-scan:<category>]` prefix, and collapsing whitespace before comparing. Skip findings whose normalized title matches an existing issue. If `--limit 1000` returns exactly 1000, paginate with `--search "... created:<<date-of-oldest>"` until empty.
7. **File issues.** One `gh issue create` per surviving finding, using the template in `references/issue-template.md`. Labels: always exactly five — apply the labels from the step-2 mapping table for `scan:<category>`, the matching subcategory, `severity:<level>`, `confidence:<score>`, plus an `area:<top-level-dir>` resolved lazily here. For the area label: first check `/tmp/devpilot-existing-labels.json` for a suitable existing label (e.g. repo already has `area-cmd` or `cmd` covering the dir). If it's an exact match, reuse; if it's a semantic match with a non-canonical name, rename it to `area:<dir>` via `gh label edit`; otherwise run `gh label create area:<dir>`.
8. **Summarize.** Print a compact table to the user: `[category] [severity] title → #issue-number`.

## Finding format

Every scanner returns a JSON array of objects with exactly these fields:

```json
{
  "category": "security | edge-case | coverage | doc-drift",
  "subcategory": "sec:injection | sec:authn-authz | sec:secrets | sec:crypto | sec:path-traversal | sec:ssrf-csrf | sec:deserialization | sec:tls-misconfig | edge:nil-deref | edge:bounds-overflow | edge:error-swallowed | edge:concurrency | edge:resource-leak | edge:input-validation | cov:no-test-file | cov:error-paths | cov:integration-seam | cov:stale-test | doc:broken-link | doc:missing-file | doc:command-mismatch | doc:stale-claim | doc:cross-doc-conflict",
  "title": "<≤80 chars, imperative — e.g. 'Sanitize shell input in cmd/devpilot/run.go'>",
  "severity": "high | medium | low",
  "file": "<path relative to repo root>",
  "line_range": "L42-L58",
  "evidence": "<2–5 lines quoted from the file, with line numbers>",
  "why_it_matters": "<1–3 sentences, no business-logic claims>",
  "suggested_fix": "<1–3 sentences; null if scanner can't confidently propose one>"
}
```

`subcategory` must match `category` (`sec:*` for security, `edge:*` for edge-case, `cov:*` for coverage, `doc:*` for doc-drift). See `references/labels.md` for the fixed enum — scanners do NOT invent new subcategory values. If a finding doesn't fit any subcategory, the scanner picks the closest fit OR drops the finding.

For doc-drift, the `file` field is the **doc** containing the wrong claim (e.g. `README.md`, `docs/cli-reference.md`), not the source file the claim is about. The source file (or its absence) goes in `evidence`.

`evidence` is mandatory. A finding without quotable code is speculation — drop it at the scanner level.

## Scoring rubric (summary — full rubric in `references/scoring.md`)

| Score | Meaning | Action |
|---|---|---|
| 0 | False positive / pre-existing / would be caught by CI | drop |
| 25 | Might be real; scanner couldn't verify | drop |
| 50 | Real but minor nit | drop |
| 75 | Real, meaningful, verified in code | **file** |
| 100 | Certain, load-bearing, reproducible | **file** |

Threshold is 75, matching the confidence bar of a senior reviewer who doesn't cry wolf.

## What scanners MUST NOT flag

(These are pre-filtered at the scanner level — do not let findings like this through.)

- Anything a linter, formatter, type-checker, or compiler catches.
- Business-logic correctness — "this discount calculation is wrong" requires domain context the scanner doesn't have.
- Style / naming / readability nits — defer to style skills.
- "Missing tests" for files that are pure types, generated code, or obviously trivial (constants, one-line accessors).
- Dependency CVEs — that's Dependabot's job.
- Pre-existing issues on lines the scanner can see were untouched in recent history (scanner should `git log` to check if desired).

See each agent prompt for category-specific false-positive classes.

## Quick reference

| I want to… | Do this |
|---|---|
| Scan current repo | `gh repo view --json nameWithOwner` → pass to step 1 |
| Scan a specific path only | Tell each scanner: "scope = `internal/auth/` only" |
| Preview without filing issues | Skip step 2 and 7; print the table |
| Re-scan and not duplicate issues | Step 6 handles this; don't delete existing `repo-scan` issues |
| Change threshold | Edit step 5 — keep the 0/25/50/75/100 scale, move only the cutoff |

## Acceptance criteria (the "test" this skill is written against)

A correct run of this skill produces:

1. Exactly four scanner sub-agent dispatches, in parallel (security, edge-case, coverage, doc-drift).
2. Every filed issue has exactly five labels: `scan:<category>`, a matching `sec:*`|`edge:*`|`cov:*`|`doc:*` subcategory, `severity:<level>`, `confidence:<score>` (75 or 100), and an auto-derived `area:<top-level-dir>` (for doc-drift findings, derived from the doc file's path — e.g. a finding in `README.md` → `area:root`, in `docs/cli-reference.md` → `area:docs`).
3. No issue is filed whose scoring-agent score is below 75.
4. No duplicate of an existing open scan issue (under either the new `scan:*` labels or the legacy `repo-scan` label).
5. No finding cites business-logic correctness as the sole reason.
6. Every filed issue body quotes ≥2 lines of actual code with a `<file>#L<start>-L<end>` link.
7. If zero findings survive scoring, no issues are filed and the user is told so explicitly.

If any of these is violated, the skill failed — stop and correct before continuing.

## Scaling for large repos

A real-world stress test on `kubernetes/kubernetes` (16,942 source files, ~268k tokens just for the path list) surfaced three failure modes the workflow MUST defend against. This section is the contract — it is not optional.

### Failure modes (measured, not theoretical)

1. **Path-list explosion.** The full file list of a large monorepo can consume more tokens than a sub-agent's entire window, *before* a single file is read. Each scanner re-walking the tree triplicates the cost.
2. **Hardcoded directory allowlist misses.** The naive fallback `cmd/, internal/, api/, handlers/, controllers/, auth/, crypto/, utils/` matched **3% of code** on kubernetes (which has none of `internal/`, `api/`, `handlers/`, `controllers/`, `auth/`, `crypto/`, `utils/` at the top level). Hardcoded allowlists cannot be the fallback.
3. **Sink-grep volume swamps verify.** `exec.Command` alone returns 252 hits, `math/rand` 214, `InsecureSkipVerify` 93 — totalling 647 hits across just 6 patterns. A scanner told to "read the surrounding function" for each will either skip verification (and emit raw greps) or stop after a tiny fraction.

### The manifest algorithm (step 2.5)

The orchestrator builds the manifest BEFORE dispatching scanners:

1. **Inventory.** Run `find . -type f \( -name '*.go' -o -name '*.py' -o -name '*.js' -o -name '*.ts' -o -name '*.rb' -o -name '*.java' -o -name '*.rs' \) -not -path './vendor/*' -not -path './node_modules/*' -not -path '*/gen/*' -not -path '*/.git/*' | wc -l`. Call this `N`.
2. **If `N` ≤ 800**, the manifest is the full list — no sampling needed.
3. **If `N` > 800**, sample down to 800 with this priority order (concatenate, dedupe, truncate at 800):
   1. **Hot-path churn.** Files changed in the last 90 days, ranked by commit count: `git log --since=90.days.ago --name-only --pretty=format: -- '*.go' '*.py' '*.js' '*.ts' '*.rb' '*.java' '*.rs' | sort | uniq -c | sort -rn | awk '$2 != "" {print $2}'`. Take the top 400.
   2. **Top-level high-signal dirs by file count.** Dynamically discover (do NOT use the hardcoded allowlist): `find . -mindepth 1 -maxdepth 1 -type d -not -name '.*' -not -name 'vendor' -not -name 'node_modules' -not -name 'test' -not -name 'tests' -not -name 'docs' -not -name 'examples'`, count source files in each, take the top 4 dirs by count, sample evenly from them up to 300 files. This is the part that fixes the kubernetes-allowlist mismatch — a repo with `pkg/` and `staging/` at the top will have those picked, not ignored.
   3. **Security-sensitive name patterns** as a final fill: any path containing `auth`, `crypto`, `secret`, `cred`, `token`, `password`, `cert`, `tls`, `oauth`, `session`, `jwt`, `signin`, `login`. Take up to 100.
4. **Write `/tmp/devpilot-scan-manifest.txt`**, one path per line. Print to the user: total `N`, manifest size, and the top 5 dirs represented (so they can sanity-check that the sampling caught the right slice).
5. **`--full` mode** raises the cap to 2000 but still goes through the same priority sampling. Above 2000, refuse and ask the user to scope.
6. **`--scope <dir>` mode** skips churn ranking — manifest is `find <dir>` capped at 800.

### Sink-grep budget (per scanner)

Each scanner runs its dangerous-sink greps ONLY against files in the manifest, not the whole tree:

```bash
grep -nE 'exec\.Command|InsecureSkipVerify|...' $(cat /tmp/devpilot-scan-manifest.txt)
```

Cap per pattern: **40 hits**. If a pattern returns more, take the top 40 by file (prefer files that also appear in the churn list — recently-modified hits beat ancient ones). The remainder is logged as "skipped: N additional `<pattern>` hits not verified" in the scanner's output, so the user sees coverage gaps explicitly rather than silently.

### Scoring batching (step 4)

Group findings by category, send up to **25 findings per scoring sub-agent** dispatch in one message. For 200 findings across three categories that's ~9 scoring dispatches, not 200. The scoring agent's batched prompt lives in `references/scoring.md`.

### Dedupe pagination (step 6)

`gh issue list --limit 200` silently truncates — confirmed against kubernetes/kubernetes. Always use `--limit 1000`, check whether exactly 1000 returned, and paginate with `created:<<<date>` until the page is short. See step 6.

## Common mistakes

- **Letting scanners filter their own output.** They should over-report. The scoring pass does the filtering. Merging the two loses calibration.
- **Using the scanner agent to also create the issue.** Don't — the scanner has too much context. File issues from the main agent after scoring.
- **Dropping the evidence block in the issue body.** Without it the human has to re-derive the finding. File a crap issue once and nobody trusts the skill.
- **Mass-creating the full taxonomy upfront without checking the repo.** The skill MUST snapshot existing labels (step 2) and reconcile against them. Blindly creating `scan:security` when the repo already has `security` doubles the label space and fragments triage queries — rename `security` to `scan:security` instead.
- **Reusing a non-canonical name as-is instead of renaming it.** If the repo has `security` and we file under `scan:security`, both labels now exist with overlapping intent. Rename the existing one to bring the repo onto the canonical taxonomy.
- **Renaming a too-generic label.** Don't rename `bug` to `edge:nil-deref` — it's attached to unrelated existing issues. Generic labels stay; create the canonical one alongside.
- **Creating subcategory or severity labels inside the issue-creation loop.** Reconcile in step 2, file in step 7. Only `area:*` is resolved lazily, and even then it goes through the same suitable-existing-label check.
- **Asking scanners to rank severity *and* confidence.** Confidence is the scoring pass's job; scanners assign severity only.
- **Forgetting the dedupe step.** Re-running the skill must be idempotent or the user will stop running it.

## Evaluation

Test scenarios for this skill live in `evals/evals.json`. Each eval gives a prompt, expected output shape, and machine-checkable assertions (e.g. *`exactly_three_scanner_dispatches`*, *`no_business_logic_findings_filed`*, *`all_issues_have_three_labels`*). Run before shipping any change to scanner prompts or the scoring rubric.
