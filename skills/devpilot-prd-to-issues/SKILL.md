---
name: devpilot-prd-to-issues
description: Use when the user wants to turn a PRD, spec, design doc, or feature brief into a set of GitHub issues — "break this PRD into tickets", "create issues from the spec", "split this into tasks", "file the work for this feature", "decompose into deliverables", "/prd-to-issues". Produces an issue tree where every ticket is a deliverable slice with explicit parent/child/blocks relationships and a bounded change size.
---

# PRD → GitHub Issues

## Overview

A PRD is one document. Engineering needs a graph of small, mergeable, individually-shippable issues. This skill decomposes a PRD into that graph: every issue is a **deliverable slice** (mergeable on its own, demonstrable to a stakeholder), every issue is a **reasonable change size** (≤ ~400 LOC diff or ≤ 2 dev-days), and every issue **declares its relationships** to others (epic / parent / blocks / blocked-by / related).

**Core principle:** an issue that ships nothing on its own, or that bundles unrelated work, is the wrong unit. Split until each ticket is independently mergeable; group with relationships, not scope creep.

## When NOT to Use

- The PRD is one paragraph and the work is one PR → just open one issue, don't decompose.
- Bug triage from a scan → use `devpilot-scanning-repos` (already issue-shaped) or `devpilot-issue-triage`.
- Roadmap-level epics with no spec yet → use `devpilot-pm` / `devpilot-product-research` first; don't manufacture sub-tickets from vapor.
- Project tracker is Linear / Jira / Trello, not GitHub → adapt the relationship model but use the appropriate tool (`devpilot-trello` for Trello).

## Workflow

1. **Ingest the PRD.** Read the source (file, URL, pasted text). Extract: goal, user-visible outcomes, non-goals, constraints (deadlines, dependencies, platforms), and any acceptance criteria already written. If goal or outcomes are missing, **stop and ask** — do not invent scope.
2. **Confirm target repo and conventions.** `gh repo view --json nameWithOwner`. Snapshot existing labels: `gh label list --limit 200 --json name,description`. Check for an existing epic / parent-issue convention (e.g. `epic`, `tracking`, task-list checkboxes in a parent issue, GitHub sub-issues, or Projects v2 hierarchy). Reuse what the repo already does — do **not** introduce a new relationship model.
3. **Draft the work breakdown (no issues filed yet).** Produce a single markdown plan in the conversation with this shape:

   ```
   Epic: <PRD title>
   ├── #A1 <slice>            [parent: epic] [blocks: A2, A3]
   ├── #A2 <slice>            [parent: epic] [blocked-by: A1]
   ├── #A3 <slice>            [parent: epic] [blocked-by: A1] [related: B1]
   └── #B1 <slice>            [parent: epic]
   ```
   Each node has: provisional ID, one-line title, est. size (S/M/L = ½ day / 1 day / 2 days), and explicit relationships. **Show this to the user and get explicit "go" before filing anything.** Iterating on paper is cheap; iterating on filed issues is not.
4. **Apply the size-and-shape rules to every node.** See "Slice rules" below. If any ticket fails the rules, split or merge before filing. Re-show the tree if it changed materially.
5. **File issues bottom-up.** Create leaf issues first so parents can reference real numbers in their task lists. For each issue use the body template below, the **canonical title format**, and the **canonical label set** (see "Title & label conventions"). Reconcile against the step-2 label snapshot — reuse exact matches, rename non-canonical semantic matches via `gh label edit`, only create labels that have no match. Do not file any issue that is missing the title prefix or any of the four required labels.
6. **Wire relationships using the repo's convention.** In order of preference: (a) GitHub native sub-issues (`gh api … /sub_issues`) if the repo uses them; (b) task-list checkboxes in the epic (`- [ ] #123`) which GitHub auto-tracks; (c) `Blocks:` / `Blocked by:` / `Parent:` / `Related:` lines in each issue body, with `#N` cross-links. Pick one and apply it consistently across the whole tree.
7. **Post the summary.** Print a compact table to the user: `ID | title | size | parent | blocks | blocked-by | URL`. Confirm the epic's task list renders correctly (open it in `gh browse`) before declaring done.

## Title & label conventions

Every issue filed by this skill MUST use the same title shape and the same four-label spine, so the whole tree is filterable in one query and humans can read the relationship at a glance.

### Title format

```
[<feature-slug>] <type>: <imperative one-liner>
```

- `<feature-slug>` — short kebab-case identifier for the PRD, identical across every issue in the tree (e.g. `oauth-rotation`, `mobile-onboarding`). Pick once in step 3, reuse everywhere. This is what makes `is:issue [oauth-rotation]` return the whole tree.
- `<type>` — one of: `epic`, `feat`, `chore`, `infra`, `docs`, `test`, `spike`. Exactly one. The epic issue uses `epic`; everything else picks the most accurate non-epic type.
- `<imperative one-liner>` — ≤ 70 chars, starts with a verb (`Add`, `Wire`, `Migrate`, `Expose`), no trailing period, no ticket numbers, no emoji.

Examples:
- `[oauth-rotation] epic: Rotate OAuth refresh tokens daily`
- `[oauth-rotation] feat: Add token-rotation worker behind flag`
- `[oauth-rotation] chore: Backfill rotation_at column`

If a node can't be expressed in this format, the slice is probably wrong-shaped — fix the slice before fighting the title.

### Label spine (exactly four, every issue)

| Slot | Allowed values | Purpose |
|---|---|---|
| `feature:<slug>` | one per PRD, matches the title slug | Groups the whole tree |
| `type:<kind>` | `type:epic`, `type:feat`, `type:chore`, `type:infra`, `type:docs`, `type:test`, `type:spike` | Mirrors the title `<type>` |
| `size:<bucket>` | `size:S`, `size:M`, `size:L` | ≈ ½d / 1d / 2d; matches the body's Size field |
| `area:<top-level-dir-or-surface>` | e.g. `area:api`, `area:web`, `area:cli` | Routes to the right owner |

Optional fifth labels (apply only when meaningful, never as filler):
- `priority:<p0|p1|p2>` — apply ONLY when the PRD or the user explicitly states a priority for that slice. Do not invent a default. If unsure, omit the label.
- `blocked` — set automatically when the issue has open `Blocked by` issues; clear when they close.
- `good-first-issue` — only when the slice is genuinely standalone, well-scoped, and a newcomer could pick it up.

Repo-specific meta labels that are orthogonal to the spine (e.g. `tracked-by-roadmap`, `needs-design`, `tech-debt`) MAY be applied on top when they apply, but are never substitutes for any of the four spine slots.

**Reconciliation rule (same as `devpilot-scanning-repos`):**
- Exact match in the repo → reuse as-is.
- Semantic match with a non-canonical name (e.g. repo has `feature` for `type:feat`, or `service/api` for `area:api`) → `gh label edit "<old>" --name "<canonical>" --description "<canonical-desc>"`. Renaming preserves all existing issue associations.
- Too generic / wrong intent / orthogonal to this PRD (e.g. `bug` for a feature PRD, `tech-debt`, `needs-design`, `tracked-by-roadmap`) → **leave alone**, do not force a mapping. Create the canonical label fresh if the spine slot needs filling.
- **No canonical bucket exists** (e.g. repo has `XS`/`XL` size labels but the spine only defines `size:S|M|L`) → leave alone. Do NOT rename to the closest bucket — that lies about the size budget on existing issues. New tickets from this PRD use the canonical S/M/L; legacy XS/XL remain on legacy issues.
- **Layer vs. surface for `area:*`:** prefer surface labels (`area:api`, `area:web`, `area:ios`, `area:cli`, `area:worker`) that route to an owner. A repo label like `frontend` / `backend` is borderline — rename only when it consistently maps to one surface in this repo; if it's a cross-surface layer label, leave alone and create the surface label fresh.

**Confirmation gate:** if the reconciliation plan would rename **more than 3** labels, OR rename any label currently used on more than 50 issues, show the plan to the user and get explicit "go" before running any `gh label edit`. Bulk renames touch unrelated existing issues' triage queries.

**Side-effect warning:** renaming a generic label (e.g. `epic` → `type:epic`) retroactively co-tags every issue currently carrying it. Call this out in the reconciliation summary: `epic → type:epic (also tags 5 pre-existing epic issues)` so the user can veto before the rename runs.

Print a reconciliation summary before filing: `N reused, R renamed (old → new, with affected-issue counts), M created, K left alone (with reason)`.

### Title ⇔ label invariants

These must hold for every issue, on every run. Verify before declaring done:

1. The `<feature-slug>` in the title equals the value of the `feature:<slug>` label.
2. The `<type>` in the title equals the value of the `type:<kind>` label.
3. The Size field in the body equals the `size:<bucket>` label.
4. Exactly one issue **in this PRD's tree** has `type:epic` and no `Parent:` line. (The repo may have other `type:epic` issues from prior PRDs — that's fine; the invariant is per-tree, not per-repo.)

If any invariant fails, the title and labels are out of sync — fix before continuing. Asymmetric metadata is the same bug class as asymmetric `Blocks` / `Blocked by`.

## Slice rules

Every leaf issue MUST satisfy all four:

| Rule | What it means | If it fails |
|---|---|---|
| **Deliverable** | Merging this issue alone leaves `main` better than before — feature-flagged if needed, but not a half-wired stub that breaks build or UX. | Either combine with the issue that completes it, or add a flag/scaffold so this one ships meaningfully. |
| **Bounded** | ≤ ~400 lines diff OR ≤ 2 dev-days. Hard ceiling: 800 LOC / 3 days. Tests count toward the budget; generated code does not. | Split along a natural seam: data-model vs. handler vs. UI; happy-path vs. error-paths; one entity at a time. |
| **Independently testable** | Has its own acceptance criteria the reviewer can verify without merging siblings. | Pull the verification surface (test fixtures, a CLI flag, a stub UI page) into this ticket. |
| **Single-owner-friendly** | One engineer can pick it up without coordinating mid-flight with another open ticket in the tree. | If two tickets must land in the same PR to be useful, they're one ticket. Merge them. |

If a node can't be made deliverable on its own, the right move is usually a **scaffolding issue** first (introduce the interface / flag / migration) so subsequent slices have a place to land.

## Relationship taxonomy

Every issue body declares relationships explicitly. Use exactly these four kinds — don't invent more:

- **Parent** — the epic / tracking issue this rolls up to. Exactly one.
- **Blocks** — issues that cannot start (or cannot ship) until this one merges.
- **Blocked by** — the inverse; issues this one waits on. Symmetric with `Blocks:` on the other side.
- **Related** — useful context, not a dependency. Use sparingly; if everything is "related" the field is noise.

Every `Blocks` on issue A MUST appear as `Blocked by` on the named issue. Asymmetric edges are bugs in the breakdown — fix before filing.

## Issue body template

```markdown
## Context
<2–4 sentences: where this fits in the PRD, what problem this slice solves. Link the PRD.>

## Scope
- <bullet 1, concrete>
- <bullet 2, concrete>
- <bullet 3, concrete>

## Out of scope
- <thing a reviewer might assume but we are NOT doing here; usually points to a sibling issue>

## Acceptance criteria
- [ ] <observable behaviour 1>
- [ ] <observable behaviour 2>
- [ ] <test / verification step>

## Size
**S | M | L** (≈ ½d / 1d / 2d, ≤ ~400 LOC diff)

## Relationships
- **Parent:** #<epic>
- **Blocks:** #<id>, #<id>
- **Blocked by:** #<id>
- **Related:** #<id>

## PRD reference
<link or path, plus the section heading this slice maps to>
```

The epic issue gets the same template plus a top-level task list of every child:

```markdown
## Children
- [ ] #A1 …
- [ ] #A2 …
```

GitHub auto-renders this list with checkboxes that flip when the child closes — that's the cheapest progress dashboard available. Use it.

## Quick reference

| Situation | Move |
|---|---|
| Slice is 1500 LOC | Split along data-model / handler / UI seam |
| Slice "has to ship with" the next one | They're one slice — merge them |
| Slice can't be merged without breaking main | Add a feature flag, OR push the user-visible part to a later ticket |
| Two slices both need the same new helper | Extract a scaffolding ticket; both depend on it |
| PRD has 30 leaf items | Group into 3–5 chapters under the epic; chapters are intermediate parents |
| Mixed repos (frontend + backend) | One epic per repo, link them via `Related:` across the boundary |

## Common mistakes

- **Filing issues before the user signs off on the tree.** Iterating on a markdown plan is free. Iterating on 14 already-filed issues with cross-links is not.
- **Inventing a relationship model the repo doesn't use.** If the repo uses task-list checkboxes, don't introduce sub-issues; if it uses sub-issues, don't fall back to plaintext `Parent:` lines. Pick what's already there.
- **"Phase 1 / Phase 2" tickets that bundle unrelated work.** A phase isn't a slice. Each phase decomposes into its own real tickets.
- **Bulk-renaming high-volume labels without confirmation.** Renaming `S`/`M`/`L` when ~80 existing issues carry them, or `service/api` when ~200 do, mass-mutates triage queries the team relies on. Hit the >3-renames OR >50-affected-issues threshold → confirm first.
- **Forcing every existing label into the spine.** Orthogonal labels (`bug` on a feature PRD, `tracked-by-roadmap`, `needs-design`) get left alone, not bent into a `type:*` they don't fit.
- **Renaming `XS`/`XL` to `size:S`/`size:L`.** The spine has no XS/XL bucket and renaming lies about size budget. Leave them alone.
- **Inventing `priority:*` defaults.** Priority labels appear only when the PRD or user states one. "I think this is p1" is filler.
- **Inconsistent titles or labels across the tree.** Half the issues say `[oauth-rotation]`, half say `oauth rotation —`; some carry `type:feat`, some don't. The whole point of the convention is one filter returns the whole tree. If you wouldn't bet $20 on `is:issue label:feature:<slug>` returning every child, the run failed.
- **Asymmetric relationships.** `A blocks B` on A's body but nothing on B's body. Reviewers can't see the dependency from B. Always wire both sides.
- **Tickets with no acceptance criteria** ("implement auth"). The reviewer has nothing to verify against. If the PRD doesn't give criteria for a slice, write them and confirm with the user before filing.
- **Renaming the PRD as the epic title and shoving everything under one ticket.** That's not decomposition; that's a rebrand. The epic exists to *track* the children, not to *contain* the work.
- **Breaking research / spike work into the same tree.** Spikes don't ship. File them separately or close them when their output (a doc, a decision) lands; don't put them in the deliverable graph.
- **Skipping the dedupe step against existing open issues.** Before filing, `gh issue list --search '<keywords>' --state open` and reuse / link to anything that already covers a slice.

## Acceptance criteria for a correct run

A correct invocation of this skill produces:

1. A markdown work-breakdown shown to the user **before** any issue is filed, and explicit user confirmation captured.
2. Exactly one epic / tracking issue, with a task list of every child issue that GitHub auto-tracks.
3. Every child issue: ≤ ~400 LOC estimated diff, deliverable on its own (possibly behind a flag), with its own acceptance criteria.
4. Every relationship is **symmetric** — `Blocks` on one side ⇔ `Blocked by` on the other.
5. Every issue title matches `[<feature-slug>] <type>: <imperative one-liner>` with the same `<feature-slug>` across the whole tree.
6. Every issue carries the four-label spine: `feature:<slug>`, `type:<kind>`, `size:<bucket>`, `area:<surface>`. Title `<type>` and `<feature-slug>` agree with the matching labels. Existing repo labels reused or renamed to canonical; no orphan label families introduced.
7. A final summary table printed: ID, title, size, parent, blocks, blocked-by, URL.

If any of these is missing, the skill failed — stop and correct before continuing.
