# Organizing Agent Context

## Principle

"Giving the agent more context" does not mean "dumping more tokens." It means organizing and exposing the *right* information so the agent can reason over it — the same way you'd onboard a new teammate on product principles, engineering norms, and team culture.

Context has a cost: every token in the system prompt or tool description is paid on every turn. Treat context budget like a performance budget.

## The Context Stack (Progressive Disclosure)

Load cheap, broad context always. Load specialized context on demand.

| Layer | Scope | Loaded when | Size target |
|-------|-------|-------------|-------------|
| `AGENTS.md` / `CLAUDE.md` | Always | Every prompt | ~60 lines (HumanLayer); definitely split before ~100 |
| MCP tool descriptions | Always | Every prompt | Prune aggressively |
| Skills | On demand | Trigger matches task | Full body on invoke |
| Skill references/ | On demand | Agent chooses to read | Heavy detail here |
| Sub-agent reports | On demand | Parent delegates | Condensed, cited |

## AGENTS.md — The Repo Onboarding Doc

A good AGENTS.md answers, in the order an agent needs them:

1. **What is this project?** — one paragraph
2. **Where does code live?** — 3–6 bullets on layout
3. **How do I build / test / lint?** — exact commands
4. **Non-obvious conventions** — the 5–10 things a new hire gets wrong
5. **When to use which skill** — pointers, not copies

Keep it **concise and hand-written**. Auto-generated AGENTS.md files drift and bloat; they become noise the agent learns to ignore.

## Skills — Progressive Disclosure

Package specialized know-how as a skill:

```
skill-name/
  SKILL.md           # frontmatter + short body
  references/        # heavy docs loaded on demand
  scripts/           # executable helpers
```

- Frontmatter `description` is the router — it decides when the skill loads
- Body gives the agent enough to act
- References hold detail the agent fetches only when needed

## Sub-Agents — Context Firewalls

Sub-agents have isolated context windows. Use them when:

- A step would flood the parent's context with intermediate output (large searches, long file reads)
- A step can be done with a cheaper model
- A step is independent and parallelizable

The sub-agent returns a *condensed, cited* summary. The parent never sees the raw transcript. This is the single biggest lever for keeping long agent sessions coherent.

## MCP Tools — Less Is More

Every MCP tool injects its description into the system prompt on every turn. Rules of thumb:

- Remove tools the current agent isn't using
- Prefer one well-scoped tool over three overlapping ones
- Audit tool descriptions for verbosity — each is paid per turn
- Be aware tool descriptions are a prompt-injection surface
- **Prefer CLIs the model already knows over custom MCP wrappers.** `gh`, `git`, `jq`, `rg`, `kubectl` are in the training data; a bespoke MCP tool over the same surface mostly adds context cost (HumanLayer).

## Hooks — Lifecycle Context

Hooks fire on tool use, submit, and stop. Use them to:

- Run typecheck/build after the agent edits code, feeding errors back as context
- Block risky actions before they happen
- Notify humans at handoff points

This turns the IDE/harness itself into a sensor that speaks to the agent.

## Context Rot

Even inside a model's advertised context window, quality degrades as the session grows — a phenomenon often called *context rot*. Sub-agents (context firewalls), summarization, and explicit compaction aren't just token optimizations; they protect reasoning quality in long sessions.

## Anti-Patterns

- **Everything in AGENTS.md.** Bloat → the agent stops reading it carefully.
- **Conditional steering in the top-level doc.** "If you're writing X, do Y" belongs in a skill whose description matches X.
- **No sub-agents, ever.** The parent context fills with search noise and the session degrades.
- **Dumping full tool output.** Let sub-agents summarize with citations.
- **Speculative MCP / skill installs.** Add tools and skills *in response to observed failures*, not preemptively. Every speculative install is permanent context cost for hypothetical benefit.
