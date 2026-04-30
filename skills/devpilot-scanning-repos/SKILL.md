---
name: devpilot-scanning-repos
description: Use when the user asks to scan, audit, or sweep an entire GitHub repository for issues and file them as tickets — "scan this repo", "audit the codebase", "find bugs/security holes/missing tests", "/repo-scan", "open issues for all the problems you find". Scans security, edge cases, and testing coverage without assuming business logic. Do NOT use for reviewing a single PR (use devpilot-pr-review) or language-specific style review (use devpilot-google-go-style).
---

# Repo Scan (Security / Edge Cases / Coverage → GitHub Issues)

## Files in this skill

| File | When to load |
|---|---|
| `agents/security-scanner.md` | Step 3 — sub-agent prompt for the security scanner. |
| `agents/edge-case-hunter.md` | Step 3 — sub-agent prompt for edge-case hunting (no business logic). |
| `agents/coverage-auditor.md` | Step 3 — sub-agent prompt for test-coverage gap detection. |
| `references/scoring.md` | Step 4 — full 0/25/50/75/100 rubric + false-positive classes. |
| `references/issue-template.md` | Step 7 — exact `gh issue create` body and label contract. |
| `references/labels.md` | Step 2 — one-shot `gh label create` commands. |
| `scripts/check-findings.py` | Step 3.5 — validates each scanner's JSON output against the schema. |
| `evals/evals.json` | Test scenarios for skill behavior (not loaded at runtime). |

## Overview

A whole-repo sweep that dispatches **three parallel specialist sub-agents**, scores every finding 0–100 for confidence, filters below threshold, then files each surviving finding as a labeled GitHub issue. Business logic is out of scope — scanners only catch mistakes a reasonable reader could flag without domain knowledge.

**Core principle:** coverage during scan, filtering during scoring, noise-free issues at the end. The sub-agents are told to surface everything they notice; a separate scoring pass kills the noise so the human only sees load-bearing issues.

## When NOT to Use

- Single PR / diff review → `devpilot-pr-review`.
- Pure style / lint / formatting → the relevant style skill (`devpilot-google-go-style`, etc.) + the project's linter.
- Business-logic correctness ("does this function compute the right tax rate?") → a human with domain context.
- Repo without a `.github`-style issue tracker, or user doesn't want issues created → ask first; print findings to terminal instead.

## Workflow

1. **Resolve target.** Accept `owner/repo`, a clone URL, or "this repo" (use `gh repo view --json nameWithOwner`). Verify with `gh repo view`.
2. **Reconcile labels — reuse what the repo already has, only create what's missing.** Do NOT blindly paste a `gh label create` block. The procedure:
   1. **Snapshot existing labels:** `gh label list --limit 200 --json name,description > /tmp/devpilot-existing-labels.json`. If the count returned equals the limit, raise `--limit` and re-run until the result is shorter than the limit.
   2. **For each label the skill needs** (the full taxonomy lives in `references/labels.md`): check whether the repo already has a *suitable* label for the same purpose. "Suitable" means same semantic intent, not just same name. A repo label `security` or `type:security` covers `scan:security`; `bug` does NOT cover `edge:nil-deref` (too generic). The match rule: the existing label's name OR description clearly maps onto the canonical purpose listed in `references/labels.md`. When in doubt, do NOT reuse — false reuse poisons triage queries.
   3. **Build a name-mapping table** for this run: `canonical_name → label_to_apply` (either the canonical `type:value` name, or the suitable existing repo label). The orchestrator uses this mapping when filing issues in step 7.
   4. **Only create the labels that have no suitable existing match.** Use the canonical `type:value` form. See `references/labels.md` for the exact `gh label create` invocation per label (colors and descriptions). Skip any label already mapped to an existing repo label.
   5. **Print the reconciliation summary** to the user before continuing: `N reused, M created, list of each` — so they can spot a wrong reuse before issues get filed.

   `area:*` labels follow the same rule but are reconciled lazily at filing time (step 7): for each finding, check the snapshot for an existing area-ish label covering the same top-level dir before creating `area:<dir>`.
2.5. **Build a shared file manifest.** The orchestrator does ONE walk and hands a sampled file list to all three scanners. Scanners MUST NOT re-walk — they may only read paths in the manifest. See "Scaling for large repos" below for the algorithm. The manifest is a plain text file written to `/tmp/devpilot-scan-manifest.txt`, one repo-relative path per line. Default cap: **800 files**. Override with `--full` (raises to 2000) or `--scope <dir>` (manifest = `find <dir>` capped at 800).
3. **Dispatch scanners in parallel.** In ONE message, launch three sub-agents using the prompts in `agents/`:
   - `agents/security-scanner.md`
   - `agents/edge-case-hunter.md`
   - `agents/coverage-auditor.md`
   Pass the manifest path to each scanner. Each returns a list of `Finding` objects (see format below). Scanners are told to emit everything they notice — including low-severity — because filtering happens in step 4, not in the scanner.
3.5. **Validate scanner output.** Pipe each scanner's JSON array through `python3 scripts/check-findings.py --manifest /tmp/devpilot-scan-manifest.txt`. The `--manifest` flag is mandatory — it makes the script reject any finding whose `file` is not on the manifest, enforcing the contract from step 2.5. The script also rejects missing required fields, invalid `category`/`subcategory`/`severity` enums, and empty `evidence`. Fix (or ask the scanner to re-emit) before scoring.
4. **Score every finding, in batches.** Group findings by category and dispatch ONE scoring sub-agent per batch of up to **25 findings** (not one per finding — the per-finding fan-out doesn't scale past ~50). The scoring agent returns a JSON array of `{index, score, reason}` aligned with the input order. See `references/scoring.md` for the rubric and the batched prompt.
5. **Filter.** Drop every finding with score `< 75`. If zero survive, stop — report "no high-confidence issues found" to the user and do not create issues.
6. **Deduplicate against existing issues.** Before filing, query existing scan issues. Use a search that covers BOTH the new taxonomy and the legacy `repo-scan` label (so re-runs against repos scanned under the old label set still dedupe correctly):
   ```bash
   gh issue list --search 'label:scan:security,scan:edge-case,scan:coverage,repo-scan in:title' \
     --state all --limit 1000 --json title,number,state
   ```
   Normalize titles by lower-casing, stripping the `[scan:<category>]` / `[repo-scan:<category>]` prefix, and collapsing whitespace before comparing. Skip findings whose normalized title matches an existing issue. If `--limit 1000` returns exactly 1000, paginate with `--search "... created:<<date-of-oldest>"` until empty.
7. **File issues.** One `gh issue create` per surviving finding, using the template in `references/issue-template.md`. Labels: always exactly five — apply the labels from the step-2 mapping table for `scan:<category>`, the matching subcategory, `severity:<level>`, `confidence:<score>`, plus an `area:<top-level-dir>` resolved lazily here. For the area label: first check `/tmp/devpilot-existing-labels.json` for a suitable existing label (e.g. repo already has `area-cmd` or `cmd` covering the dir) and reuse it; only run `gh label create area:<dir>` when no suitable repo label exists.
8. **Summarize.** Print a compact table to the user: `[category] [severity] title → #issue-number`.

## Finding format

Every scanner returns a JSON array of objects with exactly these fields:

```json
{
  "category": "security | edge-case | coverage",
  "subcategory": "sec:injection | sec:authn-authz | sec:secrets | sec:crypto | sec:path-traversal | sec:ssrf-csrf | sec:deserialization | sec:tls-misconfig | edge:nil-deref | edge:bounds-overflow | edge:error-swallowed | edge:concurrency | edge:resource-leak | edge:input-validation | cov:no-test-file | cov:error-paths | cov:integration-seam | cov:stale-test",
  "title": "<≤80 chars, imperative — e.g. 'Sanitize shell input in cmd/devpilot/run.go'>",
  "severity": "high | medium | low",
  "file": "<path relative to repo root>",
  "line_range": "L42-L58",
  "evidence": "<2–5 lines quoted from the file, with line numbers>",
  "why_it_matters": "<1–3 sentences, no business-logic claims>",
  "suggested_fix": "<1–3 sentences; null if scanner can't confidently propose one>"
}
```

`subcategory` must match `category` (`sec:*` for security, `edge:*` for edge-case, `cov:*` for coverage). See `references/labels.md` for the fixed enum — scanners do NOT invent new subcategory values. If a finding doesn't fit any subcategory, the scanner picks the closest fit OR drops the finding.

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

1. Exactly three scanner sub-agent dispatches, in parallel.
2. Every filed issue has exactly five labels: `scan:<category>`, a matching `sec:*`|`edge:*`|`cov:*` subcategory, `severity:<level>`, `confidence:<score>` (75 or 100), and an auto-derived `area:<top-level-dir>`.
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
- **Mass-creating the full taxonomy upfront without checking the repo.** The skill MUST snapshot existing labels (step 2) and reuse any suitable ones. Blindly creating `scan:security` when the repo already has `security` doubles the label space and fragments triage queries.
- **"Suitable" stretched too far.** Reusing `bug` for every `edge:*` finding loses the per-subcategory triage signal. When the existing label is too generic, create the canonical one.
- **Creating subcategory or severity labels inside the issue-creation loop.** Reconcile in step 2, file in step 7. Only `area:*` is resolved lazily, and even then it goes through the same suitable-existing-label check.
- **Asking scanners to rank severity *and* confidence.** Confidence is the scoring pass's job; scanners assign severity only.
- **Forgetting the dedupe step.** Re-running the skill must be idempotent or the user will stop running it.

## Evaluation

Test scenarios for this skill live in `evals/evals.json`. Each eval gives a prompt, expected output shape, and machine-checkable assertions (e.g. *`exactly_three_scanner_dispatches`*, *`no_business_logic_findings_filed`*, *`all_issues_have_three_labels`*). Run before shipping any change to scanner prompts or the scoring rubric.
