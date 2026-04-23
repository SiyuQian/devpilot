# Golden Principles and Garbage-Collection Refactoring

## Principle

At agent scale, quality doesn't fail from one bad PR — it fails from **compounding drift**: every PR is individually reasonable, but the cumulative codebase becomes inconsistent. The fix, per OpenAI's harness experiment, is to:

1. Encode the team's non-negotiable tastes as **"golden principles"** in the repository
2. Run **recurring background agent tasks** that scan for deviations, grade quality, and open small, targeted refactor PRs
3. Auto-merge the refactor PRs (most are < 1 minute of human review)

This is **garbage collection for code quality**. Individual PRs stay fast; a background loop keeps the heap tidy.

## Golden Principles — What They Are

Not style rules (those go in linters). Not conventions (those go in AGENTS.md). Golden principles are the **opinions** that distinguish this codebase from a generic one:

- "Prefer functional options over config structs for constructors."
- "All external calls go through a single client with retry + metrics — no raw `http.Get`."
- "Errors are wrapped at every layer boundary with `%w` and context."
- "No mocks for our own packages; use the real thing in tests."
- "Every public function has a one-sentence doc comment explaining *why it exists*, not what it does."

Characteristics:
- Opinionated (another team could reasonably do it differently)
- Finite — small enough that the agent can hold the whole list in mind (dozens, not hundreds)
- Expressible as a before/after example
- Detectable by either a linter, a test, or an LLM judge

Store them in a single file (`GOLDEN_PRINCIPLES.md` or similar) that both humans and the GC agent read.

## The GC Loop

A background agent runs on a regular cadence (pick whatever matches your merge rate) and does:

```
for each principle P in GOLDEN_PRINCIPLES.md:
    scan recent changes for deviations from P
    grade the deviation (severity, scope, blast radius)
    if fixable mechanically:
        open a small PR with the fix
        link to P in the description
        let CI verify; auto-merge on green
    else:
        open an issue with examples, tagged for human review
```

Key properties:

- **Small PRs.** One principle, one area, one diff. Reviewable in seconds.
- **Auto-merge on green.** The sensor (CI) is the gate, not a human.
- **Cited principle.** Every PR description points to the principle and a before/after example.
- **Grade trend.** Track a score per principle over time; regressions are the early-warning signal.

## How It Differs From a Linter

A linter catches a rule *at the moment of commit*. The GC loop:

- Catches principles that can't be expressed as a deterministic rule (judgement-based)
- Sweeps the *whole repo*, not just the diff — so it finds historical drift
- Can refactor, not just flag — most findings are already fixed when the human sees them
- Evolves: new principles get added, the loop re-sweeps, the codebase catches up

A linter and a GC agent are complements, not substitutes.

## Principle Lifecycle

1. **Notice a recurring taste call** in human review comments (e.g., "we prefer X here")
2. **Promote it**: add to `GOLDEN_PRINCIPLES.md` with a before/after example
3. **Decide the enforcement layer**:
   - Mechanical? → add a linter rule (sensor shifts left)
   - Judgement? → add to the GC loop
4. **Sweep**: let the GC agent fix the backlog
5. **Monitor**: the principle's grade should rise and stay high; if it regresses, investigate the root cause

## Grading and Observability

Assign each principle a grade (e.g., A–F or a 0–100 score) based on the ratio of compliant vs. deviant instances. Surface the grades in a repo dashboard or a pinned issue. A regressing grade is the cheapest possible early warning that something in the context stack (AGENTS.md, skills, guides) is no longer steering the agent correctly.

## Integration With the Rest of the Harness

| Signal | Action |
|---|---|
| Principle grade rising | Guides + sensors working |
| One principle regressing | Update its guide; strengthen its sensor |
| Many principles regressing | The broader context (AGENTS.md, skills) is stale or too long |
| GC agent keeps opening the same fix | Promote the principle to a hard linter rule |

## Anti-Patterns

- **Principles nobody enforces.** They become aspirational folklore; drift continues.
- **Giant refactor PRs from the GC loop.** Makes review costly; break them up by principle and area.
- **Auto-merge without sensors.** The GC loop is only safe when CI is trusted.
- **Too many principles.** Once the list stops fitting comfortably in context, the agent starts skimming it; prune, or split into domain-scoped files loaded via skills.
- **Principles that are actually style rules.** If a linter can express it, use the linter — the GC loop is for taste and structure, not spacing.

## The Compounding Payoff

The GC loop is what makes agent-speed sustainable. Without it, every week of high-throughput agent work leaves a week of human cleanup behind. With it, the cleanup happens in the background, in minutes-of-review chunks, and the codebase *converges* toward the team's tastes instead of drifting away from them.
