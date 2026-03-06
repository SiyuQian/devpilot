## 1. Prompt and Plan Parsing

- [x] 1.1 Update `prompts/commit.tmpl` to include full diff content, request JSON output with `commits` and `excluded` arrays, and add truncation markers for large diffs
- [x] 1.2 Create `commit_plan.go` with `CommitPlan` struct (commits + excluded), JSON parsing with fallback to single-commit on parse failure, and plan validation (check all files exist in staged changes, no files missing)
- [x] 1.3 Update `commitData` struct and `buildCommitPrompt` to accept and truncate full diff content (200 lines per file, 15K chars total), handle binary files as `Binary file: <path>`

## 2. Bubble Tea Model and Phases

- [x] 2.1 Create `commit_model.go` with `CommitModel` struct: phases (staging, analyzing, plan, executing, done), commit plan state, execution progress tracking, and key handling (a/y/e/n)
- [x] 2.2 Implement `Init` and `Update` — phase transitions, spawn async commands for git operations and claude generation, handle results as tea.Msg types
- [x] 2.3 Implement `[e]dit` flow: serialize plan to human-readable markdown, open `$EDITOR`, re-parse edited result back into CommitPlan

## 3. View Rendering

- [x] 3.1 Create `commit_view.go` with lipgloss styles matching taskrunner conventions (color 12 primary, 10 success, 9 error, 240 muted, RoundedBorder panels)
- [x] 3.2 Implement analyzing phase view: animated spinner with "Analyzing changes..." text
- [x] 3.3 Implement plan phase view: numbered commit list with colored type/scope, file status indicators (M/A/D), per-file +/- stats, excluded section, and prompt line
- [x] 3.4 Implement executing phase view: checkmarks for completed commits, spinner for current, circle for pending
- [x] 3.5 Implement done phase view: summary with short hash, first line of message, and file change statistics per commit

## 4. Git Operations

- [x] 4.1 Implement diff collection: `git add .`, `git diff --cached` (full), `git diff --cached --name-status`, `git diff --cached --stat` — package as helper functions
- [x] 4.2 Implement plan execution: for each commit in plan, `git reset HEAD -- .` then `git add <files>` then `git commit -m <message>`, capture short hash from `git rev-parse --short HEAD`
- [x] 4.3 Implement excluded file handling: `git reset HEAD -- <excluded files>` before starting commit sequence
- [x] 4.4 Handle mid-sequence failure: stop execution, report which commits succeeded, exit non-zero

## 5. Integration and Entry Point

- [x] 5.1 Rewrite `RunCommit` in `commit.go` to launch `tea.Program` with the new `CommitModel`, pass through `--dry-run` and `--model` flags
- [x] 5.2 Support dry-run mode: show plan and exit with "(dry-run: not committing)" without executing
- [x] 5.3 Add tests for plan parsing (valid JSON, malformed JSON fallback, file validation) and diff truncation logic
