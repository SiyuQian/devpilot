# Guardrails

Safeguards that prevent runaway self-modification. Check all of these before applying any mutation.

---

## Guardrail 1: Evidence Required

**Risk:** Hallucinated or speculative mutations that don't match real failures.

**Check:** Before proposing any mutation, verify that the signal record has:
- A concrete `evidence` field with at least one exact quote (lint line, PR comment text, commit hash)
- Occurrence count meeting the threshold in `signal-types.md`

**Blocking logic:** If a proposed mutation has no concrete citation, reject it with:
> "No evidence citation for this change. Provide a lint line number, PR comment URL, or commit hash before adding a rule."

---

## Guardrail 2: CLAUDE.md Line Limit

**Risk:** CLAUDE.md bloat. Every line is injected into every agent prompt. Bloated CLAUDE.md degrades all agent performance.

**Check:**
```bash
wc -l CLAUDE.md
```

**Blocking logic:** If current line count ≥ 100:
> "CLAUDE.md is at N lines (limit: 100). To add a new rule, first identify which existing rule should move to a skill reference. Propose that move first, then add the new rule."

Never add to CLAUDE.md if it would exceed 100 lines. The move-then-add pattern is mandatory.

---

## Guardrail 3: Modification Frequency Limit

**Risk:** Self-referential loops where the skill keeps modifying the same file.

**Check:** Before mutating any artifact, scan recent git log for this skill's commits:
```bash
git log --oneline --since="24 hours ago" --grep="harness:" -- <target-file>
```

**Blocking logic:** If the same file has ≥3 `harness:` commits in the last 24 hours:
> "This file has been modified 3 times by harness evolution in the last 24h. Pausing to prevent runaway modification. Review the recent changes and confirm you want to continue."

Await explicit user confirmation before proceeding.

---

## Guardrail 4: Append-Only for Existing Rules

**Risk:** Accidentally deleting or overwriting an existing rule that was intentional.

**Check:** When editing CLAUDE.md or any skill file, diff the proposed change against the current content. Verify that no existing lines are removed.

**Blocking logic:** If the proposed mutation removes or substantially rewrites an existing rule:
> "This mutation removes existing rule at line N. Deletions require explicit user instruction. Confirm: should I remove '<existing rule>'?"

Only proceed with deletion after the user explicitly states the reason.

---

## Guardrail 5: No Cross-Skill Contamination

**Risk:** Adding a rule to a skill that doesn't own that domain, creating duplicated or conflicting guidance.

**Check:** Before adding a rule to a skill reference, verify:
- The skill's `SKILL.md` description covers this domain
- The rule doesn't already exist in another skill's references
- If the rule is cross-cutting (applies to all domains), it belongs in CLAUDE.md, not a skill

**Blocking logic:** If the rule already exists in another skill:
> "This rule already appears in <other-skill>/references/<file>.md. Do not duplicate. If the existing coverage is insufficient, propose a clarification there instead."

---

## Guardrail 6: CI Must Pass After Mutation

**Risk:** A mutation (especially linter additions) breaks the existing build, blocking all PRs.

**Check:** After every mutation, immediately run:
```bash
make lint && make test
```

**Blocking logic:** If CI fails after mutation:
1. Revert the change immediately
2. Report the failure:
   > "Mutation caused CI failure: <error>. Reverting. Diagnose before re-proposing."
3. Do not attempt a different mutation until the failure is understood

Never commit a mutation that leaves CI broken.

---

## Guardrail 7: Self-Referential Loop Detection

**Risk:** The skill modifies its own SKILL.md or references in a loop.

**Check:** If the proposed mutation target is any file inside `skills/haiku-agent-self-evolution/` or `.claude/skills/haiku-agent-self-evolution/`:
> "This mutation targets the self-evolution skill itself. Self-referential changes require explicit user review. Show the proposed change and await confirmation."

This is not a hard block — self-improvement of the skill is valid — but it requires explicit user oversight every time.
