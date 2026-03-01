# Agent Team Blueprints Skill — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a Claude Code skill that guides users through selecting and configuring Agent Team blueprints, generating project-specific CLAUDE.md sections, hook scripts, and spawn prompts.

**Architecture:** Pure markdown skill with progressive disclosure. SKILL.md (~170 lines) contains dialogue flow and decision tree. Six reference files (~80-120 lines each) provide blueprint-specific templates loaded on demand.

**Tech Stack:** Markdown only. No code dependencies. Follows existing skill conventions from pm and mcp-builder skills.

---

### Task 1: Create skill directory structure

**Files:**
- Create: `skills/agent-teams/SKILL.md` (empty placeholder)
- Create: `skills/agent-teams/references/` (directory)

**Step 1: Create the directory structure**

Run:
```bash
mkdir -p skills/agent-teams/references
touch skills/agent-teams/SKILL.md
```

**Step 2: Verify structure**

Run:
```bash
ls -la skills/agent-teams/ && ls -la skills/agent-teams/references/
```
Expected: SKILL.md exists, references/ directory exists and is empty.

**Step 3: Commit**

```bash
git add skills/agent-teams/
git commit -m "chore: scaffold agent-teams skill directory"
```

---

### Task 2: Write SKILL.md

**Files:**
- Create: `skills/agent-teams/SKILL.md`

**Step 1: Write the complete SKILL.md**

Write the file with this exact content:

```markdown
---
name: devpilot:agent-teams
description: Agent Team blueprint library for Claude Code. Use when the user wants to set up agent teams, coordinate multiple Claude instances, do parallel development, run multi-agent workflows, swarm tasks, or orchestrate teammates. Triggers on "agent team", "multi-agent", "parallel agents", "swarm", "团队协作", "多agent", "并行开发".
---

# Agent Team Blueprints

Configure and deploy Claude Code Agent Teams using pre-built orchestration patterns. This skill guides you through selecting a blueprint, customizing parameters, and generating all configuration files needed to run a team.

## Prerequisites

Agent Teams is an experimental feature. Ensure it is enabled:
```json
// .claude/settings.json
{ "env": { "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1" } }
```

## Process

### Step 1: Understand the Task

Ask the user what they want to accomplish. Listen for signals that map to a blueprint:

| Signal | Blueprint |
|--------|-----------|
| Multi-angle review, audit, analysis of same code | Parallel Specialists |
| Sequential stages with dependencies (design → build → test) | Pipeline |
| Many similar/repetitive tasks (batch migration, bulk tests) | Swarm |
| Complex bug, root cause analysis, multiple theories | Competing Hypotheses |
| Large feature needing architecture design before implementation | Plan-First Parallel |

### Step 2: Recommend Blueprint

Based on the task, recommend 1-2 blueprints. Explain why. If ambiguous, show the comparison from [📋 Overview](./references/overview.md).

After the user selects a blueprint, read the corresponding reference file:
- [Parallel Specialists](./references/parallel-specialists.md)
- [Pipeline](./references/pipeline.md)
- [Swarm](./references/swarm.md)
- [Competing Hypotheses](./references/competing-hypotheses.md)
- [Plan-First Parallel](./references/plan-first-parallel.md)

### Step 3: Collect Parameters

Ask one question at a time. Every blueprint needs:

1. **Team name** — short identifier (e.g., `auth-refactor`)
2. **Verification commands** — test, lint, build commands for this project
3. **Blueprint-specific parameters** — see the reference file for the chosen blueprint

Infer sensible defaults from the project's existing CLAUDE.md, package.json, or Makefile when possible.

### Step 4: Generate Configuration

Using the templates from the reference file, generate 3 artifacts with the user's parameters filled in:

**Artifact 1: CLAUDE.md Team Section**

Append to the project's CLAUDE.md (never overwrite existing content):

```
## Agent Team: {team_name}

### Blueprint: {blueprint_type}
{one-line team goal}

### Module Boundaries
| Module | Owner | Files |
|--------|-------|-------|
| {module} | {teammate_name} | {file_glob} |

### Verification Commands
- Tests: `{test_command}`
- Lint: `{lint_command}`
- Build: `{build_command}`

### Conventions
- Each teammate only modifies files in their assigned module
- Run verification commands before marking any task complete
- {additional blueprint-specific conventions}
```

**Artifact 2: Hook Scripts + settings.json**

Create `.claude/hooks/task-completed.sh`:
```bash
#!/bin/bash
{test_command}
if [ $? -ne 0 ]; then
  echo "Tests failed. Fix before marking complete." >&2
  exit 2
fi
```

Create `.claude/hooks/teammate-idle.sh` (blueprint-specific, see reference file).

Merge into `.claude/settings.json`:
```json
{
  "env": { "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1" },
  "hooks": {
    "TaskCompleted": [{ "hooks": [{ "type": "command", "command": ".claude/hooks/task-completed.sh" }] }],
    "TeammateIdle": [{ "hooks": [{ "type": "command", "command": ".claude/hooks/teammate-idle.sh" }] }]
  }
}
```

**Artifact 3: Spawn Prompt List**

For each teammate, output:

```
### Teammate: {name}
- Agent type: {type}
- Model: {model}
- Run in background: true

Prompt:
  You are {role_description}.
  Your task: {specific_task}
  Files you own: {file_list}
  Files you must NOT modify: {exclusion_list}
  Verification: Run `{verify_command}` before marking any task complete.
  When done, update your task status to completed.
```

### Step 5: Write Files

Show all 3 artifacts to the user. After confirmation:

1. Append CLAUDE.md section (never overwrite)
2. Create `.claude/hooks/` scripts with `chmod +x`
3. Merge settings.json hooks (preserve existing fields)
4. Output spawn prompts as text for the user to execute

Do NOT auto-execute TeamCreate or spawn teammates — the user decides when to start.

## Edge Cases

- **No CLAUDE.md**: Create a new file with just the team section
- **Existing settings.json**: Deep merge hooks, preserve all other config
- **User unsure which blueprint**: Read and present overview.md comparison table
- **Agent Teams not enabled**: Prompt user to add the env var to settings.json
- **User wants customization beyond parameters**: Direct them to read the blueprint's reference file

## Key Rules

- One question at a time during parameter collection
- Always read the reference file for the selected blueprint before generating
- Never overwrite existing CLAUDE.md content — always append
- Never auto-execute team creation — generate config only
- Infer defaults from project context when possible
```

**Step 2: Validate line count**

Run:
```bash
wc -l skills/agent-teams/SKILL.md
```
Expected: ~150-180 lines (under the 500-line skill body guideline).

**Step 3: Validate YAML frontmatter**

Run:
```bash
head -5 skills/agent-teams/SKILL.md
```
Expected: Valid `---` delimited YAML with `name` and `description` fields.

**Step 4: Commit**

```bash
git add skills/agent-teams/SKILL.md
git commit -m "feat: add agent-teams skill with dialogue flow and decision tree"
```

---

### Task 3: Write references/overview.md

**Files:**
- Create: `skills/agent-teams/references/overview.md`

**Step 1: Write the overview reference**

Write the file with this content — a comparison table and selection guide that Claude reads when the user is unsure which blueprint to pick:

```markdown
# Agent Team Blueprints — Overview

## Quick Comparison

| Blueprint | Best For | Team Size | Dependencies | Token Cost |
|-----------|----------|-----------|--------------|------------|
| Parallel Specialists | Multi-angle review/audit | 3-5 | None (all parallel) | Medium |
| Pipeline | Full-stack features with stages | 3-5 | Sequential chain | Medium-High |
| Swarm | Batch/bulk homogeneous tasks | 3-8 | None (all parallel) | Low per worker |
| Competing Hypotheses | Complex debugging, root cause | 2-3 | None (parallel + debate) | Medium |
| Plan-First Parallel | Large features needing design | 1 then 3-5 | Phase 1 → Phase 2 | High |

## Selection Guide

Ask these questions to pick the right blueprint:

1. **Are all tasks independent and similar?**
   - Yes, reviewing same code from different angles → **Parallel Specialists**
   - Yes, many copies of the same task type → **Swarm**

2. **Do tasks depend on each other?**
   - Yes, stage A must finish before stage B starts → **Pipeline**
   - Yes, need design before implementation → **Plan-First Parallel**

3. **Is this an investigation/debugging task?**
   - Yes, multiple theories to test → **Competing Hypotheses**

## Common Parameters (All Blueprints)

Every blueprint requires:
- **Team name**: Short identifier (e.g., `auth-refactor`, `v2-migration`)
- **Verification commands**: `test_command`, `lint_command`, `build_command`
- **File boundaries**: Which directories/files each teammate owns

## Token Cost Guidance

Real-world approximate costs per session:
- Solo session: ~200K tokens
- 3-person team: ~800K tokens
- 5-person team: ~1.3M tokens

Use Haiku model for review-only and simple tasks. Reserve Opus/Sonnet for complex implementation work.

## Best Practices (All Blueprints)

1. **Start small**: 3 teammates before 5. Three focused teammates > five scattered ones.
2. **5-6 tasks per worker**: Optimal productivity density.
3. **Clear file ownership**: Assign non-overlapping directories to avoid merge conflicts.
4. **Rich CLAUDE.md**: Define module boundaries and verification commands. Three teammates reading a clear CLAUDE.md is cheaper than three exploring independently.
5. **Plan first**: Use plan mode to create task breakdowns before spinning up the team.
6. **Delegate mode**: Press `Shift+Tab` to keep the lead as coordinator only.
```

**Step 2: Verify file**

Run:
```bash
wc -l skills/agent-teams/references/overview.md
```
Expected: ~60-70 lines.

**Step 3: Commit**

```bash
git add skills/agent-teams/references/overview.md
git commit -m "feat: add agent-teams overview reference with comparison table"
```

---

### Task 4: Write references/parallel-specialists.md

**Files:**
- Create: `skills/agent-teams/references/parallel-specialists.md`

**Step 1: Write the Parallel Specialists blueprint reference**

Write the file following the design doc's reference file format (8 sections):

```markdown
# Blueprint: Parallel Specialists

Multiple teammates review or analyze the same codebase in parallel, each bringing a specialized perspective. The lead coordinates and synthesizes their findings into a unified report.

---

## When to Use

- Code review from multiple angles (security, performance, maintainability)
- Architecture audit with different lenses
- Pre-merge quality gate with parallel checks
- Competitive analysis of implementation approaches

## When NOT to Use

- Tasks that require sequential dependencies — use Pipeline
- Tasks where teammates need to modify code — this is read-only analysis
- Simple single-perspective reviews — a single subagent suffices

## Team Structure

```
Lead (coordinator — synthesizes findings)
├── Specialist A (e.g., security-reviewer)
├── Specialist B (e.g., performance-reviewer)
└── Specialist C (e.g., test-coverage-reviewer)
```

- All specialists run in parallel with no dependencies
- Each specialist reports findings independently
- Lead waits for all to complete, then produces a combined report

## Parameters to Collect

| Parameter | Question | Default |
|-----------|----------|---------|
| perspectives | What review angles do you need? | security, performance, test-coverage |
| scope | Which files/directories to review? | Entire project |
| output_format | How should findings be reported? | Markdown summary |
| model | Which model for specialists? | haiku |
| teammate_count | How many specialists? | 3 |

## CLAUDE.md Template

```
## Agent Team: {team_name}

### Blueprint: Parallel Specialists
{goal_description}

### Review Scope
{file_glob_or_directory}

### Specialists
| Role | Perspective | Focus |
|------|-------------|-------|
| {specialist_name} | {perspective} | {specific_focus_areas} |

### Verification Commands
- Tests: `{test_command}`
- Lint: `{lint_command}`

### Conventions
- Specialists are read-only — do not modify code
- Each specialist produces a structured findings report
- Lead synthesizes all reports into a final summary
```

## Spawn Prompt Templates

### Lead Prompt
```
You are the team lead coordinating a parallel specialist review.

Team: {team_name}
Goal: {goal_description}
Scope: {scope}

Your role:
1. Create one task per specialist with their specific review focus
2. Spawn all specialists in parallel (run_in_background: true)
3. Wait for all specialists to complete
4. Synthesize their findings into a single report with:
   - Critical issues (must fix)
   - Warnings (should fix)
   - Suggestions (nice to have)
5. Present the report to the user
```

### Specialist Prompt
```
You are a {perspective} specialist reviewing code.

Your focus: {specific_focus_areas}
Scope: {scope}

Instructions:
1. Read all files in scope
2. Analyze from your specialized perspective
3. Document findings as:
   - CRITICAL: {description} at {file}:{line}
   - WARNING: {description} at {file}:{line}
   - SUGGESTION: {description} at {file}:{line}
4. Mark your task as completed when done
```

## Hook Templates

### teammate-idle.sh
```bash
#!/bin/bash
# Parallel Specialists: no extra work after review complete
# Specialists should go idle after finishing their review
exit 0
```

## Best Practices

- **3-5 specialists** is optimal. More than 5 adds cost without proportional value.
- **Use Haiku** for all specialists — review tasks rarely need Opus reasoning.
- **Non-overlapping perspectives** — ensure each specialist has a distinct focus, not just "review everything."
- **Structured output format** — require CRITICAL/WARNING/SUGGESTION categories for easy synthesis.

## Example

**Scenario:** Pre-release security and quality audit of an authentication module.

```
Team name: auth-audit
Goal: Multi-perspective review of src/auth/ before v2.0 release
Specialists:
  - security-reviewer: OWASP top 10, credential handling, injection vectors
  - performance-reviewer: N+1 queries, memory leaks, connection pooling
  - test-coverage-reviewer: Missing edge cases, error paths, integration gaps
Model: haiku
Scope: src/auth/**  tests/auth/**
Verification: npm test
```
```

**Step 2: Verify file**

Run:
```bash
wc -l skills/agent-teams/references/parallel-specialists.md
```
Expected: ~110-130 lines.

**Step 3: Commit**

```bash
git add skills/agent-teams/references/parallel-specialists.md
git commit -m "feat: add parallel-specialists blueprint reference"
```

---

### Task 5: Write references/pipeline.md

**Files:**
- Create: `skills/agent-teams/references/pipeline.md`

**Step 1: Write the Pipeline blueprint reference**

```markdown
# Blueprint: Pipeline

Sequential stages where each teammate's output feeds the next. Tasks use `addBlockedBy` to enforce ordering, with parallel branches where dependencies allow.

---

## When to Use

- Full-stack feature development (design → backend → frontend → tests)
- Refactoring with migration steps (analyze → transform → validate)
- Any workflow where stage N depends on stage N-1's output

## When NOT to Use

- All tasks are independent — use Parallel Specialists or Swarm
- Tasks are all identical — use Swarm
- Need to explore multiple approaches — use Competing Hypotheses

## Team Structure

```
Lead (coordinator)
├── Stage 1: Architect (design API schema)
├── Stage 2: Backend (implement endpoints) ← blocked by Stage 1
├── Stage 3: Frontend (integrate UI)       ← blocked by Stage 2
└── Stage 4: Tests (integration tests)     ← blocked by Stage 2
```

Stages 3 and 4 run in parallel — both depend only on Stage 2.

## Parameters to Collect

| Parameter | Question | Default |
|-----------|----------|---------|
| stages | What stages does your workflow need? | design, backend, frontend, tests |
| stage_dirs | Which directories does each stage own? | (infer from project) |
| plan_approval | Should the lead approve each stage output? | false |
| model | Which model for each stage? | sonnet for architect, haiku for others |

## CLAUDE.md Template

```
## Agent Team: {team_name}

### Blueprint: Pipeline
{goal_description}

### Pipeline Stages
| Stage | Role | Files | Blocked By |
|-------|------|-------|------------|
| 1 | {role_1} | {dir_1} | — |
| 2 | {role_2} | {dir_2} | Stage 1 |
| 3 | {role_3} | {dir_3} | Stage 2 |
| 4 | {role_4} | {dir_4} | Stage 2 |

### Verification Commands
- Tests: `{test_command}`
- Lint: `{lint_command}`
- Build: `{build_command}`

### Conventions
- Each stage only modifies files in its assigned directory
- Downstream stages must not start until their dependencies complete
- Each stage runs verification commands before marking complete
```

## Spawn Prompt Templates

### Lead Prompt
```
You are the team lead coordinating a pipeline workflow.

Team: {team_name}
Goal: {goal_description}

Your role:
1. Create tasks for each pipeline stage with correct dependencies:
   - Task 1: {stage_1_description} (no dependencies)
   - Task 2: {stage_2_description} (addBlockedBy: [Task 1])
   - Task 3: {stage_3_description} (addBlockedBy: [Task 2])
   - Task 4: {stage_4_description} (addBlockedBy: [Task 2])
2. Spawn one teammate per stage (run_in_background: true)
3. Each teammate will auto-start when their dependencies resolve
4. Monitor progress and intervene if a stage fails
5. After all stages complete, verify the integrated result
```

### Stage Teammate Prompt
```
You are the {role} teammate in a pipeline workflow.

Your stage: Stage {N} — {stage_description}
Files you own: {stage_dirs}
Files you must NOT modify: {other_dirs}
Blocked by: {dependency_description}

Instructions:
1. Wait for your task to become unblocked (dependencies will auto-resolve)
2. Claim your task via TaskUpdate
3. {stage_specific_instructions}
4. Run `{verify_command}` to validate your work
5. Mark your task as completed
```

## Hook Templates

### task-completed.sh
```bash
#!/bin/bash
# Pipeline: run tests and build to verify stage output
{test_command} && {build_command}
if [ $? -ne 0 ]; then
  echo "Stage verification failed. Fix before completing." >&2
  exit 2
fi
```

### teammate-idle.sh
```bash
#!/bin/bash
# Pipeline: stages should go idle after completing their single task
exit 0
```

## Best Practices

- **Minimize stage count** — 3-4 stages is ideal. More stages = more handoff overhead.
- **Parallel branches where possible** — if stages 3 and 4 don't depend on each other, let them run in parallel.
- **Clear interface contracts** — document what each stage produces (e.g., "Stage 1 produces API schema at `docs/api-schema.md`").
- **Plan approval for critical stages** — enable for architecture/design stages to catch problems early.

## Example

**Scenario:** Adding a new payment endpoint to a full-stack app.

```
Team name: payment-feature
Goal: Add POST /api/payments endpoint with Stripe integration

Stages:
  Stage 1 (architect): Design API schema and data model → docs/payment-api.md
  Stage 2 (backend): Implement endpoint and Stripe client → src/api/payments/
  Stage 3 (frontend): Payment form UI → src/client/payments/  (blocked by 2)
  Stage 4 (tests): Integration tests → tests/payments/  (blocked by 2)

Model: sonnet for architect, haiku for others
Verification: npm test && npm run build
```
```

**Step 2: Verify file**

Run:
```bash
wc -l skills/agent-teams/references/pipeline.md
```
Expected: ~120-140 lines.

**Step 3: Commit**

```bash
git add skills/agent-teams/references/pipeline.md
git commit -m "feat: add pipeline blueprint reference"
```

---

### Task 6: Write references/swarm.md

**Files:**
- Create: `skills/agent-teams/references/swarm.md`

**Step 1: Write the Swarm blueprint reference**

```markdown
# Blueprint: Swarm

A pool of identical workers that self-organize around a shared task list. The lead creates all tasks upfront, workers claim and complete them independently. Maximum parallelism for homogeneous work.

---

## When to Use

- Batch file migration (e.g., migrate 20 API endpoints to new framework)
- Bulk test writing (add unit tests to 15 modules)
- Mass refactoring (rename pattern across 30 files)
- Parallel data processing tasks

## When NOT to Use

- Tasks have dependencies on each other — use Pipeline
- Tasks need different expertise per task — use Parallel Specialists
- Tasks require design decisions — use Plan-First Parallel

## Team Structure

```
Lead (task creator — creates all tasks upfront)
├── Worker 1 (claims tasks from TaskList)
├── Worker 2 (claims tasks from TaskList)
├── Worker 3 (claims tasks from TaskList)
└── ...Worker N
```

- All workers use the same spawn prompt
- Workers poll TaskList, claim unclaimed tasks, complete them, repeat
- No dependencies between tasks — maximum parallelism

## Parameters to Collect

| Parameter | Question | Default |
|-----------|----------|---------|
| worker_count | How many parallel workers? | 3 |
| task_list | What are the individual tasks? (file glob or manual list) | (user provides) |
| task_template | What should each worker do per task? | (user describes) |
| verify_command | How to verify each task? | project test command |
| model | Which model for workers? | haiku |

## CLAUDE.md Template

```
## Agent Team: {team_name}

### Blueprint: Swarm
{goal_description}

### Task Pool
{N} tasks total, {worker_count} parallel workers

### Task Template
Each worker performs: {task_description_template}

### Verification Commands
- Per task: `{verify_command}`
- Final: `{final_verify_command}`

### Conventions
- Workers claim ONE task at a time from TaskList
- Run verification after each task before marking complete
- Do not modify files outside the current task's scope
```

## Spawn Prompt Templates

### Lead Prompt
```
You are the team lead coordinating a swarm of {worker_count} workers.

Team: {team_name}
Goal: {goal_description}

Your role:
1. Create all tasks upfront. Each task should be:
{task_list_with_descriptions}
2. Spawn {worker_count} workers with identical prompts (run_in_background: true)
3. Workers will self-organize — they claim tasks from TaskList automatically
4. Monitor progress. If a worker is stuck, send a message with guidance.
5. After all tasks complete, run final verification: `{final_verify_command}`
```

### Worker Prompt
```
You are a worker in a swarm team.

Team: {team_name}
Your job: {task_description_template}

Instructions:
1. Check TaskList for unclaimed pending tasks
2. Claim one task via TaskUpdate (set status: in_progress)
3. Complete the task:
   {per_task_instructions}
4. Run `{verify_command}` to validate
5. Mark task as completed
6. Repeat from step 1 until no pending tasks remain
7. When no tasks are left, go idle
```

## Hook Templates

### task-completed.sh
```bash
#!/bin/bash
# Swarm: verify each completed task
{verify_command}
if [ $? -ne 0 ]; then
  echo "Task verification failed." >&2
  exit 2
fi
```

### teammate-idle.sh
```bash
#!/bin/bash
# Swarm: check if there are remaining unclaimed tasks
# If yes, exit 2 to keep the worker active
# Workers will re-check TaskList automatically
exit 0
```

## Best Practices

- **5-6 tasks per worker** for optimal throughput.
- **Haiku model** for all workers — swarm tasks are typically simple and repetitive.
- **Atomic tasks** — each task should be completable independently in one file or small scope.
- **No shared state** — workers must not depend on each other's outputs.
- **Generate task list from file glob** — `find src/ -name "*.ts" | head -20` to create task list automatically.

## Example

**Scenario:** Migrate 15 REST endpoints from Express to Hono.

```
Team name: express-to-hono
Goal: Migrate all Express route handlers to Hono framework

Workers: 4
Tasks (15 total, auto-generated from file glob):
  - Migrate src/routes/users.ts
  - Migrate src/routes/payments.ts
  - Migrate src/routes/products.ts
  - ... (12 more)

Per task: Replace Express Router with Hono app, update middleware, adjust types
Model: haiku
Verify per task: npx tsc --noEmit && npm test -- --testPathPattern={file}
Final verify: npm test && npm run build
```
```

**Step 2: Verify file**

Run:
```bash
wc -l skills/agent-teams/references/swarm.md
```
Expected: ~110-130 lines.

**Step 3: Commit**

```bash
git add skills/agent-teams/references/swarm.md
git commit -m "feat: add swarm blueprint reference"
```

---

### Task 7: Write references/competing-hypotheses.md

**Files:**
- Create: `skills/agent-teams/references/competing-hypotheses.md`

**Step 1: Write the Competing Hypotheses blueprint reference**

```markdown
# Blueprint: Competing Hypotheses

Multiple investigators test different theories about a problem in parallel. Each investigator gathers evidence for their hypothesis and can challenge others' findings. The lead judges which hypothesis best fits the evidence.

---

## When to Use

- Complex bugs with unclear root cause
- Performance issues with multiple potential bottlenecks
- Technical decision-making (choose between 2-3 approaches with evidence)
- Security incident investigation

## When NOT to Use

- Root cause is obvious — just fix it
- Need to implement a solution — use Pipeline or Plan-First Parallel
- Reviewing existing code — use Parallel Specialists

## Team Structure

```
Lead (judge — evaluates evidence, picks winning hypothesis)
├── Investigator A (hypothesis: database deadlock)
├── Investigator B (hypothesis: race condition)
└── Investigator C (hypothesis: memory leak)
```

- All investigators run in parallel
- If debate is enabled, investigators can challenge each other via SendMessage
- Lead collects all evidence reports and determines the strongest hypothesis

## Parameters to Collect

| Parameter | Question | Default |
|-----------|----------|---------|
| hypotheses | What are the possible causes/approaches? | (user provides or lead generates) |
| investigation_scope | Where should investigators look? (logs, code, config) | Entire project |
| enable_debate | Should investigators challenge each other? | true |
| model | Which model for investigators? | sonnet |
| investigator_count | How many investigators? | 2-3 (one per hypothesis) |

## CLAUDE.md Template

```
## Agent Team: {team_name}

### Blueprint: Competing Hypotheses
{problem_description}

### Hypotheses
| # | Hypothesis | Investigator | Scope |
|---|------------|-------------|-------|
| 1 | {hypothesis_1} | investigator-1 | {scope_1} |
| 2 | {hypothesis_2} | investigator-2 | {scope_2} |
| 3 | {hypothesis_3} | investigator-3 | {scope_3} |

### Verification Commands
- Tests: `{test_command}`
- Reproduce bug: `{reproduce_command}`

### Conventions
- Investigators are read-only — do not modify code
- Each investigator must provide concrete evidence (file paths, log entries, reproduction steps)
- Investigators may challenge each other's findings via SendMessage
- Lead makes the final determination based on evidence strength
```

## Spawn Prompt Templates

### Lead Prompt
```
You are the lead judge in a competing hypotheses investigation.

Team: {team_name}
Problem: {problem_description}

Your role:
1. Create one task per hypothesis:
   - Task 1: Investigate "{hypothesis_1}"
   - Task 2: Investigate "{hypothesis_2}"
   - Task 3: Investigate "{hypothesis_3}"
2. Spawn investigators in parallel (run_in_background: true)
3. Wait for all investigations to complete
4. Evaluate the evidence:
   - Which hypothesis has the strongest supporting evidence?
   - Which hypotheses were disproven?
   - Are there any findings that suggest a different root cause?
5. Present your verdict with reasoning to the user
```

### Investigator Prompt
```
You are an investigator testing a specific hypothesis.

Team: {team_name}
Your hypothesis: {hypothesis}
Investigation scope: {scope}

Instructions:
1. Search for evidence that SUPPORTS your hypothesis
2. Search for evidence that DISPROVES your hypothesis (be honest)
3. Document your findings:
   - SUPPORTING: {evidence with file paths and line numbers}
   - CONTRADICTING: {evidence that weakens your hypothesis}
   - CONFIDENCE: HIGH / MEDIUM / LOW with reasoning
4. {if_debate_enabled} If you find evidence that contradicts another investigator's hypothesis, send them a message via SendMessage challenging their findings.
5. Mark your task as completed with your evidence report
```

## Hook Templates

### teammate-idle.sh
```bash
#!/bin/bash
# Competing Hypotheses: investigators should go idle after reporting
exit 0
```

## Best Practices

- **2-3 investigators** — more than 3 rarely adds value and doubles token cost.
- **Use Sonnet** — investigation requires reasoning, not just code reading.
- **Require disconfirming evidence** — investigators must look for evidence AGAINST their hypothesis, not just for it.
- **Time-box investigations** — set a task scope limit to prevent investigators from going too deep.
- **Enable debate for complex problems** — peer challenges catch blind spots.

## Example

**Scenario:** API response times degraded 5x after a recent deploy.

```
Team name: api-perf-investigation
Problem: GET /api/users response time went from 50ms to 250ms after deploy #847

Hypotheses:
  1. Database query regression (investigator-1): Check query plans, N+1 queries
     Scope: src/db/, src/models/, database logs
  2. Middleware overhead (investigator-2): Check new auth middleware timing
     Scope: src/middleware/, src/api/routes.ts
  3. External API latency (investigator-3): Check third-party service call times
     Scope: src/services/external/, network logs

Model: sonnet
Debate: enabled
Reproduce: curl -w "%{time_total}" http://localhost:3000/api/users
```
```

**Step 2: Verify file**

Run:
```bash
wc -l skills/agent-teams/references/competing-hypotheses.md
```
Expected: ~120-140 lines.

**Step 3: Commit**

```bash
git add skills/agent-teams/references/competing-hypotheses.md
git commit -m "feat: add competing-hypotheses blueprint reference"
```

---

### Task 8: Write references/plan-first-parallel.md

**Files:**
- Create: `skills/agent-teams/references/plan-first-parallel.md`

**Step 1: Write the Plan-First Parallel blueprint reference**

```markdown
# Blueprint: Plan-First Parallel

Two-phase workflow: first an architect designs the approach and gets lead approval, then a team of implementers executes the approved plan in parallel. Best for large features that need design before coding.

---

## When to Use

- Large features spanning 5+ files
- Cross-cutting changes (new auth system, database migration)
- Greenfield modules that need architecture decisions
- Any work where "build it wrong" costs more than "design it first"

## When NOT to Use

- Task is well-defined and doesn't need design — use Pipeline or Swarm
- Task is investigation/analysis — use Competing Hypotheses or Parallel Specialists
- Small changes (< 3 files) — a single agent suffices

## Team Structure

### Phase 1: Design
```
Lead (coordinator)
└── Architect (plan mode, read-only)
    → Produces design document
    → Sends plan_approval_request to Lead
    → Lead approves or requests revision
```

### Phase 2: Implementation (after approval)
```
Lead (coordinator)
├── Implementer A (module 1)
├── Implementer B (module 2)
└── Implementer C (module 3)
```

## Parameters to Collect

| Parameter | Question | Default |
|-----------|----------|---------|
| architect_constraints | Any tech stack or architecture constraints? | (infer from project) |
| implementer_count | How many parallel implementers? | 3 |
| impl_plan_approval | Should implementers also need plan approval? | false |
| model_architect | Which model for architect? | opus or sonnet |
| model_implementers | Which model for implementers? | sonnet |

## CLAUDE.md Template

```
## Agent Team: {team_name}

### Blueprint: Plan-First Parallel
{goal_description}

### Phase 1: Architecture
Architect designs: {what_to_design}
Constraints: {constraints}
Output: Design document at {design_doc_path}

### Phase 2: Implementation
| Module | Implementer | Files | Blocked By |
|--------|-------------|-------|------------|
| {module_1} | impl-1 | {dir_1} | Architecture approval |
| {module_2} | impl-2 | {dir_2} | Architecture approval |
| {module_3} | impl-3 | {dir_3} | Architecture approval |

### Verification Commands
- Tests: `{test_command}`
- Lint: `{lint_command}`
- Build: `{build_command}`

### Conventions
- Architect must get plan approval before Phase 2 begins
- Implementers only modify files in their assigned module
- Implementers follow the approved design document
- Each implementer runs verification before marking complete
```

## Spawn Prompt Templates

### Lead Prompt
```
You are the team lead coordinating a plan-first parallel workflow.

Team: {team_name}
Goal: {goal_description}

Phase 1 — Architecture:
1. Create a task: "Design {feature} architecture"
2. Spawn architect teammate (agent type: Plan, model: {model_architect})
3. Architect will work in plan mode and send a plan_approval_request
4. Review the design. Approve if sound, or reject with feedback.
5. Architect may revise and resubmit.

Phase 2 — Implementation (after approval):
1. Based on the approved design, create implementation tasks:
   - Task: Implement {module_1} (addBlockedBy: [design task])
   - Task: Implement {module_2} (addBlockedBy: [design task])
   - Task: Implement {module_3} (addBlockedBy: [design task])
2. Spawn {implementer_count} implementers (run_in_background: true)
3. Monitor progress, verify the integrated result after all complete
4. Run final verification: `{test_command} && {build_command}`
```

### Architect Prompt
```
You are the architect designing {feature_description}.

Team: {team_name}
Constraints: {constraints}

Instructions:
1. Read the existing codebase to understand current architecture
2. Design the solution considering:
   - Module boundaries (what can be built in parallel)
   - Interface contracts between modules
   - Data flow and dependencies
   - Error handling strategy
3. Write your design document covering:
   - Overview and approach
   - Module breakdown with file ownership
   - Interface contracts (function signatures, API schemas)
   - Implementation order and dependencies
4. Submit your design via plan_approval_request
5. If rejected, revise based on feedback and resubmit
```

### Implementer Prompt
```
You are an implementer building {module_name}.

Team: {team_name}
Design document: {design_doc_path}
Files you own: {file_list}
Files you must NOT modify: {exclusion_list}

Instructions:
1. Read the approved design document
2. Implement {module_name} following the design exactly
3. Follow interface contracts — do not change function signatures or API schemas
4. Write tests for your module
5. Run `{verify_command}` to validate
6. Mark your task as completed
```

## Hook Templates

### task-completed.sh
```bash
#!/bin/bash
# Plan-First Parallel: verify implementation against design
{test_command} && {build_command}
if [ $? -ne 0 ]; then
  echo "Implementation verification failed." >&2
  exit 2
fi
```

### teammate-idle.sh
```bash
#!/bin/bash
# Plan-First: teammates complete one task then go idle
exit 0
```

## Best Practices

- **Opus or Sonnet for architect** — design quality is critical, don't skimp on reasoning.
- **Sonnet for implementers** — they need to write correct code, not just read.
- **Clear interface contracts** — the architect must define exact function signatures and data types at module boundaries.
- **3 implementers** is the sweet spot. More than 5 creates coordination overhead.
- **Design doc on disk** — architect should write to a file (e.g., `docs/design-{feature}.md`) so implementers can reference it.
- **Reject bad designs early** — it's cheaper to have the architect iterate than to have 3 implementers build on a flawed foundation.

## Example

**Scenario:** Adding a real-time notification system to an existing app.

```
Team name: notifications
Goal: Add WebSocket-based notification system with persistence

Phase 1 — Architect:
  Constraints: Use existing PostgreSQL, no new infrastructure, TypeScript
  Output: docs/design-notifications.md
  Model: sonnet

Phase 2 — Implementers (3):
  Module 1 (impl-data): Data layer — src/models/notification.ts, src/db/migrations/
  Module 2 (impl-api): WebSocket server — src/ws/, src/api/notifications/
  Module 3 (impl-ui): Frontend components — src/client/notifications/
  Model: sonnet

Verification: npm test && npm run build && npm run typecheck
```
```

**Step 2: Verify file**

Run:
```bash
wc -l skills/agent-teams/references/plan-first-parallel.md
```
Expected: ~140-160 lines.

**Step 3: Commit**

```bash
git add skills/agent-teams/references/plan-first-parallel.md
git commit -m "feat: add plan-first-parallel blueprint reference"
```

---

### Task 9: Validate complete skill structure

**Files:**
- Validate: `skills/agent-teams/SKILL.md`
- Validate: `skills/agent-teams/references/*.md`

**Step 1: Validate directory structure matches design**

Run:
```bash
find skills/agent-teams -type f | sort
```
Expected:
```
skills/agent-teams/SKILL.md
skills/agent-teams/references/competing-hypotheses.md
skills/agent-teams/references/overview.md
skills/agent-teams/references/parallel-specialists.md
skills/agent-teams/references/pipeline.md
skills/agent-teams/references/plan-first-parallel.md
skills/agent-teams/references/swarm.md
```

**Step 2: Validate SKILL.md frontmatter parses correctly**

Run:
```bash
head -4 skills/agent-teams/SKILL.md
```
Expected: Valid YAML frontmatter with `name: devpilot:agent-teams` and description.

**Step 3: Check total line counts are within bounds**

Run:
```bash
echo "=== SKILL.md ===" && wc -l skills/agent-teams/SKILL.md && echo "=== References ===" && wc -l skills/agent-teams/references/*.md
```
Expected: SKILL.md ~150-180 lines, each reference ~60-160 lines.

**Step 4: Verify all blueprint references are linked in SKILL.md**

Run:
```bash
grep -c "references/" skills/agent-teams/SKILL.md
```
Expected: 6 (overview + 5 blueprints).

**Step 5: Run quick_validate.py if available**

Run:
```bash
python3 .claude/skills/skill-creator/scripts/quick_validate.py skills/agent-teams
```
Expected: Validation passes.

**Step 6: Final commit**

```bash
git add -A skills/agent-teams/
git commit -m "feat: complete agent-teams blueprint skill with 5 blueprints"
```
