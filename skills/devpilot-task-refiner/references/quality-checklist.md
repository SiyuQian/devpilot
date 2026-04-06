# Plan Quality Checklist

Use this checklist to evaluate and improve Trello card plans. Read this file when performing refinement.

## Required Elements

Every plan must have:

- [ ] Clear title as `# Heading` (becomes Trello card name)
- [ ] Goal statement: one sentence describing what this builds
- [ ] Ordered implementation steps (numbered list)
- [ ] Each step specifies: what to do, which files to touch, how to do it
- [ ] Verification steps at the end (test commands, build checks)

## Quality Dimensions

### 1. Specificity

- [ ] Steps reference concrete file paths (e.g., `internal/config/config.go`)
- [ ] Steps include actual code snippets or describe exact changes
- [ ] No vague language: "update config" -> "add `TimeoutField` to `Config` struct in `internal/config/config.go`"

### 2. Executability

- [ ] Each step can be executed by Claude without human judgment
- [ ] No steps that say "decide how to..." or "figure out..."
- [ ] External dependencies are documented (APIs, tools, libraries)

### 3. Test Strategy

- [ ] Unit tests specified for new functions/methods
- [ ] Test file paths included (e.g., `internal/foo/foo_test.go`)
- [ ] Verification commands listed (e.g., `go test ./internal/foo/...`)
- [ ] Build check included (e.g., `go build ./...`)

### 4. Architecture Consistency

- [ ] Uses patterns already in the codebase (check via codebase analysis)
- [ ] Follows existing naming conventions
- [ ] Respects package boundaries
- [ ] Does not contradict decisions in `docs/plans/`

### 5. Edge Cases

- [ ] Error handling considered for each step
- [ ] Input validation at system boundaries
- [ ] Graceful degradation where appropriate

### 6. Dependency Order

- [ ] Steps are ordered so dependencies come first
- [ ] No circular dependencies between steps
- [ ] Shared utilities created before code that uses them

## Plan Template

When expanding a vague idea, generate a plan following this structure:

```
# [Feature Name]

**Goal:** [One sentence]

**Architecture:** [2-3 sentences about approach]

## Steps

### 1. [First component/change]

**Files:**
- Create: `path/to/new/file.go`
- Modify: `path/to/existing/file.go`
- Test: `path/to/file_test.go`

**What to do:**
[Concrete description with code snippets]

**Verification:**
- Run: `go test ./path/to/...`
- Expected: All tests pass

### 2. [Next component/change]
...

## Final Verification

- Run: `go test ./...`
- Run: `go build ./...`
- Manual check: [any manual verification needed]
```
