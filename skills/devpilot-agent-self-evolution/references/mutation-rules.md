# Mutation Rules

Risk matrix for every harness artifact. Every mutation must pass the check in this table before being proposed.

## Artifact Risk Matrix

### CLAUDE.md
**Risk:** HIGH  
**Scope:** Injected into every agent prompt. Changes affect all future agent behavior immediately.

| Allowed mutations | Requires explicit user approval |
|---|---|
| Append a new rule line | YES — always |
| Clarify an existing rule (same meaning, clearer wording) | YES |
| Move a rule OUT to a skill reference | YES (two-step: add to skill first, then remove from CLAUDE.md) |
| Delete a rule | YES + user must state the reason |

**Hard stops:**
- Count current lines before proposing any addition: `wc -l CLAUDE.md`
- If count ≥ 100: do not propose an addition. Instead, propose moving an existing rule to a skill reference to make room.
- Never add more than 3 lines in a single mutation.

**Trigger signals:** Cross-cutting convention violations (affects ≥2 unrelated domains), repeated violations of a rule that has never been documented.

**Example mutation:**
```diff
 ## Conventions the agent keeps getting wrong
+- Never leave unused variables; remove them or use `_` explicitly.
```

---

### skills/\<name\>/SKILL.md
**Risk:** MEDIUM  
**Scope:** Loaded only when this specific skill is invoked. Changes are scoped.

| Allowed mutations | Requires user approval |
|---|---|
| Clarify trigger description (frontmatter) | RECOMMENDED |
| Add a new workflow step | RECOMMENDED |
| Restructure sections | RECOMMENDED |
| Delete a section | YES |

**Never:** Change the `name:` frontmatter field (breaks the slash command).

**Trigger signals:** Skill consistently misapplied (wrong trigger fires it), workflow produces wrong artifact.

---

### skills/\<name\>/references/\*.md
**Risk:** LOW  
**Scope:** On-demand reference. Only loaded when the skill explicitly reads it. Minimal blast radius.

| Allowed mutations | Requires user approval |
|---|---|
| Append an example | OPTIONAL (auto-commit if CI green) |
| Add a clarifying note | OPTIONAL |
| Add a new section | OPTIONAL |
| Rewrite an existing section | RECOMMENDED |
| Delete a section | YES |

**Trigger signals:** Domain-specific repeated pattern not covered by existing examples, review comment pointing to a missing example in a specific skill's domain.

---

### .golangci.yml
**Risk:** MEDIUM  
**Scope:** Runs on every `make lint`. A wrong rule will fail all PRs until removed.

| Allowed mutations | Requires user approval |
|---|---|
| Add a new linter to `enable:` list | RECOMMENDED |
| Add rule-specific configuration | RECOMMENDED |
| Increase strictness of existing rule | RECOMMENDED |
| Decrease strictness | YES (weakening a sensor) |
| Disable a linter | YES |

**After mutation:** Always run `make lint` immediately. If any existing code fails the new rule, either fix the violations first (preferred) or add a `nolint` comment with a TODO. Never land a new linter rule that breaks the current build.

**Trigger signals:** SENSOR GAP where the pattern is mechanically checkable.

---

### .claude/settings.json (hooks)
**Risk:** HIGH  
**Scope:** Hooks run on every session. A broken hook can prevent all agent sessions from starting.

| Allowed mutations | Requires user approval |
|---|---|
| Add a new PostToolUse hook | YES — always |
| Add a new PreToolUse hook | YES — always |
| Modify hook command | YES |
| Delete a hook | YES |

**Safety check:** After adding a hook, start a new session and verify it runs without error before committing.

**Trigger signals:** SENSOR GAP where a check must run after every tool use (not just at commit time), e.g., catching an agent that consistently edits the wrong file type.

---

### Test files (\*\_test.go)
**Risk:** MEDIUM  
**Scope:** Affects CI gate. A broken test blocks all PRs.

| Allowed mutations | Requires user approval |
|---|---|
| Add a new structural/invariant test | RECOMMENDED |
| Add coverage for an uncovered path | OPTIONAL |
| Modify assertion thresholds | RECOMMENDED |
| Delete a test | YES |

**After mutation:** Always run `make test` immediately.

**Trigger signals:** SENSOR GAP for architecture invariants (package boundary violations, forbidden imports).

---

## Commit Message Convention

Every mutation committed by this skill uses the prefix `harness:`:

```
harness: add unused-variable rule to CLAUDE.md (PR #42, #45, #51)
harness: add wrapcheck linter for error wrapping sensor (PR #38, #44)
harness: move error-wrapping rule from CLAUDE.md to go-style reference
```

The commit message must name the signal source so it's traceable.
