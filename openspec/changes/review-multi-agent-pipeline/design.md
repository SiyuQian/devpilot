## Context

The current `devpilot review` uses a single `claude -p` invocation that handles everything: cloning the repo, fetching the diff, reading conventions, reviewing code, and posting results to GitHub. This wastes tokens on deterministic work and produces unreliable results — particularly for GitHub posting where Claude must construct complex `gh api` commands from prompt instructions.

The `code-review:code-review` skill demonstrates a proven pattern: multiple specialized agents with independent review perspectives, followed by confidence scoring and filtering. We adapt this pattern to `claude -p` constraints (no Agent tool) by using sequential invocations orchestrated from Go.

## Goals / Non-Goals

**Goals:**
- Reduce token cost by ~60% through Go-side pre-processing and model tiering
- Improve review signal-to-noise ratio via confidence scoring and false positive filtering
- Make GitHub posting reliable by moving it to Go code
- Maintain compatibility with the task runner's review integration

**Non-Goals:**
- Parallel agent invocations within a single review (adds complexity, `claude -p` has no Agent tool — sequential rounds are simpler)
- Changing the task runner's self-heal/fix loop (Fix() API remains unchanged)
- Supporting non-GitHub platforms (GitLab, Bitbucket)
- Adding new review dimensions beyond what the current prompt covers

## Decisions

### Decision 1: Two-round sequential pipeline over single invocation

Round 1 (Opus) produces structured findings as JSON. Round 2 (Haiku) scores each finding independently.

**Why Opus for review?** Code review is a reasoning-heavy task where missing a real bug is costlier than the token difference. Opus provides the deepest understanding of code semantics, cross-file implications, and subtle bugs. The cost is offset by eliminating wasted tokens on context gathering and posting.

**Why not single invocation with self-scoring?** Self-scoring is unreliable — the same model that produced a finding is biased toward defending it. An independent model provides genuine second-opinion filtering.

**Why not parallel agents?** `claude -p` doesn't support spawning sub-agents. We could launch multiple `claude -p` processes in parallel (like the skill's 5 agents), but the complexity isn't worth it for v1. Sequential is simpler and still captures the key benefit (scoring/filtering).

### Decision 2: Go-side context gathering via `gh` CLI

Go code runs `gh pr view` and `gh pr diff` to gather PR metadata and diff, then injects them directly into the prompt. Convention files are fetched via `gh api` for the target repo (or read from disk if cwd matches).

**Why `gh` CLI over GitHub API client?** The project already depends on `gh` for PR creation and merging. Adding a Go GitHub API client is unnecessary dependency weight. `gh` handles auth transparently.

### Decision 3: JSON output format for Round 1

Round 1 prompt instructs Claude to output findings as a JSON array, not markdown. This makes parsing deterministic — no more regex-based verdict extraction.

```json
{
  "summary": "...",
  "findings": [
    {
      "file": "path/to/file.go",
      "line": 42,
      "end_line": 45,
      "severity": "WARNING",
      "title": "Missing error check",
      "explanation": "...",
      "suggestion": "..."
    }
  ],
  "assessment": "..."
}
```

**Why JSON over markdown?** Markdown parsing is fragile (the current `IsApproved` function is a line-scanning heuristic). JSON parsing is deterministic. Claude reliably produces valid JSON when instructed.

### Decision 4: Go-primary, LLM-fallback posting

Go code constructs and executes `gh api` calls to post the review (primary path). If the Go posting fails (e.g., unexpected API error, auth issue), the system falls back to a Haiku invocation that receives the error message, the findings, and the diff, and attempts to post the review with adaptive error handling.

**Why Go-primary?** Most posting failures are predictable (line out of diff range, auth errors) and cheaper to handle in Go. Go can validate line ranges against the diff before posting, retry on transient errors, and move out-of-range findings to the review body — all without spending tokens.

**Why LLM-fallback?** Go can only handle errors we anticipated. When an unexpected error occurs (new GitHub API behavior, edge cases in diff format), an LLM can read the error, understand it, and adapt — something rigid code cannot do. Using Haiku for this keeps the cost minimal.

**Why Haiku for posting?** Posting is a formatting/API task, not a reasoning task. Haiku is sufficient to construct `gh api` commands and interpret error messages. No need for Sonnet or Opus.

### Decision 5: Confidence threshold default of 50

The `code-review` skill uses 80 as its threshold. We use 50 because:
- Our pipeline has only 2 rounds (not 5 parallel reviewers + scoring), so fewer findings are generated
- A threshold of 80 would filter too aggressively with a single reviewer
- Users can override via `--threshold`

Scale:
- 0-25: Almost certainly false positive
- 25-49: Possible issue but likely noise — filtered out
- 50-74: Real issue worth mentioning — included
- 75-100: High confidence, likely critical — included

### Decision 6: Diff chunking for large PRs

If the diff exceeds 30,000 characters (~500 lines), split into file-level chunks and review each chunk separately in Round 1. Merge all findings before Round 2 scoring.

**Why 30k chars?** Keeps Round 1 prompt well within Sonnet's effective attention range. Larger diffs see quality degradation even with models that can technically handle the context.

## Risks / Trade-offs

**[Latency increase from two rounds]** → Two sequential `claude -p` invocations add ~10-20s overhead. Mitigated by: Round 2 (Haiku) is fast (~2-5s per batch), and total token usage is lower so per-round time may decrease.

**[JSON output reliability]** → Claude occasionally produces invalid JSON. Mitigated by: wrapping output instruction with "respond with ONLY valid JSON, no markdown fences", and adding a JSON parse retry (re-prompt on parse failure, max 1 retry).

**[Diff chunking loses cross-file context]** → File-level chunks can't catch issues that span multiple files. Mitigated by: including full file list and PR summary in each chunk's prompt, so Claude has awareness of the broader change even when reviewing one file.

**[Review cost with Opus]** → Opus is more expensive per token than Sonnet. Mitigated by: Go-side pre-processing eliminates ~40% of wasted tokens (no clone, no gh commands), and Haiku handles the cheaper rounds (scoring, posting fallback). Net cost should be comparable or lower than the current single-Opus-does-everything approach. Users can override with `--model claude-sonnet-4-6-20250514` for cost-sensitive reviews.

**[Go posting fails on unforeseen errors]** → Go code can only handle predicted error patterns. Mitigated by: LLM fallback (Haiku) receives the error and adapts. The review text is already output to stdout, so a posting failure never loses the review itself.
