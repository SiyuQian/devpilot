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
| "Blocking finding, but I'll post as `COMMENT` to be polite." | Event follows severity: a Blocking finding goes with `REQUEST_CHANGES`. |
| "I'll list all findings in the body — easier to read than scrolling inline." | Findings tied to a line go inline so the author can act on each one in place. The body is for TL;DR, sweep summary, counts, and overall observations only. |
| "No clean line for this one, so I'll put it in the body." | Anchor to the most representative line and say so in the comment. The author can ask for a different anchor; they cannot resolve a body bullet. |
| "This is a code-quality nit, not behavior — skip it." | The skill runs both passes: behavior sweep *and* the quality checklist. Code quality, architecture, testing, requirements, production-readiness are in scope. |
| "Behavior trace is the whole point — checklist items are filler." | Behavior sweep is what makes the review more than style; the checklist is what makes it more than a behavior trace. Run both. |
| "I'll post the inline comments as standalone PR comments via `gh pr comment`." | Standalone comments aren't part of the review and don't show up in the right pane. Use one combined `POST .../pulls/:num/reviews` with body + `comments[]` + event. |
| "I'll split blockers and nits into two reviews." | One review per pass. The author gets one notification, one set of comments, one verdict. |
| "A small emoji softens the tone." | Keep the review in professional prose. The review is part of the PR record. |
| "Greeting feels redundant, skipping it." | The greeting is part of the template and addresses the author by handle. |
| "The version comment is noise, I'll drop it." | Keep `<!-- devpilot-pr-review (devpilot vX.Y.Z) -->` so readers can attribute the review. |
| "Disclaimer feels defensive, I'll skip or shorten it." | Keep the disclaimer. It protects authors from treating AI findings as authoritative. |
| "I'll ask 'what happens when X?' so the author clarifies." | If the code can answer it, state the answer. Author questions live in Open Questions only. |
| "I have a better approach but I'll stay neutral." | Name it, one sentence on why, ask the author to confirm. |

## Self-check before posting

Before running `gh pr review`, run through this list. A "yes" on any item means the review is not ready; fix the underlying issue and re-check.

- Writing findings before the five blind-spot questions were answered.
- Quality checklist (`checklist.md`) skipped — only the behavior sweep was run.
- Findings are all naming / formatting / "could be cleaner".
- Comparing two options the author already listed instead of surfacing ones they did not consider.
- "LGTM" without a single traced input.
- Only files in the diff were opened; no callers, tests, or configs.
- Findings tied to a line dumped into the body instead of attached as inline comments.
- An inline comment that repeats the file path or line number inside its text.
- A cross-cutting finding promoted to the body because "no line fit" — should have anchored to the most representative line.
- Author questions about behavior the code could have answered.
- Known-better alternatives hidden as vague questions.
- Findings missing a `Confidence` line.
- Review event does not match the highest-severity finding.
- Inline comments and body posted as separate API calls instead of one combined POST.
- Review is not in the PR's language end-to-end (including the disclaimer and inline-comment field labels).
