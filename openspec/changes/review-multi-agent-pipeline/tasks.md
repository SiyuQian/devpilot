## 1. Go-side Context Gathering

- [x] 1.1 Add `GatherContext(prURL string) (*ReviewContext, error)` function that runs `gh pr view` and `gh pr diff` to collect PR metadata and diff text
- [x] 1.2 Add convention file discovery — detect if cwd matches target repo (read from disk) or fetch via `gh api` (CLAUDE.md, linter configs)
- [x] 1.3 Add diff chunking logic — split diffs > 30k chars at file boundaries, preserving file list and PR summary per chunk

## 2. Review Prompt Restructuring

- [x] 2.1 Rewrite `review-prompt.md` to receive pre-gathered context (diff, metadata, conventions) as prompt input instead of instructing Claude to fetch them; add false positive guidance section
- [x] 2.2 Create `review-scoring.md` prompt for Round 2 — confidence scoring instructions with the 0-100 scale and false positive definitions
- [x] 2.3 Update `BuildPrompt()` to inject gathered context into the prompt; remove repo clone instructions and `gh` command instructions
- [x] 2.4 Add `BuildScoringPrompt(findings []Finding) string` function for Round 2 prompt assembly
- [x] 2.5 Remove `review-template.md` and `review-posting.md` embeds (replaced by JSON output and Go-side posting)

## 3. Pipeline Orchestration

- [x] 3.1 Define `Finding` and `ScoredFinding` structs and JSON parsing for Round 1 and Round 2 output
- [x] 3.2 Implement `Pipeline` struct with `Run(ctx, prURL) (*PipelineResult, error)` — orchestrates context gathering → Round 1 → parse → Round 2 → parse → filter → assemble
- [x] 3.3 Add JSON parse retry logic — if Round 1 or Round 2 output is invalid JSON, re-prompt once
- [x] 3.4 Add confidence threshold filtering — discard findings with score < threshold (default 50)
- [x] 3.5 Implement result assembly — convert `PipelineResult` to human-readable markdown for stdout and structured verdict for runner

## 4. Go-side GitHub Posting with LLM Fallback

- [x] 4.1 Add diff range parser — extract valid new-side line ranges per file from diff hunk headers (`@@`)
- [x] 4.2 Implement `PostReview(pr *PRInfo, result *PipelineResult) error` — construct and execute `gh api` call with review body and inline comments
- [x] 4.3 Add line range validation — move findings with out-of-range lines to review body instead of inline comments
- [x] 4.4 Add Haiku fallback — when Go posting fails, invoke Haiku with error message, findings, diff, and posting instructions to adaptively retry
- [x] 4.5 Create `review-posting-fallback.md` prompt for the Haiku fallback — includes error context and posting instructions

## 5. CLI and Integration Updates

- [x] 5.1 Add `--threshold` flag (default 50) and `--scoring-model` flag (default Haiku) to review command
- [x] 5.2 Update `Review()` function to use the new pipeline instead of single Claude invocation
- [x] 5.3 Keep `newReviewExecutor()` defaulting to Opus; add separate executor factories for scoring (Haiku) and posting fallback (Haiku)
- [x] 5.4 Update streaming progress — emit events for each pipeline phase (gathering context, reviewing, scoring, posting)
- [x] 5.5 Update task runner integration — ensure `review.Review()` API changes are reflected in `internal/taskrunner/`
- [x] 5.6 Update `IsApproved()` to use structured `PipelineResult` verdict instead of stdout text parsing

## 6. Tests

- [x] 6.1 Test context gathering — mock `gh` commands, verify metadata and diff parsing
- [x] 6.2 Test diff chunking — verify split at file boundaries, chunk size limits
- [x] 6.3 Test JSON parsing — valid findings, invalid JSON retry, empty findings
- [x] 6.4 Test confidence filtering — threshold edge cases (0, 50, 100), all filtered, none filtered
- [x] 6.5 Test diff range validation — findings inside/outside diff ranges, inline vs body placement
- [x] 6.6 Test posting fallback — Go posting fails, Haiku fallback invoked; both fail, graceful degradation
- [x] 6.7 Test pipeline end-to-end — mock all Claude invocations, verify full flow
