# Issue template

Exactly one `gh issue create` per surviving finding. Use this body verbatim, filling placeholders. The template is optimized so the maintainer can triage in < 30 seconds.

## `gh` invocation

```bash
# Resolve the five labels via the step-2 mapping table (canonical → label_to_apply).
# For area: first check /tmp/devpilot-existing-labels.json for a suitable existing
# area-ish label covering the finding's top-level dir. Reuse if suitable; otherwise:
area_dir=$(echo "<finding-file>" | cut -d/ -f1)
area_label="area:$area_dir"   # or the reused suitable existing label
# Only run create when no suitable repo label was found:
gh label create "$area_label" --color FEF2C0 --description "Auto-derived from finding file path"

gh issue create \
  --title "[scan:<category>] <title>" \
  --label "<category_label>,<subcategory_label>,<severity_label>,<confidence_label>,$area_label" \
  --body "$(cat <<'EOF'
<see body template below>
EOF
)"
```

- `<category_label>` etc. come from the step-2 mapping table — they are the *resolved* labels, which may be the canonical `{type}:{value}` form or a suitable existing repo label the orchestrator chose to reuse.
- `<category>` in the title prefix is always the canonical category value (`security` | `edge-case` | `coverage`), regardless of which label was reused. This keeps title-based dedupe (SKILL.md step 6) and notification filters stable across runs.
- Canonical `<subcategory>` is one of `sec:*`, `edge:*`, `cov:*` matching the category — see `references/labels.md` for the full enum. The scanner emits the canonical subcategory; the orchestrator does NOT pick freely.
- `<severity>` ∈ `high`, `medium`, `low`.
- `<score>` ∈ `75`, `100` (output of the scoring pass; sub-75 findings are already filtered).
- The title prefix `[scan:<category>]` keeps issues trivially filterable in notifications even when labels aren't visible.

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
- Area: `<area:...>`
- Scanner: `<security-scanner | edge-case-hunter | coverage-auditor>`
- Scored: `<score>/100` (`confidence:<score>`)

---

<sub>False positive? Close the issue with the `wontfix` label and a one-line reason. The scanner will learn from closed issues on the next scan (see dedupe step).</sub>
```

## Rules

- **Use `git rev-parse HEAD` for the SHA in the link** so the link survives future commits. Insert it literally into the body; do not use `$(...)` shell substitution inside the heredoc.
- **Never** include the full scanner output or the full scoring justification in the body. The template is the contract with the maintainer; extra content degrades scannability.
- **Labels are mandatory** — always exactly five, applied via the step-2 mapping table: a category, a matching subcategory, a severity, a confidence, and an auto-derived area. Canonical names are `scan:<category>`, `sec:*` | `edge:*` | `cov:*`, `severity:<level>`, `confidence:<score>`, `area:<top-level-dir>` — but the resolved label may be a suitable existing repo label instead. See `references/labels.md`.
- **One issue per finding.** Do NOT batch findings into a single issue, even for the same file.
- **Do not auto-assign** the issue. Let the maintainer triage.
