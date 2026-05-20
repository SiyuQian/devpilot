# Devpilot Graph Phase 4 — CLI Surface Bite-Sized Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose the Phase 3 `internal/graph/query` package as `devpilot graph <verb>` subcommands with a uniform JSON envelope. Every command emits one JSON document on stdout that validates against a versioned schema.

**Architecture:** Per project convention (CLAUDE.md: *"Cobra commands live with their domain in `commands.go` — no central `cli/` router"*), all graph subcommands are registered from `internal/graph/commands.go`. Each subcommand runner lives in `internal/graph/cli_<verb>.go`. `cmd/devpilot/main.go` calls `graph.RegisterCommands(rootCmd)` once.

Envelope type + JSON-schema validator live in `internal/graph/envelope`.

**Tech Stack:** Go 1.25, cobra, `github.com/santhosh-tekuri/jsonschema/v5` (new dep), stdlib `embed`, `os/exec` for e2e.

**Conventions:** One passing test → one commit. `make lint && make test` before each commit.

---

## File Structure

```
internal/graph/
├── commands.go                  (RegisterCommands wires all subcommands)
├── cli_helpers.go               (resolveRepo, openStore, emit)
├── cli_build.go / _test.go
├── cli_status.go / _test.go
├── cli_clean.go / _test.go
├── cli_query.go / _test.go
├── cli_impact.go / _test.go
├── cli_hubs.go / _test.go
├── cli_context.go / _test.go
├── cli_detect_changes.go / _test.go
├── cli_preflight.go / _test.go
├── e2e_test.go                  (//go:build e2e)
└── envelope/
    ├── envelope.go / _test.go
    ├── validate.go / _test.go
    └── schemas/
        ├── embed.go             (//go:embed *.json)
        ├── envelope.v1.json
        ├── build.v1.json
        ├── status.v1.json
        ├── clean.v1.json
        ├── query.v1.json
        ├── impact.v1.json
        ├── hubs.v1.json
        ├── context.v1.json
        ├── detect_changes.v1.json
        └── preflight.v1.json
```

---

## Envelope Shape

```jsonc
{
  "schema_version": "1",
  "command": "graph.preflight",
  "ok": true,
  "data": { /* payload */ },
  "error": null,
  "warnings": [],
  "next_tool_suggestions": ["devpilot graph context --id ..."],
  "elapsed_ms": 142
}
```

On error: `ok=false`, `data=null`, `error={"code","message"}`, exit code 1.

---

## Task 4.1: Envelope type + helpers

- [ ] Failing test in `envelope_test.go` covering `New`, `OK`, `Err`, `Suggest`, `Marshal`.
- [ ] Implement `envelope.go` with `Envelope` struct (json tags: `schema_version`, `command`, `ok`, `data`, `error`, `warnings`, `next_tool_suggestions`, `elapsed_ms`), `New(cmd)`, `OK(data)`, `Err(code,msg)`, `Warn(msg)`, `Suggest(cmds...)`, `Marshal()`.
- [ ] Run, expect PASS. Commit `feat(graph/envelope): canonical JSON envelope type`.

## Task 4.2: JSON schemas + validator

- [ ] `go get github.com/santhosh-tekuri/jsonschema/v5 && go mod tidy`
- [ ] Failing test in `validate_test.go`.
- [ ] Write `envelope.v1.json` (base) and `status.v1.json` (concrete example with `allOf $ref envelope.v1.json`).
- [ ] Implement `schemas/embed.go` with `//go:embed *.json var FS embed.FS`.
- [ ] Implement `validate.go` with `Validate(raw []byte, schemaID string) error` that lazily compiles all schemas from `schemas.FS`.
- [ ] PASS. Commit `feat(graph/envelope): jsonschema-backed validator`.

## Task 4.3: graph build + shared helpers + cobra wiring

- [ ] Failing test for `resolveRepo` (abs path, missing → error).
- [ ] Failing test for `runBuild(repo)` — temp DEVPILOT_HOME, fixture go file, assert envelope `ok=true`, command=`graph.build`.
- [ ] Implement `cli_helpers.go`: `resolveRepo`, `openStore`, `emit(e, schemaID) int` (marshals, validates, prints, returns exit code).
- [ ] Implement `cli_build.go` calling `cache.NewBuilder` + `b.Build()`, emit `{repo, mode, files_parsed, nodes, edges}`.
- [ ] Write `build.v1.json`.
- [ ] Implement `commands.go` with `RegisterCommands(parent)` adding `graph` parent + `buildCmd()`. Wire into `cmd/devpilot/main.go`.
- [ ] PASS. Commit `feat(graph/cli): graph build subcommand`.

## Task 4.4: graph status

- [ ] Failing test seeding cache via builder, asserts `nodes>0`, `languages` includes `go`, validates against `status.v1.json`.
- [ ] Implement `cli_status.go` reading `cache.ReadMeta` + `store.CountNodes/CountEdges` (add count helpers if missing).
- [ ] Register `statusCmd()` with `--repo`. PASS. Commit.

## Task 4.5: graph clean

- [ ] Three failing tests: `--repo X`, `--all`, neither (→ `args_required`).
- [ ] Implement `cli_clean.go` using `os.RemoveAll` on `cache.GraphDir(home, RepoKey(abs))` or `<home>/graphs`.
- [ ] Register `cleanCmd()` with `--repo`, `--all`. PASS. Commit.

## Task 4.6: graph query (6 v1 patterns)

Patterns: `callers_of`, `callees_of`, `tests_for`, `implementors_of`, `hubs`, `context`.

- [ ] Table-driven failing test seeding nodes/edges directly via `store.InsertNodes/Edges` for all six patterns.
- [ ] Implement `cli_query.go` with `runQuery(opts)` switching on pattern → `query.CallersOf/CalleesOf/TestsFor/ImplementorsOf/Hubs/Context`. Emit `data={pattern_result:{<key>:<value>}}`.
- [ ] Write `query.v1.json` with `oneOf` over six sub-shapes.
- [ ] Register `queryCmd()` with positional args `<pattern> [target]` and flags `--repo`, `--depth`, `--threshold`.
- [ ] PASS. Commit `feat(graph/cli): graph query with six v1 patterns`.

## Task 4.7: thin wrappers — impact / hubs / context

Per-command failing test → implement → schema → register → commit.

- [ ] **4.7a** `graph impact --files a,b,c --depth N` → `query.ImpactRadius` → `data={changed_symbols,callers}`. Commit.
- [ ] **4.7b** `graph hubs --threshold N` → `query.Hubs` → `data.hubs=[{id,caller_count}]`. Commit.
- [ ] **4.7c** `graph context --id ID --depth N` → `query.Context` → `data.context={target,callers}`. Commit.

## Task 4.8: detect-changes + preflight composites

### 4.8a graph detect-changes

- [ ] Failing test using a small temp git repo; assert `changed_symbols` non-empty, each `change_type ∈ {added,removed,modified,renamed}`.
- [ ] Implement `cli_detect_changes.go` calling `query.DetectChanges(st, abs, base, head)`. Suggest follow-up `graph preflight`.
- [ ] Write `detect_changes.v1.json`. Register. Commit.

### 4.8b graph preflight

- [ ] Failing test mirroring Phase 3 roundtrip: build graph for `internal/auth/`, run against two synthetic SHAs, assert top-level keys from design §6.
- [ ] Implement `cli_preflight.go` calling `query.Preflight(st, query.PreflightInput{...})`. Add `Suggest` for top 3 risky symbols → `graph context --id …`.
- [ ] Write `preflight.v1.json` mirroring `query.PreflightResult` struct tags.
- [ ] Register `preflightCmd()` with `--base`, `--head`, `--repo`. PASS. Commit `feat(graph/cli): graph preflight subcommand emits §6 payload`.

## Task 4.9: end-to-end test via compiled binary

- [ ] Create `internal/graph/e2e_test.go` (`//go:build e2e`).
- [ ] `TestE2EAllCommands` builds `cmd/devpilot` into `t.TempDir()`, sets `DEVPILOT_HOME`, runs each subcommand via `os/exec` against a fixture git repo, parses stdout, calls `envelope.Validate(stdout, "<verb>.v1.json")`, asserts `ok=true`.
- [ ] Run `go test -tags=e2e ./internal/graph/ -v`. PASS. Commit `test(graph/cli): e2e suite for all graph subcommands`.

---

## Phase 4 acceptance

- [ ] All nine subcommands + six `query` patterns have passing e2e tests.
- [ ] Every JSON output validates against its v1 schema.
- [ ] `devpilot graph preflight` against devpilot itself completes in < 10s on warm cache.
- [ ] `make lint` clean.
- [ ] `cmd/devpilot/main.go` adds exactly one new `RegisterCommands` call.
- [ ] No subcommand owns a file under `cmd/devpilot/`; all CLI code lives under `internal/graph/`.
