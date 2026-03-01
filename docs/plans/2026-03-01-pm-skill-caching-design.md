# PM Skill: Research Caching & Idea Expansion

## Problem

The PM skill runs 3 parallel web research agents every time, even when the same topic was researched earlier the same day. This wastes tokens and time. Additionally, 2-3 feature recommendations is too few for broader ideation.

## Changes

### 1. Research caching (`docs/research/`)

Add a cache layer around Phase 2 (Parallel Deep Research):

- **Before agents run**: Glob for `docs/research/{YYYY-MM-DD}-{topic-slug}.md`
- **Cache hit**: Skip all 3 agents, read the cached file, proceed to Phase 3 synthesis
- **Cache miss**: Run agents as usual, then write combined results to cache file

Topic slug is derived from the product description as lowercase kebab-case (e.g. "CLI developer tools" -> `cli-developer-tools`).

### 2. Cache file format

```markdown
---
topic: "{product_description}"
target_users: "{target_users}"
date: YYYY-MM-DD
---
## Competitor Analysis
{agent 1 output}

## User Pain Points
{agent 2 output}

## Market Trends
{agent 3 output}
```

### 3. Idea count: 5-10

Phase 3 scoring and Phase 4 presentation change from "top 2-3 features" to "top 5-10 features". Scoring dimensions remain the same (Demand, Gap, Trend, Feasibility — each 0-3).

## What doesn't change

- Phase 1 (scope clarification) — unchanged
- Phase 1.5 (rejected ideas check) — unchanged
- Agent prompts — unchanged
- Phase 4.5 (record decisions) — unchanged
- `references/search-patterns.md` — unchanged
