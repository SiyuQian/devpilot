## Context

`devpilot commit` currently runs `git add .` → sends file names + stat to `claude --print` → gets a single commit message → prompts y/n/e → commits. The generate package (`internal/generate/`) uses Go templates for prompts and shells out to `claude --print` for AI generation. The project already uses Bubble Tea + lipgloss in `internal/taskrunner/` for the runner TUI dashboard.

## Goals / Non-Goals

**Goals:**
- Claude analyzes full diff content and returns a structured JSON plan with multiple atomic commits
- Dangerous files (secrets, build artifacts) are automatically identified and excluded
- Interactive Bubble Tea UI with spinner, formatted plan, execution progress, and summary
- Reuse existing lipgloss color scheme and style patterns from taskrunner TUI

**Non-Goals:**
- Hunk-level splitting (splitting a single file across commits) — file-level only for v1
- Custom exclusion rules or `.devpilotignore` config file
- Streaming AI output (we use `claude --print` which buffers)
- Replacing the taskrunner TUI or extracting shared TUI components into a library

## Decisions

### 1. Claude output format: JSON with commits + excluded arrays

Claude returns:
```json
{
  "commits": [
    {
      "message": "feat(auth): add token refresh\n\n- Add refresh flow\n- Handle expiry",
      "files": ["internal/auth/oauth.go", "internal/auth/refresh.go"]
    }
  ],
  "excluded": [
    {"file": ".env.local", "reason": "Contains secrets"}
  ]
}
```

**Why JSON over freeform text**: Reliable parsing. The prompt instructs Claude to output only JSON. `cleanOutput` already strips markdown fences; we add JSON unmarshalling with a fallback that wraps all files into a single commit if parsing fails.

**Alternative considered**: YAML or custom format — JSON is simpler to parse in Go and Claude is very reliable at generating it.

### 2. Diff truncation strategy

Send full `git diff --cached` content, but truncate to stay within reasonable token limits:
- Per-file: truncate individual file diffs to first 200 lines
- Total: cap at 15,000 characters total diff content
- When truncated: append `[truncated — N more lines]` so Claude knows context is missing
- Binary files: skip diff content, show only `Binary file: <path>`

**Why these thresholds**: `claude --print` with Haiku/Sonnet handles ~30K tokens input comfortably. 15K chars of diff ≈ 5K tokens, leaving room for prompt template, file list, and generation.

**Alternative considered**: Sending only stat for large diffs — but that's the current broken behavior. Truncated diff is still much better than no diff.

### 3. Bubble Tea as a lightweight phase machine

The commit TUI model has 5 phases:

```
staging → analyzing → plan → executing → done
```

Each phase renders differently. This is simpler than the taskrunner TUI (no event channels, no polling) because the commit flow is synchronous and short-lived.

**Key difference from taskrunner TUI**: The taskrunner uses event channels + `waitForEvent` for long-running async operations. The commit TUI uses `tea.Cmd` functions that return messages when git/claude operations complete — simpler because each phase has exactly one async operation.

### 4. Execution: sequential git add + commit per plan entry

For each commit in the plan:
1. `git reset HEAD -- .` (unstage everything)
2. `git add <files...>` (stage only this commit's files)
3. `git commit -m <message>`

**Why reset-then-add instead of incremental**: Avoids state bugs where a failed mid-sequence commit leaves partial staging. Each commit starts from a clean index.

Excluded files are simply never staged — after the initial `git add .` and analysis, we `git reset` excluded files before starting the commit sequence.

### 5. File layout

```
internal/generate/
├── commit.go          → RunCommit entry point (rewritten)
├── commit_model.go    → Bubble Tea Model + Update
├── commit_view.go     → View rendering (lipgloss styles)
├── commit_plan.go     → JSON parsing, plan validation
├── claude.go          → Generate function (unchanged)
└── prompts/
    └── commit.tmpl    → Updated prompt template
```

### 6. Single-commit simplification

When the plan contains exactly one commit and no exclusions, the UI simplifies:
- Skip the numbered list format
- Show message directly with file list below
- Prompt shows `[y]es / [e]dit / [n]o` instead of `[a]ccept all / [e]dit / [n]o`

This keeps the common case (small, focused changes) feeling lightweight.

## Risks / Trade-offs

- **[JSON parse failure]** → Fallback: wrap all files in a single commit, use raw Claude output as message. Log a warning. User can still edit.
- **[Claude lists wrong files]** → Validation: check every file in plan exists in `git diff --cached --name-only`. Reject plan entries with unknown files and warn user.
- **[Excluded file false positives]** → User sees exclusions in the plan and can override via `[e]dit` (move files from excluded back into a commit group).
- **[Mid-sequence commit failure]** → Stop immediately, show which commits succeeded and which failed. Already-committed changes remain (they're valid commits). User can fix and re-run.
- **[Large monorepo performance]** → `git diff --cached` on thousands of files could be slow. Acceptable for v1; most devpilot users work on small-to-medium repos.
