# Backend / Infra Feature PR Template

```markdown
## Description

{What this feature adds and why, based on the actual diff}

## Review Guide

**Start here:** {The most important file/module to review first}

{Suggested review order, any tricky logic or architectural decisions to watch for}

## Verification

- [ ] `{test command}` passes
- [ ] `{lint command}` passes
- [ ] API tested: {specific endpoints or commands to exercise}
- [ ] Migration tested (if applicable): {migration steps}

## Additional Notes

{Deploy dependencies, feature flags, environment variables, follow-up work — remove if nothing}

## For Reviewers (human)

- [ ] Self-review of the code
- [ ] Checked for security implications
```

## Notes

- Detect the project's actual lint/test commands from `Makefile`, `pyproject.toml`, `go.mod`, or CI config
- Remove Migration line if no migrations
- Remove Additional Notes section if nothing to say
