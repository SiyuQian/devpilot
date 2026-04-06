# Frontend Feature PR Template

```markdown
## Description

{What this feature adds and why, based on the actual diff}

## Review Guide

**Start here:** {The most important file/component to review first}

{Suggested review order, any tricky logic to watch for}

## Visual Changes

Before:

After:

## Verification

- [ ] `{lint command}` passes
- [ ] `{test command}` passes
- [ ] Tested in browser: {specific pages/flows to check}
- [ ] Responsive: tested at mobile, tablet, desktop widths
- [ ] Accessibility: keyboard navigation and screen reader tested

## For Reviewers (human)

- [ ] Self-review of the code
- [ ] Design matches spec/mockup
```

## Notes

- Detect the project's actual lint/test commands from `package.json` scripts, `Makefile`, or CI config
- Remove Visual Changes section if the feature has no visible UI change
- Remove Accessibility line if change doesn't affect interactive elements
