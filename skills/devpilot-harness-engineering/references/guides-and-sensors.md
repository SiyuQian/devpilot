# Guides and Sensors

## The Two Control Types

A harness is a **cybernetic regulator**: it steers the codebase toward a desired state using two complementary control types.

### Guides (Feedforward)

Applied *before* the agent acts. They raise the probability of a correct first attempt.

Examples:
- AGENTS.md / CLAUDE.md conventions
- Skills with ✅/❌ example pairs
- Architecture decision records the agent can read
- Tool descriptions that nudge toward the right choice
- Templates for common task shapes (new endpoint, new migration)

### Sensors (Feedback)

Applied *after* the agent acts. They detect drift and trigger correction — either by the agent itself or by a human.

Examples:
- Linters, type checkers, formatters
- Unit, integration, and fitness tests
- Build / compile steps
- LLM-as-judge review agents
- Runtime observability checks in staging

## The Execution Axis

| | Computational | Inferential |
|---|---|---|
| **Speed** | ms – s | s – min |
| **Cost** | near zero | non-trivial |
| **Determinism** | high | low |
| **Example guide** | Code template generator | Skill with natural-language style guide |
| **Example sensor** | `go vet`, ArchUnit | PR review agent, custom LLM judge |

Use computational controls wherever possible. Reach for inferential controls only where rules can't be expressed mechanically (e.g., "is this API name clear?").

## The Three Regulation Categories

### 1. Maintainability Harness (most mature)

Internal code quality — style, structure, naming, duplication.

| Guide | Sensor |
|---|---|
| Style skill with examples | Language linter, custom rules |
| Naming conventions in AGENTS.md | Name-pattern linter |
| Template for common module layout | Structural test |

### 2. Architecture Fitness Harness (medium maturity)

Performance, module boundaries, dependency hygiene.

| Guide | Sensor |
|---|---|
| Layered architecture doc | ArchUnit-style import checks |
| Latency budget per route | Perf test fails PR on regression |
| Approved-lib list | Dependency whitelist check |

### 3. Behavior Harness (least mature)

Functional correctness. Current practice: AI-generated tests + manual verification. Known gap.

| Guide | Sensor |
|---|---|
| Test-style skill (table-driven, etc.) | Unit + integration tests |
| Approved test fixtures | Mutation testing, coverage deltas |
| Acceptance criteria in the task | E2E in staging |

## Shift Quality Left

Place each control where it's cheapest to fail:

```
pre-commit  →  pre-PR CI  →  post-merge CI  →  staging  →  prod
 fastest                                                 slowest
 cheapest                                                dearest
```

A linter in pre-commit is orders of magnitude cheaper than the same issue caught by a human reviewer; a perf regression caught in CI is orders of magnitude cheaper than one caught in production.

## Making Sensors Agent-Readable

The single biggest upgrade to a sensor for agent use: make the error message *actionable by a model*.

Instead of:
> `error: exported function Foo should have comment`

Write:
> `Exported function 'Foo' is missing a doc comment. Add a comment above the declaration starting with 'Foo ...'. See references/commentary.md for examples.`

The agent now self-corrects on the next turn without human mediation.

## Mutation Testing — The Sensor-of-Sensors

How do you know your sensors actually catch bugs? Mutation testing introduces small faults and checks that tests fail. Agent-authored test suites are especially prone to testing happy paths only; mutation scores expose this.

## Anti-Patterns

- **Guide without a sensor.** Hope is not a harness.
- **Sensor without a guide.** The agent retries blindly, burning tokens.
- **Human-only error text.** Loses the agent-self-correction loop.
- **Inferential control where computational would do.** Slow, expensive, and non-deterministic for no reason.
