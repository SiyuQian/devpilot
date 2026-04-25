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
- **Performance** — no chatty I/O on hot paths; no synchronous calls inside loops where batching applies; no unnecessary allocations in inner loops.
- **Security** — input validated at trust boundaries; secrets not logged; auth/authz not bypassed by the new path; no SSRF / SQLi / XSS / template injection / path-traversal surface.
- **Observability hooks** — new code paths have logs / metrics / errors a human can find when paged.

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

## Where each finding goes

- A category finding tied to one line (or a contiguous range) → **inline comment** at that line.
- A finding without a natural anchor (e.g. "this PR has no tests at all", "the entire approach hand-rolls X when Y is in `go.mod`") → **inline comment** anchored to the most representative line (the new function's signature, the changed file's first new line) with a one-liner saying the comment is about the change as a whole.
- The Unknown-Unknowns Sweep summary stays in the **body**; specific issues the sweep surfaces become inline comments.
- Overall verdict, Open Questions, what's working well → **body**.

If you find yourself reaching for the body to dump a finding because no line fits, anchor it instead. The author can ask for a different anchor; they cannot act on a body bullet.
