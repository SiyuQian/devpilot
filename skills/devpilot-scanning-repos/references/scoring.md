# Scoring rubric

Findings are scored in **batches of up to 25** per sub-agent dispatch. The per-finding fan-out (one dispatch per finding) does not scale past ~50 findings — empirically observed on large-repo runs where scanners returned 200+ findings. Group by category, send one batch per dispatch.

Keep the scoring agent isolated from the scanner agents — it must evaluate each finding on its own merits, not be anchored by the scanner's framing.

## Batched scoring prompt (paste verbatim)

> You are scoring a batch of code-scan findings for confidence. The input is a JSON array of `Finding` objects. Return ONLY a JSON array of `{"index": <int>, "score": <int>, "reason": "<one sentence>"}` aligned with the input order. Use the scale below.
>
> For each finding: read the cited file at `line_range`, evaluate against the rubric, assign a score in `{0, 25, 50, 75, 100}`. Do NOT skip findings; if you can't read a file, score it 25 with reason "could not verify".

(Per-finding rubric is identical to the single-finding version below.)

## Single-finding prompt (legacy — for spot rescoring)

> You are evaluating a single code-scan finding for confidence. Read the finding, read the file cited in `file` around `line_range`, and assign a score using this scale. Return ONLY `{"score": <int>, "reason": "<one sentence>"}`.
>
> - **0** — false positive. Does not stand up to light scrutiny, or is a pre-existing issue the PR didn't touch, or would be caught by the repo's linter / type-checker / compiler / CI.
> - **25** — might be real but not verifiable. The cited code could behave as claimed, but the scanner did not read enough context to confirm. If this is a style finding, it is one that is not explicitly called out in any `CLAUDE.md` / `AGENTS.md` / `ARCHITECTURE.md` in the repo.
> - **50** — real but minor. Verified, but it's a nit or happens rarely in practice.
> - **75** — real, meaningful, verified. The evidence in the finding matches the code, the failure mode is reachable via realistic inputs, and fixing it would materially improve the repo.
> - **100** — certain and load-bearing. Directly verifiable, triggered frequently in practice, and the fix is unambiguous.
>
> Rules:
> - If the finding cites business-logic correctness as the sole reason → score 0. Business logic is out of scope.
> - If the finding is in generated code or a vendored dependency → score 0.
> - If the claimed evidence does not actually appear in the cited file and line range → score 0.
> - If the finding would be caught by a common linter for this language (gofmt/golangci/eslint/ruff/etc.) → score 0.
> - If the finding is a coverage gap for pure types, constants, or one-line accessors → score 0.
> - Err toward the lower score when in doubt. A missed issue is recoverable; a filed false positive burns the maintainer's trust.

## Threshold

The orchestrator drops every finding with `score < 75`. Do not move the threshold below 75 without explicit user instruction — below 75 the precision collapses and the issue list becomes noise.

## Category-specific adjustments

Apply these *after* the base score:

- **Security finding, `severity: high`, score ≥ 75** → keep as-is; these are the highest-value issues.
- **Security finding where the attack path requires authenticated insider access that is already trusted** → cap at 50 (drop).
- **Edge-case finding in a path with an existing recover / fallback that already handles the claimed failure** → cap at 25 (drop).
- **Coverage finding on a file with < 30 lines of non-trivial code** → cap at 50 (drop); not worth a ticket.
- **Coverage finding on a security-sensitive file (auth, crypto, input parsing)** → floor at 75 if the scanner correctly identified it as a critical boundary.

## False-positive classes to score 0 aggressively

From the `/code-review:code-review` rubric, adapted for whole-repo scans:

- Pre-existing issues on code that hasn't been recently modified, unless the user explicitly asked for a legacy sweep.
- Issues the repo's CI will catch (missing imports, type errors, broken tests, formatting).
- General code-quality observations ("lacks tests", "poor documentation") not backed by a specific reachable problem.
- Issues in a file that has a `// nolint:xxx` or equivalent suppression comment covering exactly this case.
- "Could be refactored" — refactoring suggestions are never high-confidence findings.
- Speculation about future scale ("what if this is called a million times"), unless the code already does O(n²) work or materializes unbounded input.
