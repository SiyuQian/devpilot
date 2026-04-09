## Why

The current review architecture delegates all work to a single `claude -p` invocation — including cloning the repo, fetching the diff, reading project conventions, performing the review, and posting results to GitHub. This wastes ~40% of tokens on deterministic tasks Claude shouldn't do, produces unreliable GitHub API calls (Claude constructs complex `gh api` commands via prompt), and has no mechanism to filter false positives — the #1 pain point of AI code review.

## What Changes

- **Go-side pre-processing**: Gather PR metadata, diff, and project convention files in Go before invoking Claude, eliminating the need for Claude to clone repos or run `gh` commands
- **Multi-round review pipeline**: Replace single Claude invocation with a two-round pipeline — Round 1 (Sonnet) performs the review and outputs structured findings, Round 2 (Haiku) scores each finding for confidence (0-100)
- **Finding confidence filter**: Go-side filtering discards findings scoring below 50, outputting only findings the system is moderately-to-highly confident about
- **Go-side GitHub posting**: Move review posting logic from prompt instructions to Go code, eliminating unreliable prompt-driven `gh api` command construction
- **False positive guidance**: Add explicit false positive definitions (borrowed from proven patterns) to the review prompt so reviewers avoid common noise categories
- **Model tiering**: Use Sonnet for review (cost-effective, sufficient quality) and Haiku for scoring (cheap, fast), reserving Opus only when explicitly requested

## Capabilities

### New Capabilities
- `review-pipeline`: Orchestrates the multi-round review pipeline — Go-side context gathering, Sonnet review invocation, Haiku scoring invocation, confidence filtering, and result assembly
- `review-scoring`: Defines the Haiku scoring round — prompt, input/output format, confidence scale, and false positive definitions
- `review-posting`: Go-side GitHub review posting via `gh` CLI or API, replacing prompt-driven posting

### Modified Capabilities
- `review-command`: Updated to use the new pipeline instead of single Claude invocation; new `--threshold` flag for confidence cutoff (default 50); model flags now control review model (default Sonnet) vs scoring model (default Haiku)
- `review-prompt`: Restructured to receive pre-gathered context (diff, conventions) as input rather than instructing Claude to fetch them; removes repo clone and `gh` command instructions; adds false positive guidance and structured JSON output format for findings

## Impact

- **Code**: Major refactor of `internal/review/` — new pipeline orchestrator, new prompt files, new posting logic; executor gains support for multiple sequential invocations
- **Prompts**: `review-prompt.md` significantly rewritten; `review-posting.md` removed (replaced by Go code); new `review-scoring.md` added
- **CLI**: New `--threshold` flag; `--model` semantics change (now controls review model); new `--scoring-model` flag
- **Runner integration**: `review.Review()` API changes — runner code in `internal/taskrunner/` needs updating
- **Cost**: Expected ~60% reduction per review (smaller model + no wasted tool calls + pre-gathered context)
