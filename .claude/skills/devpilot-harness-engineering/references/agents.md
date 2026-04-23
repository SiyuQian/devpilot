# Writing AGENTS.md

`AGENTS.md` (or `CLAUDE.md`) is injected into every prompt. Every token costs you on every turn, forever. Treat it as the *smallest* possible always-on preamble, not as documentation.

## What belongs in AGENTS.md

Only things that meet **all three** tests:

1. The agent gets it wrong without this line.
2. The fix is too nuanced for a linter to catch.
3. It applies to *most* tasks in the repo, not a narrow corner.

If any test fails, push it into a skill, a sub-agent prompt, or a linter message instead.

## Copy-able skeleton (target: ≤60 lines, hard-split at ~100)

Fill in the bracketed slots; delete sections that don't apply. Each bullet should map to an observed agent mistake or you don't need the bullet.

```markdown
# <Project name>

<One sentence: what the project does and who uses it.>

## Repo map
- `<entry path>/` — <what lives here, 5–10 words>
- `<main internal path>/` — <what lives here>
- `skills/`, `docs/`, `.github/` — <one-line role>

## Build / test / lint
make build    # <output path>
make test     # <what it runs>
make lint     # must pass before commit

## Conventions the agent keeps getting wrong
- <rule 1 in imperative form>
- <rule 2>
- <rule 3–5 max; more = push into skills>

## Pointers (read on demand; do not inline)
- Architecture: ARCHITECTURE.md
- Taste calls: GOLDEN_PRINCIPLES.md
- Harness wiring: docs/harness.md
- Active plans: PLANS.md

## Safety rules the harness can't enforce
<Only non-automatable rules the agent must remember at prompt time.
Prefer hooks / CI / permission settings whenever possible; list a rule
here only if no mechanical check exists. Examples:
  - Never commit without an explicit user ask.
  - Never push to main.>
```

## What does NOT belong

- Long prose rationale — link to a design doc.
- Style rules a linter already enforces — fix the linter, delete the line.
- "If X then Y" branching logic — that's a skill.
- Tool inventories, MCP server lists — let discovery surface those.
- Example code longer than 5 lines — put it in a skill's SKILL.md.
- Safety rules a hook, permission setting, or CI check can enforce — wire it into the harness (`.claude/settings.json`, pre-commit, CI) instead of writing prose the agent has to remember.

## Red flags

- Over 100 lines → time to split into skills.
- Two sections about the same topic → the earlier one didn't work; delete it or rewrite, don't stack.
- "Remember to…" reminders → the agent forgot once; before adding, ask if a sensor (hook, lint) can catch it instead.
- Sections that read like a changelog ("we recently moved to X") → dated; fold into the stable rule or delete.

## Maintenance rule

Every line in AGENTS.md should trace back to an observed agent mistake. If you can't remember which mistake a line prevents, delete it and see if anything breaks.
