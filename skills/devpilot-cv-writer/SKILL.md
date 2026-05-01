---
name: devpilot-cv-writer
description: Use when the user wants to create, write, rewrite, polish, or update a CV / résumé / resume — especially for senior or staff software engineering roles. Triggers on "write my CV", "improve my resume", "rewrite this CV", "review my résumé", "update my resume", "make my CV better", "帮我写简历", "改简历", "优化简历". Produces a quantified, impact-led, one-page CV and refuses to paper over missing data with vague language.
---

# CV Writer (Senior Software Engineering)

## Overview

Good senior-engineer CVs win on **quantified impact, technical judgment, and scope growth** — not on adjectives or technology lists. This skill enforces those standards and, critically, **interviews the user for missing numbers instead of producing confident-sounding vague bullets**.

**Core principle:** No bullet ships without a quantified outcome OR a concrete artifact. If the data isn't there, ask — never paper over with fluff.

## When to Use

- User asks to write, rewrite, improve, polish, update, or review a CV / resume / résumé
- User pastes a CV draft and asks for feedback or edits
- User is preparing for senior / staff / principal software engineering roles
- User wants to "make their CV stronger"

**Do NOT use for:** cover letters, LinkedIn profile rewrites that aren't CV-shaped, recruiter outreach drafts, generic non-engineering CVs.

## The Iron Law

**No bullet without a number or a concrete artifact.**

If the user-supplied draft doesn't contain the data needed to quantify a bullet, you MUST stop and ask. You may NOT:

- Replace the missing number with a strong verb ("scaled", "drove", "led")
- Hedge with vague qualifiers ("significantly", "substantially", "high-traffic")
- Invent plausible numbers
- Keep the bullet unquantified "for now"

Concrete artifact = a named system, a measurable scope (rows, services, users, regions), a public link, or a published outcome. If none exist, the bullet doesn't go in.

## Workflow

1. **Read the input.** Identify which roles/bullets have data and which don't.
2. **Interview for gaps.** Before drafting, ask the user a *batched* set of targeted questions for missing numbers. Examples:
   - "Acme search rewrite — what was p99 latency before/after, and what query volume?"
   - "Payments work at FooBar — TPS handled? incident reduction? revenue impact?"
   - "Monolith → microservices — how many services, over what timeline, with what team size?"
   - "Mentoring — how many engineers, any promoted, any measurable team metric?"
3. **Draft the CV** using the bullet formula and structure below.
4. **Self-review against the red flags list.** Strip anything that fails.
5. **Show the user.** Flag any bullets that are still weak and ask if they want to drop them or supply data.

## Bullet Formula

```
[strong verb] [what you did + how] → [quantified outcome] [optional: why this choice]
```

**Examples:**

❌ "Worked on the search service using Elasticsearch and Go."
✅ "Rewrote search ranking pipeline in Go, cutting index rebuild from 6h → 25min and unblocking daily reindexing for the recommendations team."

❌ "Lead backend development for core platform services."
✅ "Own 4 Go services handling 12k RPS in payments path; reduced p99 320ms → 80ms by replacing per-request DB lookup with a write-through Redis cache."

❌ "Helped with code reviews and mentoring."
✅ "Mentored 3 mid-level engineers; 2 promoted to senior within 18 months. Authored team's Go style guide, now used across 6 services."

✅ With judgment: "Chose Postgres over DynamoDB for the ledger despite higher write load — cross-entity transactions outweighed scale-out cost at our 800-write/sec volume."

## Structure

One page. Two pages **only** if 10+ years and every line earns its place.

1. **Header** — Name, role title, email, GitHub, LinkedIn. No photo, no DOB, no address beyond city.
2. **Summary (optional, ≤3 lines)** — positioning + domain + one flagship achievement. Skip if the experience section already communicates this.
3. **Experience** — ~60% of the page. Reverse chronological. 3–5 bullets per role. Most recent role gets the most bullets.
4. **Selected Projects / Open Source** — only if there's a real link with traction or visible code.
5. **Skills** — grouped (Languages / Infra / Data). **Only list things backed by an experience bullet.** No star ratings. No "Word, Excel, Agile, Scrum, Git".
6. **Education** — one line, unless <3 years out of school.

## Senior-Specific Signals to Surface

A senior CV must show at least 3 of these. If the input lacks them, **ask**.

- **Scope growth across roles** — feature → module → system → org influence
- **Technical judgment** — at least one "chose X over Y because Z" statement
- **Leadership without title** — mentoring with outcomes, standards authored, cross-team alignment
- **Operational ownership** — on-call, incidents resolved, reliability numbers
- **Tradeoffs / failures** — rare but powerful: a rolled-back migration, a rewrite that taught a lesson

## Red Flags — Strip Before Shipping

If any of these appear in your draft, delete and rewrite:

| Red flag | Fix |
|---|---|
| "Passionate", "hardworking", "team player", "synergy", "self-starter" | Delete the sentence |
| "Responsible for X" | Replace with "Did X, resulting in Y" or drop |
| "Helped with", "Worked on", "Assisted in" | Replace with active verb + outcome |
| "Drove", "Led", "Scaled" with no number | Add the number or drop the bullet |
| "High-traffic", "large-scale", "mission-critical" without a metric | Add the metric or drop the adjective |
| Tech listed in Skills but not in any bullet | Remove from Skills |
| Stars / progress bars next to skills | Remove |
| Same generic bullet shape across all 3 roles | Differentiate by scope and outcome |
| Photo, DOB, marital status, full home address | Remove |
| 3+ pages | Cut |

## Common Rationalizations to Refuse

| Excuse | Reality |
|---|---|
| "User didn't give numbers, I'll use strong verbs" | Strong verbs without numbers read as inflated. **Ask the user.** |
| "It would be rude to keep asking" | Batch the questions into one round. One round of questions is normal; vague output is unprofessional. |
| "I'll estimate a reasonable number" | Fabrication. Never invent metrics on a CV. |
| "Adjective conveys the scale" | "High-traffic" means nothing to a hiring manager who saw it on 200 other CVs. |
| "It's a junior role, vague is fine" | Even junior bullets can quantify (rows processed, time saved, tickets closed). |
| "The user said don't ask questions" | You may proceed, but mark every unquantified bullet with `[NEEDS METRIC]` so the user can fill in. Never silently ship vague language. |

## Self-Review Checklist

Before showing the draft:

- [ ] Every experience bullet has a number OR a named concrete artifact
- [ ] No banned adjectives ("passionate", "high-traffic" without metric, etc.)
- [ ] At least one "chose X over Y because Z" judgment line
- [ ] Skills list contains only items used in an experience bullet
- [ ] Scope visibly grows across roles (or is explicitly explained if it didn't)
- [ ] One page (or justified two)
- [ ] Contact info includes GitHub + LinkedIn (or user told you they don't want them)

If a check fails, fix it or ask the user. Do not ship.

## Output Format

When delivering:

1. The CV in markdown.
2. A short "Open questions" section listing anything still missing — quantification gaps, unclear scope, tech without backing experience.
3. A diff-style note (≤5 bullets) of what you changed and why, if rewriting an existing CV.
