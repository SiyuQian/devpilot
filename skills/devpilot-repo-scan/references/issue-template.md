# Issue template

Exactly one `gh issue create` per surviving finding. Use this body verbatim, filling placeholders. The template is optimized so the maintainer can triage in < 30 seconds.

## `gh` invocation

```bash
gh issue create \
  --title "[repo-scan:<category>] <title>" \
  --label "repo-scan,scan:<category>,severity:<severity>" \
  --body "$(cat <<'EOF'
<see body template below>
EOF
)"
```

`<category>` ∈ `security`, `edge-case`, `coverage`. `<severity>` ∈ `high`, `medium`, `low`. The title prefix makes these trivially filterable in the issue list and in notifications.

## Body template

```markdown
> Filed by `devpilot-repo-scan`. Confidence ≥ 75/100. Business logic was NOT evaluated.

## What

<why_it_matters, verbatim from the finding — 1–3 sentences>

## Where

<repo-relative/path.go> — <line_range>

https://github.com/<owner>/<repo>/blob/<full-sha>/<path>#L<start>-L<end>

## Evidence

```<language>
<evidence block, verbatim, with line numbers>
```

## Suggested fix

<suggested_fix, or "No confident fix proposed — needs human judgment." if null>

## Metadata

- Category: `<category>`
- Severity: `<severity>`
- Scanner: `<security-scanner | edge-case-hunter | coverage-auditor>`
- Scored: `<score>/100`

---

<sub>False positive? Close the issue with the `wontfix` label and a one-line reason. The scanner will learn from closed issues on the next scan (see dedupe step).</sub>
```

## Rules

- **Use `git rev-parse HEAD` for the SHA in the link** so the link survives future commits. Insert it literally into the body; do not use `$(...)` shell substitution inside the heredoc.
- **Never** include the full scanner output or the full scoring justification in the body. The template is the contract with the maintainer; extra content degrades scannability.
- **Labels are mandatory** — always three of them: `repo-scan`, `scan:<category>`, `severity:<severity>`.
- **One issue per finding.** Do NOT batch findings into a single issue, even for the same file.
- **Do not auto-assign** the issue. Let the maintainer triage.
