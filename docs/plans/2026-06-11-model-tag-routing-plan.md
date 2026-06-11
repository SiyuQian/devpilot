# Model-Tag Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scan-filed issues carry a `model:haiku|sonnet|opus` label estimating fix complexity; `devpilot-resolve-issues` backfills the label when missing and passes its value as the Agent tool's `model` param when dispatching implementer subagents.

**Architecture:** Pure skill-content change — markdown prompt files, one Python validator, one evals JSON. The label value is the literal Agent `model` param (no mapping layer). Scanners assign the tier per-finding (they just read the code); resolve-issues backfills at triage and escalates one tier on BLOCKED. Reviewers/triage/final-verify keep the session model.

**Tech Stack:** Markdown skill files, Python 3 (`scripts/check-findings.py`), JSON evals.

**Spec:** `docs/plans/2026-06-11-model-tag-routing-design.md`

---

## Canonical rubric (referenced by several tasks; copy text exactly)

```markdown
- `model:haiku` — mechanical, single-file, low-judgment change: doc drift, typo, adding a nil check, comment fix.
- `model:sonnet` — default tier: a normal code fix plus tests, single concern.
- `model:opus` — multi-file change, or a fix requiring careful reasoning about concurrency, security, or architecture.

Judge the **cost of the fix, not the severity of the problem** — a critical security hole can be a one-line `model:haiku` fix. When unsure, pick the higher tier: wasted tokens are cheaper than a re-dispatch. There is no `model:fable`; `model:opus` is the top of the taxonomy, and beyond it is human escalation.
```

---

### Task 0: Branch

**Files:** none

- [ ] **Step 1: Create the working branch** (repo rule: never work on `main`)

```bash
cd /Users/sqian/Works/github.com/IDEXX/devpilot
git checkout -b feat/model-tag-routing
```

Expected: `Switched to a new branch 'feat/model-tag-routing'`

---

### Task 1: `model:*` group + rubric in scanning-repos labels.md

**Files:**
- Modify: `skills/devpilot-scanning-repos/references/labels.md`

- [ ] **Step 1: Add the label-create block.** After the confidence block (lines 58–60, ending `gh label create confidence:100 …`) and before the closing ` ``` ` of the bash block, insert:

```bash

# --- model (one per issue; implementer routing consumed by devpilot-resolve-issues) ---
gh label create model:haiku    --color D4C5F9 --description "Fix is mechanical: single-file, low-judgment (doc drift, typo, nil check)"
gh label create model:sonnet   --color 8A63D2 --description "Fix is a normal code change plus tests, single concern"
gh label create model:opus     --color 3C1E70 --description "Fix spans files or needs careful concurrency/security/architecture reasoning"
```

- [ ] **Step 2: Add the rubric section.** Immediately after the bash block's closing fence and the `Note: no \`|| true\` guards…` paragraph, insert a new section:

```markdown
## Model tier rubric (`model:*`)

The `model:*` label estimates **how capable a model the fix needs** — it is the routing signal `devpilot-resolve-issues` passes verbatim as the Agent tool's `model` param when dispatching implementer subagents. Scanners assign it per finding (they just read the code and are best placed to judge); `devpilot-resolve-issues` backfills it at triage for issues that lack it.

- `model:haiku` — mechanical, single-file, low-judgment change: doc drift, typo, adding a nil check, comment fix.
- `model:sonnet` — default tier: a normal code fix plus tests, single concern.
- `model:opus` — multi-file change, or a fix requiring careful reasoning about concurrency, security, or architecture.

Judge the **cost of the fix, not the severity of the problem** — a critical security hole can be a one-line `model:haiku` fix. When unsure, pick the higher tier: wasted tokens are cheaper than a re-dispatch. There is no `model:fable`; `model:opus` is the top of the taxonomy, and beyond it is human escalation.
```

- [ ] **Step 3: Extend the suitable-match table.** In the `## Suitable-match guidance` table, add a row after the `confidence:75 / confidence:100` row:

```markdown
| `model:haiku` / `model:sonnet` / `model:opus` | (almost never present) | anything priority- or size-shaped (`size/S`, `effort:low`) — those measure scope, not required model capability |
```

Also update the rule-of-thumb sentence from:

> **subcategory labels (`sec:*`, `edge:*`, `cov:*`) and `confidence:*` rarely have a suitable equivalent — almost always create.**

to:

> **subcategory labels (`sec:*`, `edge:*`, `cov:*`), `confidence:*`, and `model:*` rarely have a suitable equivalent — almost always create.**

- [ ] **Step 4: Update the per-issue label contract** (currently "exactly **five** labels", items 1–5). Change to:

```markdown
Every filed issue MUST have exactly **six** labels (using whichever name the step-2 mapping resolved to — canonical or reused):

1. One category — canonical: `scan:security` | `scan:edge-case` | `scan:coverage` | `scan:doc-drift`
2. One subcategory matching the category — canonical: `sec:*` | `edge:*` | `cov:*` | `doc:*`
3. One severity — canonical: `severity:high` | `severity:medium` | `severity:low`
4. One confidence — canonical: `confidence:75` | `confidence:100`
5. One area — canonical: `area:{top-level-dir}`
6. One model tier — canonical: `model:haiku` | `model:sonnet` | `model:opus` (from the finding's `model` field; see the rubric above)
```

- [ ] **Step 5: Verify**

```bash
grep -n "model:" skills/devpilot-scanning-repos/references/labels.md | head
grep -c "exactly \*\*five\*\*" skills/devpilot-scanning-repos/references/labels.md
```

Expected: model rows present; second command prints `0`.

- [ ] **Step 6: Commit**

```bash
git add skills/devpilot-scanning-repos/references/labels.md
git commit -m "feat(devpilot-scanning-repos): add model:* label group and fix-cost rubric"
```

---

### Task 2: Finding schema + filing contract in scanning-repos SKILL.md

**Files:**
- Modify: `skills/devpilot-scanning-repos/SKILL.md` (step 7 ≈ line 85, Finding format ≈ lines 90–106, acceptance criteria ≈ line 160)

- [ ] **Step 1: Step 7 label list — five → six.** Replace the opening of step 7:

Old:
> Labels: always exactly five — apply the labels from the step-2 mapping table for `scan:<category>`, the matching subcategory, `severity:<level>`, `confidence:<score>`, plus an `area:<top-level-dir>` resolved lazily here.

New:
> Labels: always exactly six — apply the labels from the step-2 mapping table for `scan:<category>`, the matching subcategory, `severity:<level>`, `confidence:<score>`, `model:<tier>` (taken from the finding's `model` field), plus an `area:<top-level-dir>` resolved lazily here.

(The rest of step 7 — the area-label lazy resolution — is unchanged.)

- [ ] **Step 2: Add `model` to the Finding format JSON.** In the `## Finding format` code block, after the `"severity"` line, insert:

```json
  "model": "haiku | sonnet | opus",
```

- [ ] **Step 3: Document the field.** After the paragraph that begins "`subcategory` must match `category`…", insert:

```markdown
`model` is the implementer-routing tier consumed by `devpilot-resolve-issues`: the scanner's estimate of how capable a model the **fix** needs (haiku = mechanical single-file change, sonnet = default code-fix-plus-tests, opus = multi-file or subtle concurrency/security/architecture reasoning). Judge fix cost, not severity; when unsure pick the higher tier. Full rubric in `references/labels.md`.
```

- [ ] **Step 4: Acceptance criterion 2 — five → six.** Replace:

Old:
> 2. Every filed issue has exactly five labels: `scan:<category>`, a matching `sec:*`|`edge:*`|`cov:*`|`doc:*` subcategory, `severity:<level>`, `confidence:<score>` (75 or 100), and an auto-derived `area:<top-level-dir>` (for doc-drift findings, derived from the doc file's path — e.g. a finding in `README.md` → `area:root`, in `docs/cli-reference.md` → `area:docs`).

New:
> 2. Every filed issue has exactly six labels: `scan:<category>`, a matching `sec:*`|`edge:*`|`cov:*`|`doc:*` subcategory, `severity:<level>`, `confidence:<score>` (75 or 100), `model:<tier>` (haiku, sonnet, or opus, from the finding's `model` field), and an auto-derived `area:<top-level-dir>` (for doc-drift findings, derived from the doc file's path — e.g. a finding in `README.md` → `area:root`, in `docs/cli-reference.md` → `area:docs`).

- [ ] **Step 5: Verify**

```bash
grep -n "exactly five\|exactly six\|\"model\"" skills/devpilot-scanning-repos/SKILL.md
```

Expected: zero "exactly five" hits; "exactly six" at step 7 and criterion 2; `"model"` in the Finding format block.

- [ ] **Step 6: Commit**

```bash
git add skills/devpilot-scanning-repos/SKILL.md
git commit -m "feat(devpilot-scanning-repos): add model field to finding schema and six-label contract"
```

---

### Task 3: `model` field in all four scanner agent prompts

**Files:**
- Modify: `skills/devpilot-scanning-repos/agents/security-scanner.md` (Output format ≈ lines 59–77)
- Modify: `skills/devpilot-scanning-repos/agents/edge-case-hunter.md` (Output format ≈ lines 73–91)
- Modify: `skills/devpilot-scanning-repos/agents/coverage-auditor.md` (Output format ≈ lines 51–69)
- Modify: `skills/devpilot-scanning-repos/agents/doc-consistency-auditor.md` (Output format ≈ lines 68–108)

Each agent prompt is dispatched standalone, so each carries its own copy of the rubric.

- [ ] **Step 1: security-scanner.md.** In the Output format JSON example, after `"severity": "high",` insert `"model": "sonnet",`. (Single call-site shell-injection fix with a test — normal code fix.)

- [ ] **Step 2: edge-case-hunter.md.** After `"severity": "medium",` insert `"model": "sonnet",`. (The suggested fix touches constructor or signature — not mechanical.)

- [ ] **Step 3: coverage-auditor.md.** After `"severity": "high",` insert `"model": "sonnet",`. (Writing a four-case middleware test file.)

- [ ] **Step 4: doc-consistency-auditor.md.** The example array has three findings; insert after each finding's `"severity"` line:
  - `doc:command-mismatch` (README `make release`): `"model": "haiku",` — single-line doc fix.
  - `doc:stale-claim` (positional → named queries, 17 call sites): `"model": "opus",` — multi-file migration.
  - `doc:cross-doc-conflict` (lint command): `"model": "haiku",` — single-line doc fix.

- [ ] **Step 5: Add the rubric to each of the four files.** In each agent file, append a new section immediately after the Output format section (before `## Calibration` where present; doc-consistency-auditor: before its `## Calibration`):

```markdown
## Model tier (`model` field — mandatory)

Every finding MUST set `model` to the tier of implementer model its **fix** needs. This routes the eventual fix subagent in `devpilot-resolve-issues`; it is passed verbatim as the Agent tool's `model` param.

- `haiku` — mechanical, single-file, low-judgment change: doc drift, typo, adding a nil check, comment fix.
- `sonnet` — default tier: a normal code fix plus tests, single concern.
- `opus` — multi-file change, or a fix requiring careful reasoning about concurrency, security, or architecture.

Judge the **cost of the fix, not the severity of the problem** — a critical security hole can be a one-line `haiku` fix. When unsure, pick the higher tier.
```

- [ ] **Step 6: Verify**

```bash
grep -c '"model"' skills/devpilot-scanning-repos/agents/*.md
grep -c '## Model tier' skills/devpilot-scanning-repos/agents/*.md
```

Expected: security-scanner 1, edge-case-hunter 1, coverage-auditor 1, doc-consistency-auditor 3 for the first grep; every file 1 for the second.

- [ ] **Step 7: Commit**

```bash
git add skills/devpilot-scanning-repos/agents/
git commit -m "feat(devpilot-scanning-repos): scanners assign model tier per finding"
```

---

### Task 4: issue-template.md label contract

**Files:**
- Modify: `skills/devpilot-scanning-repos/references/issue-template.md`

- [ ] **Step 1: Update the `gh` invocation comment and `--label` line.**

Old (line 8):
```bash
# Resolve the five labels via the step-2 mapping table (canonical → label_to_apply).
```
New:
```bash
# Resolve the six labels via the step-2 mapping table (canonical → label_to_apply).
```

Old (line 18):
```bash
  --label "<category_label>,<subcategory_label>,<severity_label>,<confidence_label>,$area_label" \
```
New:
```bash
  --label "<category_label>,<subcategory_label>,<severity_label>,<confidence_label>,<model_label>,$area_label" \
```

- [ ] **Step 2: Add a bullet** to the list under the invocation (after the `<score>` bullet):

```markdown
- `<model_label>` resolves the finding's `model` field (`haiku` | `sonnet` | `opus`) through the step-2 mapping table — canonical form `model:<tier>`. This is the implementer-routing signal for `devpilot-resolve-issues`.
```

- [ ] **Step 3: Update the Rules bullet.**

Old:
> **Labels are mandatory** — always exactly five, applied via the step-2 mapping table: a category, a matching subcategory, a severity, a confidence, and an auto-derived area. Canonical names are `scan:<category>`, `sec:*` | `edge:*` | `cov:*`, `severity:<level>`, `confidence:<score>`, `area:<top-level-dir>` — but the resolved label may be a suitable existing repo label instead. See `references/labels.md`.

New:
> **Labels are mandatory** — always exactly six, applied via the step-2 mapping table: a category, a matching subcategory, a severity, a confidence, a model tier, and an auto-derived area. Canonical names are `scan:<category>`, `sec:*` | `edge:*` | `cov:*`, `severity:<level>`, `confidence:<score>`, `model:<tier>`, `area:<top-level-dir>` — but the resolved label may be a suitable existing repo label instead. See `references/labels.md`.

- [ ] **Step 4: Verify**

```bash
grep -n "five\|model" skills/devpilot-scanning-repos/references/issue-template.md
```

Expected: no "five" remaining; `<model_label>` in the invocation and bullets.

- [ ] **Step 5: Commit**

```bash
git add skills/devpilot-scanning-repos/references/issue-template.md
git commit -m "feat(devpilot-scanning-repos): six-label issue contract incl. model tier"
```

---

### Task 5: check-findings.py validates `model` (test-first)

**Files:**
- Modify: `skills/devpilot-scanning-repos/scripts/check-findings.py` (REQUIRED_FIELDS lines 24–34, validation in `check()` ≈ line 86)

- [ ] **Step 1: Write the failing check.** Run the validator on a finding **missing** `model` and one with an **invalid** `model` — both currently pass (exit 0), which is the bug:

```bash
cat > /tmp/model-field-test.json <<'EOF'
[
  {"category": "coverage", "subcategory": "cov:error-paths", "title": "Happy path only in internal/auth/token_test.go", "severity": "medium", "file": "internal/auth/token.go", "line_range": "L10-L20", "evidence": "  10  if err != nil { return nil }", "why_it_matters": "Error branch untested.", "suggested_fix": "Add error-path case."},
  {"category": "coverage", "subcategory": "cov:error-paths", "title": "Happy path only in internal/auth/jwt_test.go", "severity": "medium", "file": "internal/auth/jwt.go", "line_range": "L10-L20", "evidence": "  10  if err != nil { return nil }", "why_it_matters": "Error branch untested.", "suggested_fix": "Add error-path case.", "model": "gpt5"}
]
EOF
python3 skills/devpilot-scanning-repos/scripts/check-findings.py /tmp/model-field-test.json; echo "exit=$?"
```

Expected (current behavior): `exit=0` — confirms the validator does not yet enforce the field.

- [ ] **Step 2: Implement.** In `check-findings.py`:

Add `"model",` to `REQUIRED_FIELDS` after `"severity",`:

```python
REQUIRED_FIELDS = (
    "category",
    "subcategory",
    "title",
    "severity",
    "model",
    "file",
    "line_range",
    "evidence",
    "why_it_matters",
    "suggested_fix",
)
```

Add next to `VALID_SEVERITIES`:

```python
VALID_MODELS = {"haiku", "sonnet", "opus"}
```

In `check()`, after the severity validation block:

```python
    model = finding.get("model")
    if model is not None and model not in VALID_MODELS:
        errs.append(f"[{idx}] model='{model}' not in {sorted(VALID_MODELS)}")
```

- [ ] **Step 3: Re-run the failing check — now rejects both:**

```bash
python3 skills/devpilot-scanning-repos/scripts/check-findings.py /tmp/model-field-test.json; echo "exit=$?"
```

Expected: non-zero exit; errors `[0] missing required field 'model'` and `[1] model='gpt5' not in ['haiku', 'opus', 'sonnet']`.

- [ ] **Step 4: Positive case passes.** Add `"model": "sonnet"` to finding 0 and fix finding 1 to `"model": "haiku"` in `/tmp/model-field-test.json`, re-run:

```bash
python3 skills/devpilot-scanning-repos/scripts/check-findings.py /tmp/model-field-test.json; echo "exit=$?"
```

Expected: `exit=0`.

- [ ] **Step 5: Commit**

```bash
git add skills/devpilot-scanning-repos/scripts/check-findings.py
git commit -m "feat(devpilot-scanning-repos): validate required model field in findings"
```

---

### Task 6: evals.json assertions

**Files:**
- Modify: `skills/devpilot-scanning-repos/evals/evals.json` (eval id 0 assertions `labels_provisioned_first` and `three_labels_per_issue`; eval id 1 assertion `three_labels_per_issue`)

- [ ] **Step 1: Eval 0 — `labels_provisioned_first`.** Append the model group to the description:

Old: `"… (3 categories + 18 subcategories + 3 severities + 2 confidences) before any gh issue create call; area:* labels are created lazily at filing time"`
New: `"… (3 categories + 18 subcategories + 3 severities + 2 confidences + 3 model tiers) before any gh issue create call; area:* labels are created lazily at filing time"`

- [ ] **Step 2: Eval 0 — rename and update `three_labels_per_issue`:**

```json
{"name": "six_labels_per_issue", "description": "Every filed issue has exactly six labels: scan:<category>, one matching sec:*|edge:*|cov:* subcategory, severity:<level>, confidence:<score>, model:<tier>, and one auto-derived area:<top-level-dir>"}
```

- [ ] **Step 3: Eval 1 — rename and update `three_labels_per_issue`:**

```json
{"name": "six_labels_per_issue", "description": "Every filed issue still has exactly six labels per the new contract"}
```

- [ ] **Step 4: Verify JSON parses and no stale names remain**

```bash
python3 -m json.tool skills/devpilot-scanning-repos/evals/evals.json > /dev/null && echo OK
grep -c "three_labels_per_issue\|exactly five" skills/devpilot-scanning-repos/evals/evals.json
```

Expected: `OK`, then `0`.

- [ ] **Step 5: Commit**

```bash
git add skills/devpilot-scanning-repos/evals/evals.json
git commit -m "test(devpilot-scanning-repos): six-label assertions incl. model tier"
```

---

### Task 7: resolve-issues — backfill step + model-aware dispatch + deterministic BLOCKED escalation

**Files:**
- Modify: `skills/devpilot-resolve-issues/SKILL.md` (flowchart ≈ line 80, after step 4 ≈ line 186, step 6b ≈ lines 223–229, cross-references ≈ line 396)

- [ ] **Step 1: Flowchart.** In the dot graph, route the REAL edge through a new node.

Old:
```dot
  "Verdict" -> "Create worktree + branch (cd into it)" [label="REAL"];
```
New:
```dot
  "Verdict" -> "Ensure model:* label" [label="REAL"];
  "Ensure model:* label" -> "Create worktree + branch (cd into it)";
```
Also add the node declaration next to the other `[shape=box]` declarations:
```dot
  "Ensure model:* label" [shape=box];
```

- [ ] **Step 2: New step 4.5** — insert between step 4 ("Render a verdict") and step 5 ("Create the per-issue worktree"):

```markdown
### 4.5 Ensure the `model:*` routing label (REAL verdicts only)

Every REAL issue carries exactly one `model:*` label before any implementer is dispatched. It names the Agent-tool `model` param for this issue's implementer subagents — the value is passed verbatim (`model:sonnet` → `model: "sonnet"`). Issues filed by `devpilot-scanning-repos` arrive pre-tagged; human-filed and legacy issues may not.

Check the labels already fetched in step 1:

- **Exactly one `model:*` label** — use it as-is. A pre-existing tag (scanner- or human-applied) is the supported manual override; do not second-guess it.
- **No `model:*` label** — judge the tier yourself from the step-3 investigation, then apply it:
  - `model:haiku` — mechanical, single-file, low-judgment change: doc drift, typo, adding a nil check, comment fix.
  - `model:sonnet` — default tier: a normal code fix plus tests, single concern.
  - `model:opus` — multi-file change, or a fix requiring careful reasoning about concurrency, security, or architecture.

  Judge the **cost of the fix, not the severity of the problem**; when unsure, pick the higher tier. (Keep in sync with `devpilot-scanning-repos/references/labels.md` → "Model tier rubric".)

  ```bash
  # If the repo lacks the label, create it first (same pattern as need:human):
  gh label create "model:<tier>" --color "8A63D2" --description "Implementer-model routing for devpilot-resolve-issues"
  gh issue edit <num> --add-label "model:<tier>"
  ```
- **Multiple `model:*` labels** — keep the highest tier (opus > sonnet > haiku), remove the rest:

  ```bash
  gh issue edit <num> --remove-label "model:<lower-tier>"
  ```
```

- [ ] **Step 3: Step 6b item 2 — pass the model.** Replace:

Old:
> 2. **Dispatch the implementer subagent.** Use the per-task spec from `references/subagent-spec.md`, filled in with the Evidence block, files-to-read, and acceptance criteria scoped to *this task only*. One dispatch per task. Never run implementers in parallel on the same branch.

New:
> 2. **Dispatch the implementer subagent.** Use the per-task spec from `references/subagent-spec.md`, filled in with the Evidence block, files-to-read, and acceptance criteria scoped to *this task only*. Set the Agent tool's `model` param to the issue's `model:*` tier from step 4.5 (`model:sonnet` → `model: "sonnet"`). The model tier applies to **implementers only** — per-task reviewers (6c), triage, and the step-7 final verify inherit the session model; cheap implementer + strong reviewer is the intended pairing. One dispatch per task. Never run implementers in parallel on the same branch.

- [ ] **Step 4: Step 6b BLOCKED bullet — deterministic escalation.** Replace:

Old:
> - **BLOCKED** — read the explanation. Adjust the spec, dispatch a more capable model, or escalate `NEEDS-HUMAN`. Never re-dispatch the same model with the same spec on a BLOCKED return.

New:
> - **BLOCKED** — read the explanation. If the spec was wrong, fix it and re-dispatch at the same tier. Otherwise escalate exactly one model tier (haiku→sonnet, sonnet→opus) and re-dispatch, updating the issue's `model:*` label to match (`gh issue edit <num> --add-label "model:<new>" --remove-label "model:<old>"`); a BLOCKED return at opus escalates the issue `NEEDS-HUMAN`. Never re-dispatch the same model with the same spec.

- [ ] **Step 5: Cross-references.** In the "shaped by / see also" list at the bottom (≈ line 396), add:

```markdown
- `model:*` routing labels and the fix-cost rubric → `devpilot-scanning-repos/references/labels.md`.
```

- [ ] **Step 6: Verify**

```bash
grep -n "model:" skills/devpilot-resolve-issues/SKILL.md | head -20
grep -c "more capable model" skills/devpilot-resolve-issues/SKILL.md
```

Expected: step 4.5, 6b, flowchart hits; second command prints `0`.

- [ ] **Step 7: Commit**

```bash
git add skills/devpilot-resolve-issues/SKILL.md
git commit -m "feat(devpilot-resolve-issues): model:* label backfill and model-aware implementer dispatch"
```

---

### Task 8: subagent-spec.md BLOCKED re-dispatch policy

**Files:**
- Modify: `skills/devpilot-resolve-issues/references/subagent-spec.md` (re-dispatch policy, line 120)

- [ ] **Step 1: Replace the BLOCKED bullet.**

Old:
> - **BLOCKED:** read the explanation, decide between adjusting the spec, dispatching a more capable model, or escalating `NEEDS-HUMAN`. Never re-dispatch the same model with the same spec on a BLOCKED return.

New:
> - **BLOCKED:** read the explanation. Spec wrong → fix it, re-dispatch at the same tier. Otherwise escalate exactly one model tier (haiku→sonnet, sonnet→opus), update the issue's `model:*` label to match, and re-dispatch; BLOCKED at opus → escalate `NEEDS-HUMAN` (see SKILL.md step 6b). Never re-dispatch the same model with the same spec on a BLOCKED return.

- [ ] **Step 2: Document the dispatch param.** In "Rules for the main agent filling this template", after the first bullet ("Dispatch from inside the issue's worktree…"), add:

```markdown
- **Set the Agent tool's `model` param from the issue's `model:*` label** (step 4.5 of `SKILL.md`): `model:haiku` → `"haiku"`, `model:sonnet` → `"sonnet"`, `model:opus` → `"opus"`. Implementers only — reviewers and the final verify inherit the session model.
```

- [ ] **Step 3: Verify**

```bash
grep -c "more capable model" skills/devpilot-resolve-issues/references/subagent-spec.md
grep -n "model:\*" skills/devpilot-resolve-issues/references/subagent-spec.md
```

Expected: `0`, then hits in both edited sections.

- [ ] **Step 4: Commit**

```bash
git add skills/devpilot-resolve-issues/references/subagent-spec.md
git commit -m "feat(devpilot-resolve-issues): deterministic tier escalation on BLOCKED"
```

---

### Task 9: Sync installed copy of scanning-repos

`devpilot-scanning-repos` is also installed at `.claude/skills/devpilot-scanning-repos/`; `devpilot-resolve-issues` is not installed there (verified) — only `skills/` changes for it.

- [ ] **Step 1: Confirm the installed copy has not diverged** (do not clobber unknown local edits):

```bash
diff -rq skills/devpilot-scanning-repos .claude/skills/devpilot-scanning-repos
```

Expected: differences ONLY in the files this plan edited (SKILL.md, references/labels.md, references/issue-template.md, agents/*.md, scripts/check-findings.py, evals/evals.json). If other files differ, STOP and surface the divergence to the user instead of copying.

- [ ] **Step 2: Copy the changed files**

```bash
cp skills/devpilot-scanning-repos/SKILL.md .claude/skills/devpilot-scanning-repos/SKILL.md
cp skills/devpilot-scanning-repos/references/labels.md .claude/skills/devpilot-scanning-repos/references/labels.md
cp skills/devpilot-scanning-repos/references/issue-template.md .claude/skills/devpilot-scanning-repos/references/issue-template.md
cp skills/devpilot-scanning-repos/agents/*.md .claude/skills/devpilot-scanning-repos/agents/
cp skills/devpilot-scanning-repos/scripts/check-findings.py .claude/skills/devpilot-scanning-repos/scripts/check-findings.py
cp skills/devpilot-scanning-repos/evals/evals.json .claude/skills/devpilot-scanning-repos/evals/evals.json
```

- [ ] **Step 3: Verify zero diff**

```bash
diff -rq skills/devpilot-scanning-repos .claude/skills/devpilot-scanning-repos && echo IN-SYNC
```

Expected: `IN-SYNC`.

- [ ] **Step 4: Commit**

```bash
git add .claude/skills/devpilot-scanning-repos
git commit -m "chore: sync installed devpilot-scanning-repos copy"
```

---

### Task 10: Whole-change verification

- [ ] **Step 1: No stale contract phrasing anywhere in either skill**

```bash
grep -rn "exactly five\|five labels\|more capable model" skills/devpilot-scanning-repos skills/devpilot-resolve-issues .claude/skills/devpilot-scanning-repos
```

Expected: no matches.

- [ ] **Step 2: Validator end-to-end smoke** (rerun Task 5 step 4's positive file):

```bash
python3 skills/devpilot-scanning-repos/scripts/check-findings.py /tmp/model-field-test.json; echo "exit=$?"
```

Expected: `exit=0`.

- [ ] **Step 3: `skills/index.json` untouched** (no skill added/removed):

```bash
git diff main --stat -- skills/index.json
```

Expected: empty.

- [ ] **Step 4: Repo checks** (markdown-only change, but the repo gate is cheap):

```bash
make test && make lint
```

Expected: both pass (no Go changes — this is a regression guard).
