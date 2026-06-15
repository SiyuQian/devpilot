---
name: haiku-agent-self-evolution
description: Use when the agent keeps making the same mistake across multiple PRs, when lint/test failures recur on the same rule, when PR review comments reveal a missing guide or unenforced convention, or when you want to proactively evolve the harness. Triggers on "self-evolve", "optimize harness", "update harness docs", "agent keeps making the same mistake", "harness self evolution", "harness 自我进化", "优化 harness", "更新 harness 文档".
---

# Haiku Agent Self-Evolution

Observe agent failures, classify them as Guide Gaps or Sensor Gaps, and propose targeted patches to harness artifacts (CLAUDE.md, skill references, linter config) so the same mistake cannot happen again.

Grounded in Hashimoto's law: *every time the agent makes a mistake, engineer the harness so it cannot make that mistake again.* Every mutation must cite a concrete signal; no speculative changes.

## Reference Index

- `references/signal-types.md` — taxonomy of detectable failure signals with detection methods
- `references/classification-tree.md` — decision algorithm: signal → Guide Gap vs Sensor Gap → target artifact
- `references/mutation-rules.md` — risk matrix: which artifacts are safe to mutate and under what approval gates
- `references/guardrails.md` — safeguards against runaway self-modification

## Workflow

### Phase 1 — Collect Signals

Accept any combination of inputs:

**Option A — User provides raw input:**
- Paste lint output, test failure, or CI log directly
- Provide a PR number (use GitHub MCP to fetch review comments)

**Option B — Agent scans recent history:**
```bash
git log --oneline -20
```
For each commit with a failure marker, read the relevant diff and CI output.

Parse all inputs into a structured signal list:
```
{ type: "lint_failure" | "test_failure" | "review_comment" | "repeated_violation",
  rule: <string>,
  occurrences: <number>,
  source: "ci" | "pr_review" | "git_log",
  evidence: [<exact quote or line reference>] }
```

Discard single-occurrence signals with no prior evidence — file them as "watching" but do not propose mutations yet.

### Phase 2 — Classify Failures

Run each signal through the classification tree in `references/classification-tree.md`.

Output for each signal:
```
Signal: <description>
Evidence: <concrete quote>
Classification: GUIDE GAP | SENSOR GAP | DECOMPOSITION | CONTEXT OVERLOAD | INSUFFICIENT EVIDENCE
Target artifact: CLAUDE.md | skill/<name>/references/<file>.md | .golangci.yml | settings.json | <test file>
Proposed fix type: append rule | add example | add linter rule | add hook | add test
```

### Phase 3 — Draft Patches

For each classified signal, generate a concrete diff. Show every patch before applying anything.

**GUIDE GAP → add to CLAUDE.md or skill reference:**
```diff
## Conventions the agent keeps getting wrong
+ - <new rule derived from the signal>
```

**SENSOR GAP → add linter rule or hook:**
```yaml
# .golangci.yml addition
linters-settings:
  <linter>:
    <rule>: true
```

Each patch must include:
- **Source:** exact lint line / PR comment URL / commit hash
- **Prevents:** what specific error this blocks

### Phase 4 — Gate and Execute

Apply the risk matrix from `references/mutation-rules.md`:

| Risk | Artifacts | Gate |
|------|-----------|------|
| HIGH | CLAUDE.md, settings.json | Show full diff → explicit user "yes" required before any write |
| MEDIUM | skill SKILL.md, .golangci.yml | Show diff → recommend user approves |
| LOW | skill references/*.md | Show diff → can commit if CI passes |

**After user approves each patch:**
1. Apply the edit
2. Run `make lint && make test`
3. If green: commit with message `harness: <what changed> (fixes <signal source>)`
4. If red: revert immediately, report what broke, ask for clarification

**Never batch HIGH and LOW risk patches into one commit.** Each risk tier gets its own commit.

## Guardrails

Check all guardrails in `references/guardrails.md` before applying any patch. Key gates:

- **CLAUDE.md line count:** count lines before proposing addition. If current count ≥ 100, stop: "CLAUDE.md is at N lines. Identify which rule to move to a skill reference first."
- **Evidence required:** if a patch has no concrete signal citation, reject it. State: "No concrete evidence for this change — need at least 2 occurrences before adding a rule."
- **Append only:** never delete or substantially rewrite an existing rule without explicit user instruction.
- **Modification frequency:** if the same file has been modified ≥3 times in the last 24h by this skill, pause and ask: "This file has been modified 3 times today. Continue?"

## Output Format

After completing the analysis, present:

```
## Harness Evolution Report

### Signals Found
- [GUIDE GAP] <rule> — seen N times in <sources>
- [SENSOR GAP] <rule> — rule exists in CLAUDE.md but no mechanical check
- [WATCHING] <rule> — single occurrence, not enough evidence yet

### Proposed Patches

**Patch 1 — HIGH RISK — CLAUDE.md**
Source: PR #42 review comment "..."
Prevents: agent omitting error wrapping at layer boundaries
<diff>

**Patch 2 — LOW RISK — skills/devpilot-google-go-style/references/errors.md**
Source: lint failure in commit abc1234
Prevents: bare error returns without context
<diff>

### Awaiting Your Approval
Patch 1 requires explicit approval before applying.
Patch 2 can be applied automatically once CI passes.
```

Do not apply any patch until the user has seen the full report and approved the relevant risk tiers.
