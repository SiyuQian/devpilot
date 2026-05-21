# Graph: Native AST Backends — Design

**Status:** proposed
**Author:** Siyu Qian (with Claude)
**Date:** 2026-05-20
**Related:** [Phase 1–7 plan](./2026-05-19-devpilot-graph-plan.md), Phase 5 (LSP cross-check)

## Problem

The graph subsystem currently parses Go / TS / JS / Rust with **tree-sitter**. That gave us 4-language coverage on a single parsing interface, but:

- **No type information.** Tree-sitter is a syntax parser — it has no notion of method sets, interface satisfaction, generic instantiations, or cross-package symbol resolution.
- The current Go resolver compensates with a **name-only heuristic** (`internal/graph/resolver/resolver.go`): match an unresolved `external::Foo` edge to *any* batch node with that simple name. First-occurrence wins; duplicate names across files silently lose. `implements` edges fall back to **exact method-set name matches**, which both over- and under-fires.
- The TS resolver does the same for `tsconfig` path aliases — string rewriting, not type-aware.
- Phase 5 was scoped to **measure** this gap (precision/recall vs LSP, ≥ 90% gate) but not to **close** it. If a language sits below the gate we have no remediation other than "improve the tree-sitter queries."

The user asked for the higher-precision option: replace tree-sitter with the language's native type-aware AST on a per-language basis. This document lays out the plan.

## Goals

1. Lift Go and TypeScript precision/recall to LSP-equivalent levels on `callers_of`, `tests_for`, `implements`, and `imports`.
2. Eliminate the `external::Name` placeholder + name-heuristic resolver for Go and TS — emit real cross-package node IDs at parse time.
3. Keep the `parser.Parser` interface unchanged so `store`, `cache.Builder`, `query`, `envelope`, and the CLI surface stay untouched.
4. Keep tree-sitter as the JS / Rust backend for v1; this is a per-language migration, not a rewrite.

**Non-goals:**

- Replacing Rust. `syn` is syntax-only; matching LSP precision needs `rust-analyzer` inproc, which is a separate, larger effort.
- Bringing JS up — JS without type info isn't fundamentally better in `tsserver` than in tree-sitter. JS stays on tree-sitter; TS users get the upgrade.
- Live LSP integration in the runtime — Phase 5 still uses LSP only for cross-check.
- **Generic instantiation as distinct nodes.** `Foo[T]` and `Foo[int]` collapse to a single node — `types.Implements` handles instantiations under the hood, but our node IDs are uninstantiated. Acceptable for v1; revisit only if a query consumer asks.

## Scope: which Go repos N1 supports

`packages.Load` is surprisingly permissive but the design must pin down what we promise:

- **Single-module repo with `go.mod` at the repo root** — fully supported, primary target.
- **`go.work` repo / multi-module monorepo** — N1 loads each module separately and merges. We do this by reading `go.work` (if present) and calling `LoadModule` once per `use` directive; otherwise we fall back to one `LoadModule` at the repo root.
- **Vendored deps (`vendor/`)** — `packages.Load` already honors `-mod=vendor` when `vendor/` exists. We add no extra logic.
- **cgo-generated files (`_cgo_gotypes.go`) and other generated files with synthetic positions** — dropped at node-mint time. We test `types.Object.Pos().Filename`; if the path is not inside the repo root (or is the empty string), we skip the node and any edges that would target it.
- **`Tests: true` package duplication** — when go/packages synthesizes a `pkg [pkg.test]` variant, the same `.go` file appears in both packages. **Ownership rule:** prefer the non-test package; nodes get the non-test package's IDs. External test files (package `pkg_test`) own their own nodes, IDs prefixed with their own package path. `TestXxx` lives at `<pkg>/<file>::TestXxx`.
- **Non-module repos / `GOPATH` mode** — explicitly not supported. We emit a `ParseError` and fall back to the tree-sitter Go backend for the whole run.

## Phased rollout

| Phase | Scope | Backend | Cost |
|---|---|---|---|
| **N1** | Go parser | `golang.org/x/tools/go/packages` + `go/types` | ~1 week |
| **N2** | TypeScript parser | `tsserver` IPC (stdin/stdout JSON) or `ts-morph` via Node sidecar | ~2 weeks |
| **N3** *(optional)* | Rust parser | `rust-analyzer` IPC | ~3+ weeks, deferred |

JS and (until N3) Rust remain on tree-sitter. The four parsers continue to share the `parser.Parser` interface.

## N1: Go via `go/packages`

### Backend choice

`golang.org/x/tools/go/packages` with an explicit `Need*` mode set —
`NeedName | NeedFiles | NeedImports | NeedDeps | NeedTypes | NeedSyntax | NeedTypesInfo`.
(The `LoadAllSyntax` alias is deprecated in current `x/tools`.) This gives us:

- Full `*ast.File` + `*types.Package` for every package in the module.
- `types.Info.Uses` / `.Defs` — every identifier reference resolved to its `types.Object`.
- `types.Object.Pkg()` and `.Pos()` — exact origin, so we know whether a call is intra-package, intra-module, or genuinely external.
- `types.Implements(T, I)` — exact interface satisfaction, including embedded interfaces and generics.

We already depend on Go via go.mod; no new system requirements.

### What the new parser produces

The `parser.Parser` interface stays the same, but the unit of work changes:

- **Today:** `Parse(path, src) (ParseResult, error)` — file-at-a-time, no cross-file knowledge.
- **N1:** add an optional `PackageLoader` interface that the Go parser implements; `cache.Builder` detects it and calls `LoadModule(repoRoot) (map[path]ParseResult, error)` instead of per-file `Parse`. Other parsers keep working file-at-a-time.

```go
// New, additive interface — additive so tree-sitter parsers don't have to change.
type PackageLoader interface {
    LoadModule(repoRoot string) (map[string]ParseResult, error)
}
```

`cache.Builder.FullBuild` gains a small branch: if `reg.ForLanguage("go")` implements `PackageLoader`, build all Go results in one call; otherwise fall through to the existing per-file fanout. Determinism preserved by sorting the returned map keys before insertion.

### Edges with real targets

The Go resolver shrinks to almost nothing:

- **`calls` edges** — `Uses` maps each `*ast.CallExpr.Fun` ident to a `types.Object`. If `obj.Pkg()` is in-module, dst = the node ID we minted for that object. If it's `nil` (builtin) or out-of-module, drop the edge entirely (no more `external::*` noise).
- **`implements` edges** — for every concrete type T in the module, for every interface I in the module, emit `T --implements--> I` iff `types.Implements(T, I)` is true. No method-name matching.
- **`tests` edges** — same `Uses` traversal restricted to function bodies of `TestXxx(*testing.T)`. No naming heuristic.
- **`imports` edges** — `*ast.File.Imports` already resolved by go/packages to package paths.

`internal/graph/resolver/resolver.go` and `internal/graph/resolver/implements.go` lose their Go code paths entirely (TS path-alias rewriting in `resolver/tsconfig.go` stays until N2).

### Node ID stability

Today: `path::Name` (file-relative) for top-level, `path::Container.Method` for methods. We keep the shape. N1 derives the path from `fset.Position(obj.Pos()).Filename`, made repo-relative; combined with the file-ownership rules above, this keeps existing `CallersOf` queries valid across the migration.

**Cache invalidation is NOT automatic.** Today's `cache.parserVersionTag` (`builder.go:188`) is keyed on the language *list* (`"phase2:go,javascript,rust,typescript"`), not the backend. Flipping the env flag without a code change would leave stale caches in place. **N1 must extend `parserVersionTag` to include the Go backend name** (e.g. `"phase2:go=native,javascript,rust,typescript"`) so the existing schema-mismatch path in `Builder.Build` wipes and rebuilds. A SQLite schema bump is still not needed — only the tag string changes.

### Tradeoffs

- **Build time goes up across the board.** Loading a full module type-checks the world. Targets: cold cache ≤ 5 s on a 5k-LOC repo (devpilot itself); ≤ 30 s on a 100k-LOC repo; peak RSS ≤ 500 MB. `go/packages` honors `GOCACHE` so warm builds are much faster, but CI cold builds are the budget.
- **Go incrementality collapses to whole-module.** Today's incremental path re-parses only the changed files. With native, **any change to a `*.go` file, `go.mod`, or `go.sum` re-type-checks the whole module** — there is no honest file-level incremental with `go/types`. We accept this for N1. Mitigation deferred to N1.5 (a separate plan): cache `*packages.Package` between builds keyed on file mtimes + `go.mod` hash; reuse when no Go files changed since the last build. Non-Go languages keep their existing per-file incremental path unchanged.
- **Memory.** Whole-module load holds `*types.Package` graphs in memory. For repos in the 100k LOC range this can be 100s of MB. Acceptable; document it.
- **Parse-error semantics change.** `packages.Load` returns `err == nil` even when individual packages have errors — errors live in `pkg.Errors`, and a broken `pkg/a` poisons every dependent's type info (downstream `Uses[ident]` becomes `nil`). We must walk `pkg.Errors` per package, emit a `ParseError` for each affected package, and decide per-edge whether the callee's `obj.Pkg()` is usable. Concretely:
  - Drop the edge silently when `obj == nil` or `obj.Pkg() == nil` (builtin or unresolvable).
  - When *every* package in the module has errors AND no usable type info, hard-fail the build with an actionable error message — don't emit a half-empty graph.
  - When *some* packages succeed, emit their nodes/edges and surface the per-package errors as `ParseError` entries in the merged `ParseResult`.

## N2: TypeScript via `tsserver`

(Sketched; details in the N2 phase plan once N1 lands.)

- Spawn `tsserver` once per build; speak its JSON protocol over stdin/stdout.
- Use `references`, `implementations`, `definition` requests for the four edge kinds.
- Cache the `tsserver` process across files in the same build; tear down at the end.
- Fall back to tree-sitter when `tsserver` is not installed (warn, don't fail) — keeps the binary self-sufficient on lean environments.

## Migration mechanics

1. **N1 lands behind a flag** for one release: `DEVPILOT_GRAPH_GO_BACKEND={treesitter|native}`, default `treesitter`. The registry switches backend based on this env var; `parserVersionTag` includes the chosen backend so caches invalidate cleanly across the flip.
2. **Phase 5 is reframed for Go.** Comparing a `go/types`-based parser against `gopls` is near-tautological — both consume the same type info, so precision/recall will trivially hit 100% on the things both compute and 0% on things only one knows (build tags excluded code, generated files, etc.). For Go, Phase 5 becomes a **coverage check**: assert the native graph contains every symbol `gopls` exposes via `workspace/symbol`, and log (don't gate on) deltas. Phase 5 stays a precision/recall gate for TS / Rust where the LSP is genuinely independent.
3. **Tree-sitter Go code is deleted** in the release after the flag flips to `native` by default — see Task N1.17 in the phase plan. No long-term dual-stack.

## Open questions

- Do we want to expose package-level nodes (today: file-level)? Native AST makes this cheap; consumers of `query.Hubs` might prefer the coarser grouping.
- For huge repos, do we want to support `NeedName | NeedFiles | NeedImports` only (cheaper, no type info, fallback per package) when the user explicitly asks for speed over precision? Defer.

## Next step

Write `2026-05-20-graph-native-ast-phase-n1-plan.md` at half-day granularity for the Go migration, mirroring the Phase 2 / 3 / 4 expansion format. Implementation does not start until that plan is reviewed.
