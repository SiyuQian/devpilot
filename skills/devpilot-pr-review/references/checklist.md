# Quality Checklist

The behavior sweep (`unknown-unknowns.md`) catches what a narrow diff view misses. This checklist catches what a behavior pass would skip — code quality, architecture, testing, requirements, production readiness. Run both.

Every category produces zero or more findings. Each finding becomes an inline comment anchored to a specific line (see `template.md` → "Inline comment template"). Cross-cutting findings without a single natural anchor go on the most representative line, with a one-liner in the comment that says so.

## Code quality

- **Separation of concerns** — no hidden coupling, no functions doing two unrelated jobs, no helper that knows about its caller.
- **Error handling** — errors wrapped at boundaries (`fmt.Errorf("doing X: %w", err)`); no swallowed errors; no silent fallbacks; no `panic` in library code.
- **Type safety** — no unsafe casts; no `any` / `interface{}` / `unknown` where a concrete type fits; nil/optional handled explicitly.
- **DRY without premature abstraction** — duplicated logic that should be one helper *and* shared helpers stretched to fit a new caller they shouldn't.
- **Edge cases** — empty input, nil pointer, zero-length collections, max values, negative values, unicode, time-zone boundaries, timezone-aware comparisons.
- **Naming and clarity** — names match what the function/variable does at the head SHA, not at an earlier draft. Comments justify *why*, not *what*.

## Architecture

- **Public surface** — matches the change's purpose; no leaked internals; new exported symbols have a clear caller story.
- **Scalability** — cost grows reasonably with input size; no accidental O(n²) over user-controlled input; no unbounded slices, maps, or goroutines.
- **Observability hooks** — new code paths have logs / metrics / errors a human can find when paged.

## Security [REQUIRED CHECKS]

Agent B MUST walk every item below for code in the diff. For each item, produce a finding OR return an entry in the `coverage` block (`{item}: checked, no_evidence`). "Risk is low because the input isn't attacker-controlled today" is the canonical silent-skip rationalization — write it as a Consider-level finding with the assumption made explicit, not as a silent pass.

- **Hardcoded secrets** — API keys, tokens, passwords, private keys, OAuth client secrets, signing keys, JWT secrets in source/config/test fixtures added by the diff.
- **Credential placement** — secrets transported via URL query / referer-leaking surface / GET requests / debug logs; should be header / body / encrypted store.
- **String interpolation into structured contexts** — values from variables/inputs interpolated into:
  - Auth/HTTP headers (CRLF injection, quote escaping in `OAuth ...="%s"` style templates).
  - SQL / NoSQL queries (injection).
  - Shell commands / `exec` / process spawn.
  - HTML / template engines (XSS).
  - File paths (traversal).
  - Log format strings.
- **AuthN ≠ AuthZ** — every endpoint / handler / RPC method touched: both authentication AND object-level authorization checks present? A user being logged in is not authorization.
- **Error response leakage** — error strings echo internal state (DB errors verbatim, stack traces, response bodies that may contain auth tokens, secrets, PII).
- **Sensitive data in logs** — log statements added or modified include credential fields, full request bodies, PII, session IDs.
- **Cross-host credential propagation** — HTTP clients that follow redirects: does Authorization / Cookie get re-sent to a different host? (Go's default behavior changed across versions; verify explicitly for the version in `go.mod`.)
- **TLS / transport** — new outbound calls use HTTPS; certificate verification not disabled; no `InsecureSkipVerify` introduced.

## Performance [REQUIRED CHECKS]

Same protocol as Security: every item produces a finding OR a `coverage` entry. "Looks fine" is not a coverage entry.

- **N+1 / loop-bound I/O** — DB / HTTP / fs / RPC call inside a loop where batching applies.
- **Unbounded list / find_all on request paths** — pagination / `LIMIT` / max-results missing.
- **Index coverage** — new query filter / sort / join columns when DB schema is visible in the repo; flag if no index covers them.
- **Serial independent awaits** — sequential `await` / blocking calls that are mutually independent and should run concurrently (`Promise.all`, `errgroup.Go`).
- **Blocking calls on hot paths** — `time.Sleep`, synchronous fs/net, large `io.ReadAll` of externally-controlled bodies in request handlers / RPC paths.
- **Connection reuse** — new `http.Client{}` or DB connection per call instead of sharing; missing `Transport` tuning where the call pattern justifies it.
- **Inner-loop allocation in hot code** — `fmt.Sprintf` / repeated map allocation / string concatenation in tight loops on changed lines (skip if the loop is bounded by a small constant).

## Coverage block (Agent B return shape)

Agent B's response MUST include a `coverage` block alongside `findings`:

```yaml
findings:
  - ... (as before)

coverage:
  security:
    hardcoded_secrets: checked, no_evidence
    credential_placement: finding_raised
    string_interpolation: finding_raised
    authn_vs_authz: not_applicable (no endpoint touched)
    error_response_leakage: checked, no_evidence
    sensitive_data_in_logs: checked, no_evidence
    cross_host_credential_propagation: checked, no_evidence
    tls_transport: not_applicable
  performance:
    n_plus_one: checked, no_evidence
    unbounded_list: not_applicable
    index_coverage: not_applicable (no schema visible)
    serial_independent_awaits: checked, no_evidence
    blocking_hot_path: finding_raised
    connection_reuse: checked, no_evidence
    inner_loop_allocation: checked, no_evidence
```

**Allowed values per item:** `checked, no_evidence` | `finding_raised` | `not_applicable (<one-line reason>)`. Anything else, including missing keys, is invalid — main session re-dispatches once (see `confidence.md` → coverage assertion).

**`not_applicable` is not a free pass.** Use it only when the diff genuinely cannot match the item (e.g. no DB code touched → `index_coverage: not_applicable`). "I looked and it was fine" is `checked, no_evidence`, not `not_applicable`.

## Testing

- **Tests exercise behavior, not mocks of our own packages** (`CLAUDE.md` rule).
- **Edge cases covered** — same list as Code quality above.
- **Integration tests where a unit test cannot prove the contract** — anything touching DB, queue, network, file system, or cross-process state.
- **All tests passing on the head SHA.** A red CI check is itself a finding.
- **A risky path without a test is a finding**, even when nothing else is wrong with the code.

## Requirements

- **All items in the PR description / linked plan are present in the diff.** A line in the description without a corresponding code change is a finding.
- **No scope creep** — unrelated changes (drive-by refactors, formatting churn) are flagged; ask the author whether to split.
- **Breaking changes called out** — API shape, schema, on-disk format, exported behavior, CLI flags. Authors who flag them stay; reviewers who notice unflagged ones surface them.

## Production readiness

- **Migration path** — forward and rollback both work; long transactions guarded; ordering relative to deploy is sound.
- **Backward compatibility** — feature flags / version checks where the rollout is staged; old clients keep working through the deploy window.
- **Documentation** — user-visible changes reflected in README, docs, help text, or CHANGELOG (whichever this repo uses).
- **No obvious bugs** — dead branches, swapped conditions, off-by-one, leaked resources, forgotten `defer Close()`.

Inline-vs-body routing is governed by SKILL.md `<inline_by_default>` — every finding tied to a line goes inline.
