# Review Output Format

Structure your review output exactly as follows:

## Summary

[1-2 sentences describing what this PR does and your overall impression]

## Verdict

[Exactly one of: APPROVE or REQUEST_CHANGES]

[APPROVE if no CRITICAL findings — even if there are WARNINGs or SUGGESTIONs]
[REQUEST_CHANGES only if there are CRITICAL findings — list them here]

## Findings

[For each file with findings, use this format:]

### `path/to/file.ext`

**[SEVERITY]** Line N[-M]: [Brief title]

[Explanation of the issue and why it matters]

```suggestion
[Concrete fix if applicable]
```

[Repeat for each finding in this file]

[Repeat ### block for each file with findings]

[If no findings in a file, skip it entirely — do not list files with no issues]

## Overall Assessment

[2-3 sentences on code quality, test coverage, and any architectural observations. Note any praiseworthy patterns.]
