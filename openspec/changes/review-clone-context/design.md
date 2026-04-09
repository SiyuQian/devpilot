## Context

The review command currently gathers project context (CLAUDE.md, linter configs, etc.) via Go code in `context.go`. It makes up to 10 GitHub API calls per review with a hardcoded file list, silently swallows errors, and cannot discover files outside the list. Claude already has `--allowedTools=*` during review execution and can read files, run git commands, etc.

## Goals / Non-Goals

**Goals:**
- Eliminate GitHub API calls for context gathering (zero rate limit consumption)
- Let Claude dynamically discover any relevant convention/config files
- Fail loudly when repo access is denied (clone failure = clear error)
- Simplify the Go codebase by removing ~130 lines of context gathering code

**Non-Goals:**
- Changing the review prompt criteria or output template
- Changing how `IsApproved()` / verdict parsing works
- Changing the executor or how `claude -p` is invoked
- Supporting non-GitHub repos

## Decisions

### Decision 1: Clone to `/tmp/{owner}-{repo}` with reuse

Clone the repo to a predictable temp path. If the directory already exists, fetch and checkout the base branch instead of re-cloning. This avoids re-downloading large repos on repeated reviews.

The clone/update instructions go into `review-prompt.md` as part of Claude's task, not as Go code. Claude runs `git clone` or `git fetch` + `git checkout` itself.

**Alternative considered**: Shallow clone (`--depth 1`) — saves bandwidth but prevents checking out other branches. Since we checkout the base branch, a regular clone is needed. We could use `--single-branch` to limit scope.

### Decision 2: Context discovery delegated to Claude via prompt

Instead of Go code reading specific files, the review prompt instructs Claude to search the cloned repo for convention files (CLAUDE.md, AGENTS.md, linter configs, etc.) and read any it finds. This is flexible — Claude can also notice other relevant files like `.editorconfig`, `Makefile`, `CONTRIBUTING.md`, etc.

**Alternative considered**: Using `gh api repos/{o}/{r}/git/trees/{ref}` to list files in one API call, then fetching only matching ones. Still requires API calls and a hardcoded match list.

### Decision 3: Remove `ProjectContext` from `BuildPrompt` signature

`BuildPrompt` currently takes `*ProjectContext` and embeds convention file contents. After this change, it takes only `*PRInfo` — the context section is replaced by clone instructions. The prompt structure becomes: review instructions + clone/search instructions + output template + PR URL.

### Decision 4: Remove local checkout detection entirely

The `isLocalCheckout` optimization is no longer needed. Whether the user is in the target repo's directory or not, Claude will clone to `/tmp` and work from there. This simplifies the code and ensures consistent behavior regardless of cwd.

## Risks / Trade-offs

- **Clone takes time for large repos** → Mitigated by reuse (only first review is slow) and Claude can use `--single-branch --depth 1` for the initial clone if the prompt suggests it
- **Claude might not find all convention files** → Acceptable trade-off; Claude is good at searching and can find MORE files than the hardcoded list. The prompt will list common files as hints.
- **Temp directory cleanup** → Not our concern; `/tmp` is cleaned by the OS. Reuse across reviews is a feature.
- **No internet / clone fails** → Claude will see the git error and can report it clearly — better than the current silent failure
