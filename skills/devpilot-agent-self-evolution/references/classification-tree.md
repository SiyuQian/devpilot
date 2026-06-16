# Classification Tree

Maps each collected signal to a fix type and target artifact.

## Primary Decision Tree

```
SIGNAL RECEIVED
│
├─ Is this a lint/test/hook failure?
│  │
│  ├─ Does the same rule appear in ≥2 different PRs/commits?
│  │  │
│  │  ├─ YES → Check CLAUDE.md and all skill references for this rule
│  │  │        │
│  │  │        ├─ Rule NOT documented anywhere
│  │  │        │  └─> GUIDE GAP
│  │  │        │      Fix: add rule to CLAUDE.md (if cross-cutting) or
│  │  │        │           relevant skill reference (if domain-specific)
│  │  │        │
│  │  │        └─ Rule IS documented
│  │  │           └─> SENSOR GAP
│  │  │               Fix: add/strengthen mechanical check
│  │  │               (linter rule, pre-commit hook, or CI assertion)
│  │  │
│  │  └─ NO (single occurrence)
│  │     └─> WATCHING — record signal, no mutation yet
│  │         Exception: architecture invariant violations → always SENSOR GAP
│
├─ Is this a PR review comment?
│  │
│  ├─ Same comment (substantially) in ≥2 PRs?
│  │  │
│  │  ├─ Is the pattern mechanically checkable? (specific code pattern, import, naming)
│  │  │  └─> SENSOR GAP → add linter/test
│  │  │
│  │  └─ Is it a judgment call? (design preference, naming taste, architectural direction)
│  │     └─> GUIDE GAP → add to skill reference or CLAUDE.md
│  │
│  └─ Single comment
│     └─> WATCHING — record, do not mutate
│
├─ Is this an architecture invariant violation?
│  └─> Always SENSOR GAP
│      Fix: strengthen structural test or ArchUnit-style import check
│
└─ Is this a golden-principle regression across ≥2 sweeps?
   │
   ├─ Which specific principle regressed?
   │  │
   │  ├─ Is there a linter rule that could enforce it?
   │  │  └─> SENSOR GAP
   │  │
   │  └─ Is it purely a matter of awareness / convention?
   │     └─> GUIDE GAP
   │
   └─ Multiple principles regressed simultaneously
      └─> CONTEXT OVERLOAD — CLAUDE.md may be too long or stale
          Fix: audit CLAUDE.md length, split into skills, prune noise
```

## Fix Type → Target Artifact Map

| Classification | Fix Type | Target Artifact | Risk Level |
|---|---|---|---|
| GUIDE GAP (cross-cutting rule) | Append rule line | `CLAUDE.md` | HIGH |
| GUIDE GAP (domain-specific) | Add example or clarify | `skills/<name>/references/<file>.md` | LOW |
| SENSOR GAP (code pattern) | Add linter rule | `.golangci.yml` | MEDIUM |
| SENSOR GAP (pre-push check) | Add hook | `.claude/settings.json` hooks | HIGH |
| SENSOR GAP (structural) | Add import test | `<package>/<name>_test.go` | MEDIUM |
| CONTEXT OVERLOAD | Move rules out of CLAUDE.md | Create/update skill reference, trim CLAUDE.md | HIGH |

## Classification Examples

### Example 1: GUIDE GAP
**Signal:** `unused-variable` lint rule triggered in PR #42, #45, #51 (3 occurrences)
**Check CLAUDE.md:** no mention of unused variable handling
**Classification:** GUIDE GAP
**Target:** `CLAUDE.md` — append "Never leave unused variables; remove or use `_` explicitly"
**Risk:** HIGH (CLAUDE.md change)

### Example 2: SENSOR GAP
**Signal:** PR review comment "wrap errors at boundary: use fmt.Errorf" in PR #38 and PR #44
**Check CLAUDE.md:** rule exists at line 31: "Wrap errors at layer boundaries: `fmt.Errorf("doing X: %w", err)`"
**Classification:** SENSOR GAP (rule documented but not enforced)
**Target:** `.golangci.yml` — add `wrapcheck` linter
**Risk:** MEDIUM

### Example 3: WATCHING
**Signal:** single PR review comment "consider using a table-driven test here"
**Occurrences:** 1
**Classification:** WATCHING — add to signal log, no mutation
**Reason:** single subjective preference; need more evidence

### Example 4: CONTEXT OVERLOAD
**Signal:** CLAUDE.md is 112 lines; golden-principle sweep shows 4 regressions across different domains
**Classification:** CONTEXT OVERLOAD
**Action:** Audit CLAUDE.md — identify which rules belong in domain skills, move them, reduce line count below 100
**Risk:** HIGH (CLAUDE.md restructuring)
