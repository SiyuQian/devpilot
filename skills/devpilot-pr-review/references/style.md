# Tone, Stance, and Language

Rules that govern how the review reads once posted on the PR. **They apply equally to the body and to every inline comment** — both are part of the same review and the same review record.

## Language

Render every section (TL;DR, Unknown-Unknowns Sweep, findings, disclaimer, Open Questions, metadata) and every inline comment in the PR's language. Chinese PR → Chinese review, end to end, including the per-finding field labels (`Behavior today on this branch:`, etc.). Translate the blockquote disclaimer while preserving its meaning: automated, not authoritative, human judgment required.

## Tone

Write in professional prose. Skip emoji, exclamation marks, and softeners like "just a thought", "maybe", or "could be wrong but". Greet the PR author by their resolved handle:

```bash
gh pr view <url> --json author -q .author.login
```

Render as `@handle`. Fall back to "Hi there," when the handle is unavailable.

## Stance

- **State system behavior as claims, not questions.** A traced claim ("This recurses on a 401 from `/refresh` and will stack-overflow") belongs in Behavior Findings. The corresponding question belongs in `Open Questions` only when the code could not answer it.
- **When you see a concrete alternative, name it.** One sentence on why it is better, and ask the author to confirm the direction. Recommendations do not belong inside vague questions.
- **Confidence and severity are independent axes.** Every finding carries `Confidence: high | medium | low`, independent of severity. Low confidence never automatically demotes severity — a high-severity bug you are moderately sure about is still `Severity: Blocking, Confidence: medium`.
- **Open Questions holds only what the code could not answer** (e.g. callers outside the repo, product intent, runtime scale). Lives in the body; omit the section entirely when empty.

## Inline-comment specifics

- **Don't repeat the path or line number inside the comment text** — both are already part of the inline anchor metadata. Lead with the severity-tagged title.
- **Each comment is self-contained.** Don't write "see comment above" or "as I said in another finding"; the author may resolve them in any order.
- **Keep it under one screen.** Long architectural arguments belong in the body. If a finding genuinely needs more than ~8 lines, split it or move the framing to the body and keep the inline comment focused on the line.
- **Suggested change is concrete.** Name the package, the helper, the function, the diff direction — not "consider refactoring".
