---
name: developerkit:pm
description: Product manager skill for market research and feature discovery. Use when the user wants to research market needs, analyze competitors, discover user pain points, or prioritize features for a product. Triggers on /pm, "market research", "find features", "what should we build", "产品调研", "市场需求".
---

# Product Discovery & Market Research

A product manager skill that discovers market needs through parallel deep research. It runs 3 specialized agents concurrently — competitor analysis, user pain point mining, and trend tracking — then synthesizes findings into 2-3 prioritized feature recommendations.

## Process

### Phase 1: Scope Clarification

Before launching research, ask the user **one question at a time** to establish:

1. **Product direction** — What are you building? (If not provided as argument)
2. **Target users** — Who is this for?

Keep it to 1-2 questions max. Extract search keywords from the answers for the agents.

### Phase 1.5: Load Previously Rejected Ideas

Before launching research, check for previously rejected or deferred ideas:

1. Use the Glob tool to scan `docs/rejected/*.md` (skip `README.md`)
2. Read each file's YAML frontmatter to extract: `idea`, `status`, `reason`
3. Build two lists:
   - **Rejected list**: Ideas with `status: rejected` — these MUST be excluded from recommendations
   - **Deferred list**: Ideas with `status: deferred` — these may only be re-suggested if there is strong new evidence of changed market conditions

If no files exist in `docs/rejected/`, skip this phase and proceed to Phase 2.

Store these lists for use in Phase 2 agent prompts.

### Phase 2: Parallel Deep Research

Launch **3 agents in parallel** using the Task tool with `subagent_type: "general-purpose"`. Each agent uses WebSearch extensively.

**IMPORTANT**: Launch all 3 agents in a single message to maximize parallelism.

**CRITICAL**: Each agent prompt MUST end with this constraint:
> Keep your response under 800 words. Return ONLY a structured summary — no preamble, no methodology explanation. Focus on actionable findings with specific data points.

**CRITICAL**: After ALL agents return, you MUST immediately proceed to Phase 3 (Synthesis) in the SAME response. Do NOT stop or wait for user input between Phase 2 and Phase 4. The flow is: agents return → synthesize → present to user, all in one response.

#### Agent 1: Competitor Analyst

Prompt template:
```
You are a competitor analyst. Research the competitive landscape for: {product_description}

Target users: {target_users}

Do the following:
1. Use WebSearch to find 5-8 competing products. Search for:
   - "{product_type} alternatives 2026"
   - "{product_type} best tools"
   - "{product_type} comparison"
2. For each competitor, search for their features and pricing:
   - "{competitor_name} features pricing"
   - "{competitor_name} review"
3. Compile a competitor analysis table with columns:
   - Name | Core Features | Pricing | Strengths | Weaknesses | Popularity

Return ONLY the structured analysis. Be specific with facts and data.

Keep your response under 800 words. Return ONLY a structured summary — no preamble, no methodology explanation. Focus on actionable findings with specific data points.
```

#### Agent 2: User Pain Analyst

Prompt template:
```
You are a user pain point researcher. Find real user complaints and unmet needs for: {product_description}

Target users: {target_users}

Do the following:
1. Use WebSearch to find user discussions. Search for:
   - "{product_type} pain points reddit"
   - "{product_type} frustrating"
   - "{product_type} missing features"
   - "{product_type} wish list forum"
   - "{product_type} complaints"
2. For each pain point found, note:
   - What the pain point is
   - How frequently it's mentioned (high/medium/low)
   - Source (Reddit, forum, review site, etc.)
   - Whether any existing product has solved it

Return a structured list of pain points ranked by frequency. Be specific — quote real user feedback when possible.

Keep your response under 800 words. Return ONLY a structured summary — no preamble, no methodology explanation. Focus on actionable findings with specific data points.
```

#### Agent 3: Trend Analyst

Prompt template:
```
You are a technology and market trend analyst. Research industry trends for: {product_description}

Target users: {target_users}

Do the following:
1. Use WebSearch to find trends. Search for:
   - "{product_type} trends 2026"
   - "{product_type} emerging technology"
   - "{product_type} market growth"
   - "{product_type} future"
   - "{related_technology} adoption rate"
2. For each trend, note:
   - Trend direction
   - Maturity stage (early/growing/mature)
   - Related technologies
   - Potential opportunity for a new product

Return a structured trend analysis. Focus on actionable insights, not hype.

Keep your response under 800 words. Return ONLY a structured summary — no preamble, no methodology explanation. Focus on actionable findings with specific data points.
```

### Phase 3: Synthesis (MUST happen in the same response as Phase 2 results)

**CRITICAL**: Do NOT stop after agents return. Immediately synthesize and present findings in a single response. Never wait for user input between Phase 2 and Phase 4.

After all 3 agents return, synthesize their findings:

1. **Cross-validate** — Needs mentioned across multiple agents get higher weight:
   - Appears in competitor gaps AND user pain points → strong signal
   - Appears in user pain points AND aligns with trends → strong signal
   - Appears in all three → highest priority

2. **Score each potential feature** on 4 dimensions:
   - **Market demand** (0-3): How many users want this?
   - **Competitive gap** (0-3): How poorly do competitors serve this?
   - **Trend alignment** (0-3): Does this match where the market is going?
   - **Feasibility** (0-3): How realistic is this to build?

3. **Select top 2-3 features** by total score.

### Phase 4: Collaborative Decision

Present findings to the user in this format:

```markdown
## Market Research Results: {product_description}

### Key Findings

**Competitive Landscape**: {1-2 sentence summary}
**User Pain Points**: {1-2 sentence summary}
**Market Trends**: {1-2 sentence summary}

### Recommended Features

#### 1. {Feature Name} — Priority: HIGH
- **What**: {feature description}
- **Why**: {market evidence from agents}
- **Competitors**: {who does/doesn't have this}
- **Score**: Demand {x}/3 | Gap {x}/3 | Trend {x}/3 | Feasibility {x}/3 = {total}/12

#### 2. {Feature Name} — Priority: {HIGH/MEDIUM}
...

#### 3. {Feature Name} — Priority: {MEDIUM}
...

### What should we build first?
```

Then engage in discussion:
- Answer follow-up questions about any finding
- Adjust recommendations if user provides additional context
- Help narrow down to a final decision

## Key Rules

- **Always launch 3 agents in parallel** — never sequentially
- **One question at a time** in Phase 1 — don't overwhelm
- **2-3 features max** — actionable, not a laundry list
- **Evidence-based** — every recommendation must cite specific findings
- **No implementation** — this skill discovers WHAT to build, not HOW
