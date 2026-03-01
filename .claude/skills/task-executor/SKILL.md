---
name: devpilot:task-executor
description: Executes a task plan autonomously. Used by the devpilot runner to process Trello cards. Follows execution plans step-by-step using TDD and verification skills.
---

# Task Executor

Execute implementation plans autonomously. This skill is invoked by `devpilot run` via `claude -p`.

## Process

1. **Parse the plan** — Read the provided execution plan and identify each task/step.

2. **Execute step-by-step** — For each step in the plan:
   - If the step involves writing code, use the `superpowers:test-driven-development` skill
   - If the step involves debugging, use the `superpowers:systematic-debugging` skill
   - Follow exact file paths and commands from the plan
   - Commit after each logical unit of work

3. **Verify before completion** — Use the `superpowers:verification-before-completion` skill:
   - Run all tests
   - Run any verification commands specified in the plan
   - Confirm all changes compile and pass

4. **Commit and push** — Create descriptive commit messages for each change. Push to the current branch.

## Rules

- **Follow the plan exactly** — Do not add features, refactor, or deviate from what the plan specifies
- **Fail fast** — If a step is blocked and cannot be resolved, exit with a non-zero code and a clear error message to stderr
- **No interactive prompts** — This runs unattended. Never ask for user input.
- **Commit frequently** — Small, focused commits are better than one large commit
- **Push at the end** — Push all commits to the current branch when all steps are complete
