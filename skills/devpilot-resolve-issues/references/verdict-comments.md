# Verdict comment templates

Exactly one comment per verdict. Each template is the verbatim body for `gh issue comment <num> --body "$(cat <<'EOF' ... EOF)"`. Fill only the bracketed placeholders.

## FALSE-POSITIVE

Use after you have traced the code at current HEAD and the issue's premise is wrong. **The quote block is mandatory** — a FALSE-POSITIVE comment with no code excerpt is not a verdict, it's a dismissal.

```markdown
> Triaged by `devpilot-resolve-issues`. Verdict: **false-positive**.

## Why this is a false positive

<1–3 sentences: the specific reason the issue's premise does not hold at current HEAD. Reference behavior, not opinion.>

## What the code actually does

File: `<path>` at commit `<short-sha>`

```<language>
<5–15 lines quoted from the file at current HEAD, with line numbers>
```

<one sentence connecting the quoted code to the "why" above>

## Closing action

Closing with `wontfix`. If the scanner or reporter believes this is still an issue, reopen with a concrete reproduction (failing test, stack trace, or user-visible symptom) and I'll re-triage.
```

Then:

```bash
gh issue close <num> --reason "not planned" --comment ""   # comment already posted above
gh issue edit  <num> --add-label "wontfix" --remove-assignee @me
```

### False-positive classes — quote the code for each

| Class | What the quote must show |
|---|---|
| Already fixed | The commit/line that fixed it (link the commit SHA). |
| Wrong file | The file that the scanner confused this with, or the fact that the cited file doesn't contain the claimed pattern. |
| Misread of the code | The actual behavior contradicting the issue's claim. |
| Intentional / pre-existing | The test, comment, or design doc that documents the behavior as intended. |
| Scanner hallucination | The cited lines don't contain the claimed construct at all. |

If you can't produce a quote for one of these classes, your verdict isn't FALSE-POSITIVE — it's NEEDS-HUMAN.

## NEEDS-HUMAN

Use when the concern is real but fixing it requires judgment you don't have — business rules, contracts with external systems, or product decisions.

```markdown
> Triaged by `devpilot-resolve-issues`. Verdict: **needs human input**.

## What's real about this

<1–3 sentences confirming the issue has merit — what you traced in the code and why it matches.>

## Why I can't fix it autonomously

<1–3 sentences naming the missing context: business rules, API contract, user intent, data migration plan, etc.>

## Concrete questions

1. <specific question a human can answer in one line>
2. <specific question a human can answer in one line>
3. <optional third>

Unassigning so whoever owns this can pick it up. Ping me (or re-run `/resolve-issues`) once the questions are answered and I'll carry it through.
```

Then:

```bash
gh issue edit <num> --add-label "need:human" --remove-assignee @me
```

If the label doesn't exist yet, create it once per repo (yellow, short description):

```bash
gh label create "need:human" --color "fbca04" --description "Blocked on human input — business rules, product decision, or external contract" 2>/dev/null || true
```

Do not close the issue. Do not add `wontfix`. It stays open with the `need:human` label so maintainers can filter the backlog for what's blocked on them.

**This label applies to every NEEDS-HUMAN exit path**, not just the triage verdict in step 4:

- BLOCKED return from an implementer subagent (step 6b)
- Round-3 review failure on any task (step 6c)
- Second verification failure inside a subagent (step 6b)
- Final-verify failure in the worktree (step 7)

In the mid-fix escalation paths the branch is pushed as a draft first; the `need:human` label and the escalation comment still go on the **issue**, not the PR.

## Real → PR opened (posted after the PR is created)

Leave this comment on the issue after `devpilot-pr-creator` returns the PR URL, so subscribers see the trail without having to click into the PR.

```markdown
> Triaged by `devpilot-resolve-issues`. Verdict: **real**. PR opened: <pr-url>

The fix landed task-by-task with `superpowers:requesting-code-review` gating every task. Closing this issue will happen automatically on merge via `Closes #<num>`.
```

No labels change on the issue. `Closes #<num>` in the PR body does the closing on merge.

## Rules

- **One comment per verdict.** Never post two verdict comments on the same issue in the same run.
- **Never paraphrase the code** — always quote it verbatim with line numbers. Paraphrase is where false-positive reasoning rots.
- **Keep it short.** The maintainer should be able to accept or reject the verdict in under 30 seconds.
- **No emoji, no hedging language.** "Seems like", "maybe", "I think" — drop all of them. If you're hedging, the verdict isn't ready.
- **Do NOT edit the issue's title or body.** The author's words stay intact. All of your output is in the comment.
