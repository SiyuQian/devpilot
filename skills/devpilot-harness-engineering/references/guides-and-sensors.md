# Guides and Sensors

Use this when deciding how to enforce a rule: where the guide goes, what sensor to pair it with, and at which stage to run it.

## The pairing rule

Every rule must have both:

- **Guide** — steers the agent *before* it acts (AGENTS.md line, skill with good/bad pair, template, tool description).
- **Sensor** — detects drift *after* it acts (linter, test, fitness function, type check, review agent).

Rule with only a guide → decays silently. Rule with only a sensor → agent retries blindly and burns tokens. If you cannot add both, postpone adding the rule.

## Picking the right sensor — decision table

| If the rule is about… | Use a sensor of type… | Run it at… |
|---|---|---|
| Syntax, formatting, imports | Linter / formatter | pre-commit |
| Naming or structural patterns | Custom linter / AST check | pre-commit |
| Module boundaries, cyclic imports | ArchUnit-style test | pre-PR CI |
| Error wrapping / logging discipline | Custom `go vet` analyzer or lint rule | pre-commit |
| Type / API shape | Type checker, golden-file API dump test | pre-PR CI |
| Latency / binary size / resource budget | Perf test with threshold | pre-PR CI |
| Functional correctness | Unit + integration tests | pre-PR CI |
| Subjective quality ("is this name clear?") | LLM-judge review agent | post-author, pre-merge |
| Runtime invariants ("token never logged") | Observability assertion | staging |

Default to the **cheapest** sensor that catches the rule. Promote to an inferential (LLM) sensor only when no mechanical check exists.

## Placing the sensor — shift-left ladder

```
pre-commit  →  pre-PR CI  →  post-merge CI  →  staging  →  prod
  cheap                                                    expensive
```

For each sensor, pick the leftmost stage that has the required signal. A rule that *could* run in pre-commit but runs in CI wastes an agent iteration per violation.

## Guide + sensor recipes by category

Pick the row that matches the rule type; use both columns.

### Maintainability (style, structure, duplication)

| Guide | Sensor |
|---|---|
| Style skill with one good/bad example pair | Language linter with custom rules |
| Naming conventions as one line in AGENTS.md | Name-pattern linter |
| Module-layout template in a skill | Structural test asserting directory contract |

### Architecture fitness (boundaries, budgets, deps)

| Guide | Sensor |
|---|---|
| `ARCHITECTURE.md` invariant | ArchUnit-style import check |
| Latency budget line per route in code comment | Perf test failing PR on p95 regression |
| Approved-lib list in AGENTS.md | Dependency whitelist check in CI |

### Behavior (functional correctness)

| Guide | Sensor |
|---|---|
| Test-style skill (table-driven, naming) | Unit + integration tests |
| Acceptance criteria in the task block | E2E in staging |
| Approved fixtures / factories | Mutation testing, coverage delta |

## Writing agent-actionable sensor output

This is the single biggest upgrade. When you add or modify a sensor, rewrite its error message to be a fix instruction, not a diagnosis.

Pattern: **`<what is wrong>. <exact change to make>. <where to look for examples>.`**

| Human-only (bad) | Agent-actionable (good) |
|---|---|
| `error: exported function Foo should have comment` | `Exported function 'Foo' is missing a doc comment. Add a comment above the declaration starting with 'Foo '. See references/commentary.md.` |
| `undefined reference: wrapError` | `Wrap returned errors with fmt.Errorf("doing X: %w", err). See AGENTS.md § Conventions.` |
| `test failed: expected 200 got 500` | `Handler returned 500 on valid input. Check <file>:<line>. Expected: 200 with {id,status} per acceptance criteria.` |

If you can't rewrite the message at the sensor, wrap it: have CI post-process the output into the actionable form before feeding it back to the agent.

## Self-check before adding a new rule

- [ ] Guide is written *and* reachable in the agent's normal loading path (AGENTS.md, a skill description that matches real triggers, or a tool description).
- [ ] Sensor exists at the cheapest stage that can run it.
- [ ] Sensor message tells the agent what to change, not just what broke.
- [ ] Rule does not duplicate an existing one (grep first).
- [ ] You can name the observed mistake that motivated this rule (Hashimoto's law).

Any "no" → don't ship the rule yet.

## Red flags on existing rules

- Same rule violated across unrelated PRs → guide is not reaching the agent; move it up the loading path.
- Sensor fires but agent can't self-correct → message is human-only; rewrite it.
- Inferential sensor (LLM judge) where a lint would do → remove; it's slow, costly, and non-deterministic.
- Rule with a guide but no sensor → schedule removal or add the sensor.
- Mutation score low on critical paths → the sensors themselves are under-tested; AI-written tests often cover happy paths only.
