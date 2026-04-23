# Writing ARCHITECTURE.md

`ARCHITECTURE.md` is the map an agent reads *before* touching an unfamiliar part of the codebase. Its job is to stop the agent from re-deriving the wrong mental model from file names, and to record the constraints that keep an agent-authored codebase from decaying.

Unlike AGENTS.md, it is **not** always-loaded — the agent reads it on demand. So it can be longer, but every section should answer a question the agent would otherwise guess at.

## Day-one constraints, not eventual cleanup

With human teams, you can defer architecture until scaling pain forces the issue. With agents, you cannot: an agent will happily add a tenth way to do the same thing because it cannot feel the cost. The architecture work a human team would postpone until hundreds of engineers are on board has to happen on day one with agents — the constraints are what buy speed without decay.

This means ARCHITECTURE.md has two jobs: describe the current shape, **and** lock in the constraints early enough that the agent cannot drift across them.

## Audience

Assume the reader is a competent engineer who has never seen this repo, has 10 minutes, and is about to make a change. They need:

1. The shape of the system (what talks to what).
2. The invariants that must not be broken.
3. Where new code of type X belongs.

Everything else belongs in a design doc.

## Structure

```markdown
# Architecture

## System shape
One diagram (ASCII or mermaid). Boxes are processes / packages / services.
Arrows are data flow, labeled with protocol or call type.

## Components
For each top-level package/service, 2–5 lines:
- What it owns
- What it depends on
- What it must NOT depend on

## Invariants
Numbered list. Rules that, if violated, break the system in non-obvious ways.
Examples:
  1. `internal/auth` is the only package that reads credentials from disk.
  2. Cobra commands live with their domain; there is no central CLI router.
  3. External service clients live beside the domain logic that uses them.

## Extension points
"If you're adding X, it goes in Y because Z."
Cover the 5–10 most common change types. This is where agents save the most time.

## Decisions and their reasons
Short list of load-bearing choices with one-line rationale.
Link out to design docs for depth.

## Out of scope
Things this document deliberately does not cover (performance tuning, deployment, etc.)
and where to find them instead.
```

## Dimensions to constrain early

Pick an explicit rule for each dimension before the agent invents three variants. Each row becomes an Invariant plus, where possible, a paired sensor.

| Dimension | Example constraint |
|---|---|
| Module boundaries | `internal/auth/` may not import from `internal/trello/` |
| Data flow | One canonical path: request → handler → service → repo; no shortcuts |
| Error handling | One wrapping style (`fmt.Errorf("doing X: %w", err)`); one logging facade |
| Dependency surface | Whitelist of approved libs for HTTP, JSON, testing, logging |
| Naming | Canonical names for common concepts (`Client`, `Service`, `Store`) |
| Test shape | Table-driven, one subtest per row, named `TestFooBar` |
| Public API shape | Functional options, never positional bool flags |

## Invariant vs taste call vs style rule — where does the rule live?

A new rule shows up. Three docs could own it. Use this decision order:

1. **Linter can enforce mechanically?** → put it in the linter config, not in any markdown.
2. **Violation breaks the system non-obviously (build breaks, data corrupts, security leaks), AND applies repo-wide?** → ARCHITECTURE.md Invariants. Pair with a fitness function.
3. **Subjective taste — naming feel, factoring preference, "cleaner" choices, no automatic check possible?** → `GOLDEN_PRINCIPLES.md` (or equivalent taste doc).
4. **Conditional ("when doing X, prefer Y")?** → not an invariant. Push into a skill whose description matches X.

If you can't pick cleanly, the rule is probably two rules. Split it.

## What does NOT belong

- File-by-file walkthroughs — the code is the source of truth for that.
- API contracts — belong in generated docs or the code.
- Historical narrative ("originally we used X, then migrated to Y") — unless the history still constrains current design.
- Style rules, naming conventions — those are AGENTS.md or linter territory.
- Aspirational architecture ("we plan to…") — document what exists; put plans in `docs/exec-plans/`.

## Pairing with sensors

Every invariant in ARCHITECTURE.md should, where possible, be paired with a **fitness function** — a test or lint that fails when the invariant is broken. Invariants that exist only in prose rot fast under agent-authored code.

Examples of fitness functions (pick the ones your repo type needs):

*Structural (any repo):*
- ArchUnit-style import graph test: "no package under `internal/domain/*` imports `internal/transport/*`."
- Lint rule: "functions returning error must wrap with `%w` at package boundaries."
- CI check: "every new `internal/<pkg>/` has a `commands.go` or documents why not."
- Cyclic import detection → ArchUnit / `go-cleanarch`.
- Public API surface → golden-file test over generated API dump.

*Web service / backend:*
- Latency budget per endpoint → perf test failing PR if p95 regresses.
- Schema migration safety → backwards-compat check on generated SQL.

*CLI tool:*
- Binary size → CI check, fail on regression beyond N%.
- Startup time → measured in CI against a budget.
- CLI surface golden file → `<binary> --help` recursively captured; unintended changes fail CI.

*Agent-harness repos specifically:*
- Always-loaded file size budget → fail if `AGENTS.md` / `CLAUDE.md` exceeds N lines (token cost is paid every turn).
- Skill catalog integrity → every `skills/<name>/` has an entry in `skills/index.json`.
- Source-vs-installed skill drift → `skills/` and `.claude/skills/` must match (run `make check-skills-sync`).

A constraint with only a guide (prose) will decay; a constraint with only a sensor frustrates the agent into retry loops. Pair them. Sensor output must be LLM-actionable: "wrap with `fmt.Errorf(\"...: %w\", err)` to match project convention" beats `undefined reference`.

If an invariant cannot be mechanically checked, call that out explicitly so reviewers know to look for it.

## Red flags

- The diagram hasn't been updated since a major refactor → agents will follow the stale map. Regenerate or delete.
- Sections describe intent without constraints ("should be decoupled") → unchecked intent is not an invariant.
- Multiple files claim to describe architecture (README, ARCHITECTURE.md, docs/design.md, CLAUDE.md) → collapse to one canonical source; the rest link to it.
- New package added but ARCHITECTURE.md unchanged → either the package is in the wrong place, or the doc is out of date. Both are bugs.
- "We'll refactor later." With agents generating 10× the volume, "later" is 10× worse — add the constraint now, even if imperfect.
- Rules only in a wiki or design doc the agent never loads. If the agent can't see it at prompt time, it doesn't exist.

## Maintenance rule

Update ARCHITECTURE.md in the same PR that changes the architecture. If the PR description says "this also moves X into Y," but the doc doesn't reflect it, the reviewer sends it back. Treat the doc as part of the code.
