# Writing PLANS.md (and `docs/exec-plans/`)

`PLANS.md` is the index. `docs/exec-plans/{active,completed}/` holds the actual plans. Together they answer three questions for an agent about to pick up work:

1. What are we currently trying to ship?
2. For a given active plan, what's the next concrete step?
3. What have we *decided not* to do, and why?

## Two artifacts, two audiences

| Artifact | Audience | Lifetime | Shape |
|---|---|---|---|
| `PLANS.md` (root) | Any agent or human landing in the repo | Long — months | Flat index: one line per active plan linking to `docs/exec-plans/active/…` |
| `docs/exec-plans/active/<plan>.md` | An agent executing *this* plan | Short — days to weeks | Design + numbered task list + status |
| `docs/exec-plans/completed/<plan>.md` | Future agents investigating "why did we build it this way" | Permanent archive | Same file, moved when done |

## PLANS.md (index) format

```markdown
# Plans

Active work, newest first. Each entry: status, one-line goal, link.

- **[active] agent-friendly-repo** — retrofit AGENTS.md + skills + sensors onto existing repo. → `docs/exec-plans/active/2026-04-20-agent-friendly-repo.md`
- **[blocked] stream-compaction** — rewrite context compaction to handle multi-tool turns. → `docs/exec-plans/active/2026-04-15-stream-compaction.md`

## Recently completed
- **2026-04-10 skills-index** — unified skill catalog. → `docs/exec-plans/completed/…`

## Tech debt tracker
See `docs/exec-plans/tech-debt-tracker.md` for items parked but not scheduled.
```

Keep it skimmable. If it grows past one screen, it's because you're tracking individual tasks here instead of linking out — stop.

## Exec plan file format

```markdown
# <Plan title>

**Status:** active | blocked | completed
**Owner:** <human / agent>
**Started:** YYYY-MM-DD

## Goal
One paragraph. What "done" looks like, in user-visible terms.

## Non-goals
Bullet list. Things reviewers will ask about but we are NOT doing in this plan.

## Design
Enough design to make the task list unambiguous. Link out to a pair-doc in
`docs/design-docs/` if the design is large. Do not inline 500 lines here.

## Tasks
- [x] 1. <concrete step with a verification criterion>
- [ ] 2. <next step>
- [ ] 3. …

Each task should be sized so one agent can finish it in one context window and
produce one reviewable PR. Tasks must be depth-first — finish one before starting
the next; wide-shallow lists ("stub all 12 endpoints") defeat the harness. If a
task won't fit, split it. See `depth-first-decomposition.md` for per-task block
shape (Scope/Interfaces/Acceptance/Links/DoD) and sizing ceilings.

## Decisions log
Dated bullets for load-bearing choices made during execution.
- 2026-04-22 — chose X over Y because Z.

## Open questions
Things the agent should ask a human about before proceeding.
```

## OpenSpec vs plain exec-plan — which to use

Both cover "design + tasks + execution." Pick by scope.

**Use OpenSpec** (`openspec-propose`, `openspec-apply-change`, `devpilot-auto-feature`) when:
- The change modifies or adds a **spec** (user-facing behavior contract)
- You want deltas versioned alongside code (`openspec/changes/<id>/`)
- Spec sync + archival at the end matters (multiple agents will read this contract later)

**Use a plain exec-plan** (`docs/exec-plans/active/…`) when:
- Work has no spec surface — infrastructure, tooling, CI, docs, refactors
- Scope is exploratory and may be abandoned
- It's a one-off where spec-tracking is overhead

If in doubt: start with a plain exec-plan; promote to OpenSpec when the scope solidifies and touches user-visible behavior.

## What does NOT belong

- Brainstorming / exploration — put it in `docs/design-docs/` and link from the plan's Design section.
- Meeting notes, status updates for humans — use your actual project tracker.
- Long prose rationale inside task bullets — push to Decisions log or design doc.
- Plans without a verification criterion per task — the agent will declare "done" on vibes.

## Lifecycle

1. **Create** — `docs/exec-plans/active/YYYY-MM-DD-<slug>.md`; add one line to `PLANS.md`.
2. **Execute** — agent ticks off tasks, appends to Decisions log as it goes.
3. **Complete** — move the file to `docs/exec-plans/completed/`, move the `PLANS.md` entry into the "Recently completed" section, then prune that section periodically.
4. **Abandon** — same as complete, but add a final Decisions-log entry explaining *why* it stopped. Future agents will thank you.

## Red flags

- Active plans with no updates for 3+ weeks → blocked or dead; mark or archive.
- Tasks without checkboxes, or checkboxes without commits → status is drifting from reality.
- Plan file edited but `PLANS.md` not updated → index will rot; fix in the same PR.
- More plans "active" than there are people (or agents) working → you're using this as a wishlist, not a work queue. Move the rest to the tech-debt tracker.

## Maintenance rule

A plan is load-bearing context for the agent executing it. Stale plans mislead worse than missing plans. When in doubt: archive, don't leave half-true.
