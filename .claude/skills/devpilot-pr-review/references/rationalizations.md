# Rationalizations and Self-Check

Common shortcuts the reviewer may reach for, and what to do instead. The "Reality" column is the rule.

## Table

| Excuse | Reality |
|---|---|
| "This PR is small, skip the five questions." | Small PRs change defaults, delete branches, flip flags. Answer all five; "N/A" is fine. |
| "The author explained intent in the description." | Intent ≠ actual behavior. Trace one input through the code. |
| "Diff looks clean, no need to look at callers." | Diff shows what changed, not the blast radius. Check Unknown-Unknowns Sweep question 2. |
| "I recognize the pattern, I can assert the right way." | Training can be 6–18 months stale. Do question 4 first. |
| "Writing a retry loop / cache / parser here is fine." | Usually there is a mature off-the-shelf option. Do question 5. |
| "LGTM, nothing jumps out." | Trace at least one input through at least one change first. |
| "This feels minor, I'll leave it out to keep the review tidy." | Report it with Confidence + Severity labels. Filtering happens downstream, not here. |
| "I'm unsure, I'll file it as Should-fix to be safe." | Severity is impact-if-true; Confidence is how sure you are. Keep them separate. |
| "I'll just print the review in chat, user can paste it." | Post by default. Only skip when the user opts out or no real PR exists. |
| "Blocking finding, but I'll post as `--comment` to be polite." | Post mode follows severity: a Blocking finding goes with `--request-changes`. |
| "A small emoji softens the tone." | Keep the review in professional prose. The review is part of the PR record. |
| "Greeting feels redundant, skipping it." | The greeting is part of the template and addresses the author by handle. |
| "The version comment is noise, I'll drop it." | Keep `<!-- devpilot-pr-review (devpilot vX.Y.Z) -->` so readers can attribute the review. |
| "Disclaimer feels defensive, I'll skip or shorten it." | Keep the disclaimer. It protects authors from treating AI findings as authoritative. |
| "I'll ask 'what happens when X?' so the author clarifies." | If the code can answer it, state the answer. Author questions live in Open Questions only. |
| "I have a better approach but I'll stay neutral." | Name it, one sentence on why, ask the author to confirm. |

## Self-check before posting

Before running `gh pr review`, run through this list. A "yes" on any item means the review is not ready; fix the underlying issue and re-check.

- Writing findings before the five blind-spot questions were answered.
- Findings are all naming / formatting / "could be cleaner".
- Comparing two options the author already listed instead of surfacing ones they did not consider.
- "LGTM" without a single traced input.
- Only files in the diff were opened; no callers, tests, or configs.
- Author questions about behavior the code could have answered.
- Known-better alternatives hidden as vague questions.
- Findings missing a `Confidence` line.
- Post mode does not match the highest-severity finding.
- Review is not in the PR's language end-to-end (including the disclaimer).
