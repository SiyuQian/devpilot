# Backend / Infra Bug Fix PR Template

```markdown
## Description

**Bug:** {What was broken — specific symptom}

**Cause:** {Root cause found in the diff}

**Fix:** {What the code change does to resolve it}

## Review Guide

**Start here:** {The file with the core fix}

## Verification

- [ ] `{test command}` passes
- [ ] `{lint command}` passes
- [ ] Bug no longer reproduces: {exact steps to verify}
- [ ] No regressions in related functionality

## For Reviewers (human)

- [ ] Self-review of the code
```

## Notes

- Detect the project's actual lint/test commands from `Makefile`, `pyproject.toml`, `go.mod`, or CI config
- Bug/Cause/Fix must be specific — reference actual file names and line behavior from the diff
