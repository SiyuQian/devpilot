# Issue template

Exactly one `gh issue create` per surviving finding. Use this body verbatim, filling placeholders. The template is optimized so the maintainer can triage in < 30 seconds.

## `gh` invocation

```bash
# Derive area label from the first path segment of the finding's file
area_label="area/$(echo "<finding-file>" | cut -d/ -f1 | tr '/' '-')"
gh label create "$area_label" --color FEF2C0 --description "Auto-derived from finding file path" || true

gh issue create \
  --title "[scan/<category>] <title>" \
  --label "scan/<category>,<subcategory>,severity/<severity>,confidence/<score>,$area_label" \
  --body "$(cat <<'EOF'
<see body template below>
EOF
)"
```

- `<category>` ∈ `security`, `edge-case`, `coverage`.
- `<subcategory>` is one of `sec/*`, `edge/*`, `cov/*` matching the category — see `references/labels.md` for the full enum. The scanner emits the subcategory in its finding (see Finding schema); the orchestrator does NOT pick freely.
- `<severity>` ∈ `high`, `medium`, `low`.
- `<score>` ∈ `75`, `100` (output of the scoring pass; sub-75 findings are already filtered).
- The title prefix `[scan/<category>]` keeps issues trivially filterable in notifications even when labels aren't visible.

## Body template

```markdown
> Filed by `devpilot-scanning-repos`. Confidence ≥ 75/100. Business logic was NOT evaluated.

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

- Category: `<category>` / `<subcategory>`
- Severity: `<severity>`
- Area: `<area/...>`
- Scanner: `<security-scanner | edge-case-hunter | coverage-auditor>`
- Scored: `<score>/100` (`confidence/<score>`)

---

<sub>False positive? Close the issue with the `wontfix` label and a one-line reason. The scanner will learn from closed issues on the next scan (see dedupe step).</sub>
```

## Rules

- **Use `git rev-parse HEAD` for the SHA in the link** so the link survives future commits. Insert it literally into the body; do not use `$(...)` shell substitution inside the heredoc.
- **Never** include the full scanner output or the full scoring justification in the body. The template is the contract with the maintainer; extra content degrades scannability.
- **Labels are mandatory** — always exactly five: `scan/<category>`, one matching subcategory (`sec/*` | `edge/*` | `cov/*`), `severity/<level>`, `confidence/<score>`, and one auto-derived `area/<top-level-dir>`. See `references/labels.md`.
- **One issue per finding.** Do NOT batch findings into a single issue, even for the same file.
- **Do not auto-assign** the issue. Let the maintainer triage.
