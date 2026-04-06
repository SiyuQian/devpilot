# Frontend Bug Fix PR Template

```markdown
## Description

**Bug:** {What was broken — specific symptom, not "it didn't work"}

**Cause:** {Root cause found in the diff}

**Fix:** {What the code change does to resolve it}

## Review Guide

**Start here:** {The file with the core fix}

{Any related files that changed, why they needed to change too}

## Verification

- [ ] `{lint command}` passes
- [ ] `{test command}` passes
- [ ] Bug no longer reproduces: {exact steps to verify}
- [ ] No visual regressions in related pages

## For Reviewers (human)

- [ ] Self-review of the code
```

## Notes

- Detect the project's actual lint/test commands from `package.json` scripts, `Makefile`, or CI config
- Bug/Cause/Fix must be specific — reference actual file names and line behavior from the diff
