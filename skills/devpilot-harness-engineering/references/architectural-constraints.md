# Architectural Constraints from Day One

## Principle

With human teams, you can defer architecture until scaling pain forces the issue. With agents, you cannot: an agent will happily add a tenth way to do the same thing because it cannot feel the cost. Architectural constraints are what let an agent-authored codebase grow fast *without* decaying.

OpenAI's framing (paraphrased): the architecture work a human team would postpone until it grows to hundreds of engineers has to happen on day one with agents — the constraints are what buy speed without decay.

## What to Constrain Early

| Dimension | Example constraint |
|-----------|-------------------|
| Module boundaries | `internal/auth/` may not import from `internal/trello/`; enforce with structural tests |
| Data flow | One canonical path for request → handler → service → repo; no shortcuts |
| Error handling | One error wrapping style; one logging facade |
| Dependency surface | Whitelist of approved libs for HTTP, JSON, testing, logging |
| Naming | Canonical names for common concepts (`Client`, `Service`, `Store`); agent-readable list |
| Test shape | Table-driven, one subtest per row, named `TestFooBar` |
| Public API shape | Functional options, never positional bool flags |

## How to Enforce

Every constraint needs a **guide** (so the agent knows) and a **sensor** (so drift is caught):

- **Guide:** one line in AGENTS.md or a short skill with a ✅/❌ example pair
- **Sensor:** linter rule, custom `go vet` analyzer, ArchUnit-style test, or CI check with LLM-actionable error text

A constraint with only a guide will decay. A constraint with only a sensor frustrates the agent into retry loops. Always pair them.

## Fitness Functions

For architectural characteristics that aren't expressible as a lint rule:

- Latency budget per endpoint → perf test failing PR if p95 regresses
- Binary size → CI check
- Cyclic import detection → ArchUnit / `go-cleanarch`
- Public API surface → golden-file test over generated API dump

## Anti-Patterns

- **"We'll refactor later."** With agents generating 10× the volume, "later" is 10× worse.
- **Rules only in a wiki.** If the agent can't see it at prompt time, it doesn't exist.
- **Sensors that only humans can read.** `undefined reference` is not actionable to a model; "wrap with `fmt.Errorf(\"...: %w\", err)` to match project convention" is.
