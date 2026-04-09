## ADDED Requirements

### Requirement: Multi-round review pipeline
The system SHALL orchestrate a two-round review pipeline: Round 1 invokes a review model to produce structured findings, Round 2 invokes a scoring model to score each finding, then Go-side logic filters findings by confidence threshold.

#### Scenario: Standard review flow
- **WHEN** `Review()` is called with a PR URL
- **THEN** the system gathers context (diff, metadata, conventions) via Go, invokes Round 1 (review), parses JSON findings, invokes Round 2 (scoring), filters by threshold, and returns the assembled result

#### Scenario: No findings from Round 1
- **WHEN** Round 1 produces zero findings
- **THEN** the system skips Round 2 and returns an APPROVE result with summary and assessment only

#### Scenario: All findings filtered out
- **WHEN** Round 2 scores all findings below the threshold
- **THEN** the system returns an APPROVE result, noting that findings were identified but fell below the confidence threshold

### Requirement: Go-side context gathering
The system SHALL gather PR metadata, diff, and project convention files in Go before invoking Claude.

#### Scenario: Gather PR metadata
- **WHEN** the pipeline starts
- **THEN** Go code runs `gh pr view <url> --json title,body,baseRefName,headRefName,author` and parses the result

#### Scenario: Gather diff
- **WHEN** the pipeline starts
- **THEN** Go code runs `gh pr diff <url>` and captures the full diff text

#### Scenario: Gather convention files from remote repo
- **WHEN** the PR's target repo is not the user's cwd
- **THEN** Go code fetches convention files (CLAUDE.md, linter configs) via `gh api repos/{owner}/{repo}/contents/{path}?ref={base}`

#### Scenario: Gather convention files from local repo
- **WHEN** the user's cwd is a git checkout of the same repo as the PR
- **THEN** Go code reads convention files from disk instead of fetching via API

### Requirement: Diff chunking for large PRs
The system SHALL split diffs exceeding 30,000 characters into file-level chunks, reviewing each chunk independently in Round 1.

#### Scenario: Small diff
- **WHEN** the diff is under 30,000 characters
- **THEN** the system sends the entire diff in a single Round 1 invocation

#### Scenario: Large diff split into chunks
- **WHEN** the diff exceeds 30,000 characters
- **THEN** the system splits by file boundaries and invokes Round 1 separately for each chunk, including the PR summary and full file list for cross-file awareness

#### Scenario: Findings merged across chunks
- **WHEN** multiple chunks produce findings
- **THEN** all findings are merged into a single list before Round 2 scoring

### Requirement: Result assembly
The system SHALL assemble the final review output from pipeline results in both structured (for runner integration) and human-readable (for CLI output) formats.

#### Scenario: Structured result for runner
- **WHEN** the runner calls `Review()`
- **THEN** the returned result includes a machine-parseable verdict (APPROVE or REQUEST_CHANGES) derived from post-filter findings: REQUEST_CHANGES if any finding with severity CRITICAL scores ≥ threshold, APPROVE otherwise

#### Scenario: Human-readable output for CLI
- **WHEN** the CLI user runs `devpilot review`
- **THEN** stdout displays a formatted markdown review with Summary, Verdict, Findings (with scores), and Overall Assessment
