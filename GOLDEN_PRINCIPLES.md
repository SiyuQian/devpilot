# Golden Principles

These are the opinionated taste calls that distinguish the DevPilot codebase from a generic Go CLI. They are not style rules (those live in `golangci.yml`) — they are the structural and design decisions that tend to regress without explicit reinforcement.

**How this file is used:**
- Humans: consult before reviewing an agent PR that touches architecture, public APIs, or shared utilities.
- Agents: the `devpilot-harness-engineering` skill's GC loop scans the repo against this list and opens small refactor PRs for deviations.
- Maintainers: when a code-review comment repeats itself across 3+ PRs, promote it here (or down into a linter rule if mechanical).

Grade each principle A–F by rough ratio of compliant vs deviant call sites. A regressing grade is a signal to strengthen the guide/sensor for that principle, not to lower the bar.

---

## Structure

### 1. Domain packages own their Cobra commands
Each `internal/<domain>/` package registers its own Cobra commands in a `commands.go` inside that package. There is no central `cli/` routing layer.

✅ `internal/trello/commands.go` defines `trelloCmd` and is wired from `cmd/devpilot/main.go`.
❌ A new `internal/cli/` package that imports every domain and builds the command tree in one place.

**Why:** keeps domains self-contained; new services can be added without touching a central switchboard.

### 2. Service clients live with their domain
External service clients (Trello API, GitHub API, HTTP, Anthropic) live in the same package as the domain logic that uses them.

✅ `internal/trello/client.go` + `internal/trello/runner.go` in the same package.
❌ A shared `internal/clients/` package holding all third-party wrappers.

**Why:** a domain should read top-to-bottom in one place.

### 3. Shared *project-level* config lives in `internal/project/`
Only cross-cutting config (paths, feature flags, `.devpilot.yaml` shape) belongs in `internal/project/`. Domain-specific config belongs in the domain package.

### 4. CLI entry is thin
`cmd/devpilot/main.go` wires root-level commands from domain packages and exits. Business logic does not live here.

---

## Public API Shape

### 5. Functional options for any constructor with more than one optional parameter
New clients, executors, runners expose `NewXxx(required, ...Option)` with `WithYyy(v) Option` helpers.

✅ `trello.NewClient(apiKey, token, trello.WithHTTPClient(c), trello.WithBaseURL(u))`
❌ `trello.NewClient(apiKey, token, httpClient, baseURL, retries, verbose)`

**Why:** testability, forward-compatibility, and our existing `Executor` + `trello.Client` already use this pattern — every new constructor joining them keeps the codebase consistent.

### 6. No positional `bool` parameters in exported functions
Replace with an option, a named struct field, or a typed enum.

---

## Errors & Logging

### 7. Wrap errors with `%w` and context at every layer boundary
```go
return fmt.Errorf("fetching card %s: %w", id, err)
```
Not bare `return err` at package boundaries; not `errors.New(err.Error())`.

### 8. One logging facade per binary
Don't mix `log`, `slog`, `fmt.Fprintln(os.Stderr, …)`, and a vendored logger in the same code path.

---

## Tests

### 9. Table-driven tests with named subtests
```go
for _, tc := range []struct{ name string; ... }{…} {
    t.Run(tc.name, func(t *testing.T) { … })
}
```

### 10. No mocks for our own packages
Use the real implementation in tests. Mocks belong at third-party boundaries (HTTP servers via `httptest`, external CLIs via fakes). OpenAI's "no mocks of your own code" lesson applies here too.

### 11. `testdata/` for fixtures, not inline giant strings
If a test needs more than ~20 lines of fixture, move it under `testdata/`.

---

## Skills

### 12. Every skill under `skills/` is registered in `skills/index.json`
`name`, `description`, and exact `files` list must match the directory. The installer depends on this; CI should catch drift (sensor gap — currently manual).

### 13. SKILL.md descriptions describe *when to use*, not *what the skill does*
Start with "Use when…". Do not summarize the workflow — that creates a shortcut the model will follow instead of reading the body.

### 14. Heavy reference content goes in `references/`, not inline
SKILL.md stays scannable. Detail, API docs, long examples live in `references/*.md` loaded on demand.

---

## Documentation

### 15. Design docs come in pairs
`docs/plans/{YYYY-MM-DD}-{feature}-design.md` and `{YYYY-MM-DD}-{feature}-plan.md`. Design = *what and why*. Plan = *implementation steps*.

### 16. Rejected ideas are recorded, not deleted
`docs/rejected/` holds one-pagers for ideas considered and deferred. The PM skill reads this to avoid re-recommending them.

---

## Runner & Event System

### 17. Runner events are additive, not mutated
New event types are new structs implementing the `Event` interface; existing events are never repurposed with a new meaning.

### 18. The TUI is a consumer, never a producer
The Bubble Tea model reads from the event channel and never writes back. Keyboard input produces `tea.Cmd`s local to the TUI, not runner events.

### 19. Per-card logs to `~/.config/devpilot/logs/{card-id}.log`
Don't invent parallel log locations. One path, one format.

---

## Anti-Principles (Do Not Do)

- **Don't** create a shared `internal/utils/` or `internal/common/` package. If it's truly shared, it has a real name; if it's not, it stays in its home package.
- **Don't** auto-generate `CLAUDE.md`, `AGENTS.md`, `GOLDEN_PRINCIPLES.md`, or `ARCHITECTURE.md`, `PLANS.md`. They're hand-written context; drift is worse than staleness.
- **Don't** install MCP tools or skills speculatively into `.claude/` without an observed failure they address.
- **Don't** add a bool flag to a function when an option or a new function would do.

---

## Promotion / Demotion Rules

- A principle here for 3+ months that can be expressed as a `go vet` / `golangci-lint` / custom analyzer rule **should be demoted to a linter** (sensor shifts left).
- A review comment repeated in 3+ PRs **should be promoted** into this file with a before/after example.
- If a principle's grade has been dropping for two consecutive audits, the *guide* (this entry, CLAUDE.md, or a skill) is insufficient — rewrite it, not just re-enforce it.
