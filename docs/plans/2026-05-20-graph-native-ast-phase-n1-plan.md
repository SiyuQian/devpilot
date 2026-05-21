# Graph: Native AST — Phase N1 (Go) Bite-Sized Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Checkbox (`- [ ]`) syntax tracks task state.

**Goal:** Replace the tree-sitter Go parser with a `go/packages` + `go/types` backend, behind a `DEVPILOT_GRAPH_GO_BACKEND` env flag. Keep node IDs, the `parser.Parser` interface, and the CLI surface stable. Delete the tree-sitter Go path once the flag flips by default.

**Spec:** [`2026-05-20-graph-native-ast-design.md`](./2026-05-20-graph-native-ast-design.md)

**Tech stack:**
- `golang.org/x/tools/go/packages` (NEW dep). Use the explicit `Need*` mode set: `NeedName | NeedFiles | NeedImports | NeedDeps | NeedTypes | NeedSyntax | NeedTypesInfo`. (`LoadAllSyntax` is deprecated.)
- Stdlib `go/ast` / `go/types` / `go/token`
- Existing `internal/graph/store`, `internal/graph/parser` interface

**Conventions:** one passing test → one logical change. `make lint && make test` before each milestone. Keep determinism: sort any map iteration before producing nodes/edges.

---

## File map

```
internal/graph/parser/
├── go_native.go              (new — *GoNativeParser implementing Parser + PackageLoader)
├── go_native_test.go         (new — fixture-driven; uses real go module under testdata/go_native/)
├── parser.go                 (modify — add PackageLoader interface)
├── testdata/go_native/       (new — multi-file/multi-package fixture module)
└── go.go                     (unchanged for N1; deleted in the cleanup task at the end)

internal/graph/cache/builder.go   (modify — branch on PackageLoader before the per-file fanout)
internal/graph/cache/incremental.go (modify — when go.mod/go.sum/any *.go changes, re-run LoadModule for go)

internal/graph/parser/registry.go (modify — register GoNativeParser when DEVPILOT_GRAPH_GO_BACKEND=native)
```

---

## Task N1.1: Add dep + PackageLoader interface

- [ ] `go get golang.org/x/tools/go/packages && go mod tidy`
- [ ] **Test first** (`parser/parser_test.go`): assert a struct implementing `PackageLoader` is assignable to it; this is a contract test for the interface only.
- [ ] **Implement** in `parser/parser.go`:
  ```go
  type PackageLoader interface {
      LoadModule(repoRoot string) (map[string]ParseResult, error)
  }
  ```
  Additive — no existing parser changes. PASS, commit.

## Task N1.2: GoNativeParser skeleton — Language/Extensions/Parse stub

- [ ] **Test** (`parser/go_native_test.go`): `NewGoNativeParser()` returns a Parser with `Language()=="go"`, `Extensions()==[".go"}`. Calling `Parse(path, src)` returns an empty `ParseResult` (Parse is unused — we go through LoadModule).
- [ ] **Implement** stub. PASS, commit.

## Task N1.3: Fixture — multi-file, multi-package module

- [ ] Create `internal/graph/parser/testdata/go_native/`:
  - `go.mod` (module `example.com/native`)
  - `pkg/a/a.go` — `func Greet(name string) string` + `func main()` calls `Greet`
  - `pkg/a/a_test.go` — `func TestGreet(t *testing.T) { Greet("x") }`
  - `pkg/b/b.go` — `import "example.com/native/pkg/a"` + `func B() { a.Greet("y") }`
  - `pkg/iface/iface.go` — `type Speaker interface { Speak() string }`
  - `pkg/impl/impl.go` — `type Console struct{}` + `func (Console) Speak() string`
- [ ] No test yet — fixture is consumed by N1.4+.
- [ ] **CI note**: fixture exercises `go/packages.Load`, which needs a working Go toolchain at test time. Confirm `.github/workflows/test.yml` already provides `go` on the test runner (it does today). Add a comment to the fixture's `go.mod` calling this out so a future stripped-down CI image doesn't silently break N1 tests.

## Task N1.4: LoadModule emits file + function/method nodes

- [ ] **Test** in `go_native_test.go`: `LoadModule(testdata path)` returns a `map[string]ParseResult` where:
  - keys are repo-relative file paths
  - every file has a `kind: file` node
  - `Greet`, `main`, `B`, `(Console).Speak`, `TestGreet` exist as nodes with correct kind/container/is_exported.
- [ ] **Implement** `LoadModule`:
  - `packages.Load(&packages.Config{Mode: needSet, Dir: repoRoot, Tests: true}, "./...")` where `needSet = NeedName|NeedFiles|NeedImports|NeedDeps|NeedTypes|NeedSyntax|NeedTypesInfo`.
  - For each package, for each `*ast.File`, iterate `Decls`. For `*ast.FuncDecl`:
    - skip files whose `fset.Position(f.Pos()).Filename` is not inside `repoRoot` (cgo / generated synthetic files — see design "Scope" subsection).
    - mint ID as `<relpath>::<name>` (function) or `<relpath>::<recvType>.<name>` (method).
    - emit `contains` edge from file → symbol.
  - **File-ownership rule for `Tests: true`:** when a file appears in both `pkg` and `pkg [pkg.test]`, prefer the non-test package. External test files (`pkg_test`) own their own nodes under their own package path. Implement by deduplicating on `(filename, declName)` with non-test-package winning.
  - **Partial-package errors:** walk `pkg.Errors` for every loaded package. For each error, append a `ParseError` to the merged `ParseResult`. Continue processing other packages. Only hard-fail (return error) when *all* loaded packages have errors AND zero usable type info.
  - Determinism: sort packages by `PkgPath`, files by relative path before walking.
- [ ] PASS, commit.

## Task N1.5: Type/interface/struct nodes

- [ ] **Test:** assert `Speaker` is a `kind:interface` node, `Console` is `kind:struct`, type aliases get `kind:type`.
- [ ] **Implement:** handle `*ast.GenDecl` with `tok=TYPE`. Use `types.Info.Defs[spec.Name].Type()` to discriminate interface / struct / other.
- [ ] PASS, commit.

## Task N1.6: Real `calls` edges via `types.Info.Uses`

- [ ] **Test:** assert edge `pkg/a/a.go::main` -- `calls` --> `pkg/a/a.go::Greet` (intra-package); edge `pkg/b/b.go::B` -- `calls` --> `pkg/a/a.go::Greet` (cross-package). No `external::*` placeholder edges anywhere in the result.
- [ ] **Implement:** for each `*ast.FuncDecl.Body`, walk `*ast.CallExpr`. Resolve callee via `pkg.TypesInfo.Uses[ident].(*types.Func)`; map back to a node ID through a `(*types.Object) → string` index built in N1.4. Skip the edge silently when `obj == nil` (callee unresolvable because a dependency had errors — see N1.4 partial-error handling), `obj.Pkg() == nil` (builtin), or `obj.Pkg()` is outside the module.
- [ ] PASS, commit.

## Task N1.7: Real `implements` edges via `types.Implements`

- [ ] **Test:** assert exactly one edge `pkg/impl/impl.go::Console` -- `implements` --> `pkg/iface/iface.go::Speaker`. Add a negative case: a struct that only implements *some* methods of `Speaker` must NOT produce the edge.
- [ ] **Implement:** after type-collection, for every concrete type T in the module, for every interface I in the module, call `types.Implements(T, I.Underlying().(*types.Interface))`. Iterate over a sorted slice of `(types.Object)` pairs for determinism.
- [ ] PASS, commit.

## Task N1.8: Real `tests` edges

- [ ] **Test:** edge `pkg/a/a_test.go::TestGreet` -- `tests` --> `pkg/a/a.go::Greet`.
- [ ] **Implement:** in test files (`pkg.GoFiles` ending in `_test.go` or with `_test` package), for `TestXxx(*testing.T)` funcs, replay the same Uses walk and emit `tests` edges (in addition to `calls`).
- [ ] PASS, commit.

## Task N1.9: `imports` edges between files

- [ ] **Test:** edge `pkg/b/b.go` -- `imports` --> `pkg/a/a.go` (or `pkg/a/` — pick file-of-package convention used elsewhere). No edges to stdlib (`fmt`, `testing`).
- [ ] **Implement:** walk `*ast.File.Imports`; resolve via `pkg.Imports[importPath]`; if in-module, emit edge to that package's primary file. Skip stdlib + third-party.
- [ ] PASS, commit.

## Task N1.10: Registry + env flag

- [ ] **Test** (`parser/registry_test.go`): with `DEVPILOT_GRAPH_GO_BACKEND=native`, `DefaultRegistry().ForPath("x.go")` returns `*GoNativeParser`; default / unset returns the tree-sitter `*GoParser`.
- [ ] **Implement:** branch in `registry.DefaultRegistry()`. Document the flag in `docs/cli-reference.md`.
- [ ] **Bump `parserVersionTag` in `cache/builder.go`** to include the resolved Go backend name, e.g. `"phase2:go=native,javascript,rust,typescript"` vs `"phase2:go=treesitter,..."`. This is the cache invalidation hook — without it, switching the env flag leaves stale caches in place. Add a test in `cache/builder_test.go` that asserts the tag differs between the two backends.
- [ ] PASS, commit.

## Task N1.11: cache.Builder PackageLoader branch

- [ ] **Test** (`cache/builder_test.go`): with the native backend selected and the testdata module, `FullBuild` produces a deterministic graph (two consecutive runs match snapshot); the graph has zero edges with `dst` matching `^external::`.
- [ ] **Implement:** in `Builder.FullBuild`, after `WalkRepo` + `FilterByParser`, if `b.reg.ForLanguage("go")` implements `PackageLoader`:
  - call `LoadModule(b.repo)` once.
  - merge those results with results from non-Go parsers (TS / JS / Rust still per-file).
  - skip Go files in the per-file fanout.
- [ ] **Resolver ordering preserved**: `resolver.Resolve(results)` still runs after merging, and the TS `tsconfig` rewrite at `builder.go:121-129` still runs after that. The Go branch must produce results that pass through `Resolve` unchanged (see N1.13). Add an assertion to the test that the TS path-alias rewrite still fires on a fixture mixing Go + TS.
- [ ] **`go.work` detection**: if `repoRoot/go.work` exists, parse the `use` directives and call `LoadModule` once per module, merging maps. Otherwise call `LoadModule(repoRoot)` once. Non-module repos: return a `ParseError`, skip the native path, fall through to the tree-sitter Go fanout (so we don't crash on `GOPATH`-mode or random directory inputs).
- [ ] PASS, commit.

## Task N1.12: Incremental update for native Go (whole-module re-typecheck)

> **Honest budget:** native Go has no file-level incremental path — `go/types` requires whole-module load. Any `*.go` / `go.mod` / `go.sum` change re-types the module. This is acknowledged in the design's Tradeoffs section.

- [ ] **Test:** mutate `pkg/a/a.go`, run `BuildIncremental`, assert the resulting graph matches a fresh `FullBuild` (snapshot equality on `dumpDB`). Also assert that an incremental build where *only* non-Go files changed does NOT re-run `LoadModule` (timer assertion: Go path takes < 50 ms in that scenario).
- [ ] **Implement:** in `incremental.go`, when ANY `*.go` change OR `go.mod`/`go.sum` change is in the change set, re-run `LoadModule` for the whole module. Drop & reinsert all Go-owned rows in one transaction; non-Go incremental path unchanged.
- [ ] PASS, commit.

> **Deferred (separate plan, not N1):** cache `*packages.Package` between builds keyed on file mtimes + `go.mod` hash so the re-typecheck cost is paid only when something actually changed. Tracked as a follow-up because it doubles the implementation cost and isn't needed to unblock the migration.

## Task N1.13: Resolver shrinks

- [ ] **Test:** `resolver.Resolve` against a Go-only batch where Nodes / Edges already use real cross-package IDs must be a no-op (input == output).
- [ ] **Implement:** make the resolver skip rewriting for edges whose dst doesn't start with `external::`. The Go path effectively becomes inert because the native parser stops emitting `external::*` for in-module symbols. (Don't delete the resolver yet — TS path-alias rewriting still uses it.)
- [ ] PASS, commit.

## Task N1.14: Cross-check against existing tree-sitter outputs (advisory, not gating)

> **Why advisory:** tree-sitter emits `external::Name` edges that the name-heuristic resolver rewrites — some correctly, some wrong. Native produces real cross-package IDs directly and never emits the placeholder. A strict superset check would flag native *improvements* (where tree-sitter mis-resolved A→B but native correctly resolved A→C) as missing edges and block legitimate wins.

- [ ] **Test** (`cache/native_parity_test.go`, log-only, never fails): run `FullBuild` on devpilot with both backends, dump nodes/edges to JSON, produce a delta report:
  - **Coverage:** for every `(src, kind)` pair in tree-sitter output, native has ≥ 1 edge with the same `src` and `kind`. Log misses with sample.
  - **Net new:** edges native added that tree-sitter didn't have. Log count + 10-row sample.
  - **Net removed:** edges tree-sitter had that native doesn't, grouped by reason (`external::*` dropped, heuristic mis-rewrite no longer fires, genuinely missing).
- [ ] **Acceptance:** human review of the delta report — not a green/red gate. Commit the report in `docs/plans/2026-05-20-graph-native-ast-parity-report.md` so the rollout call in N1.16 has data behind it.
- [ ] PASS (test always logs and passes), commit.

## Task N1.15: Performance baseline

- [ ] **Test:** add a `t.Skip` benchmark `BenchmarkNativeFullBuild` that runs `FullBuild` on devpilot once; record allocations + wall time. Budgets from the design:
  - devpilot (~5k LOC): cold wall ≤ 5 s, peak RSS ≤ 500 MB.
  - 100k-LOC reference repo (e.g. `go-redis`): cold wall ≤ 30 s, peak RSS ≤ 500 MB.
- [ ] If we miss the budget, profile via `-cpuprofile` / `-memprofile` and tune. Avoid dropping `Need*` flags blindly — every removal silently disables a downstream feature (e.g. dropping `NeedTypesInfo` kills `Uses`).
- [ ] Commit profile results in `docs/plans/2026-05-20-graph-native-ast-bench.md`.

## Task N1.16: Documentation + release-toggle prep

- [ ] Update CLAUDE.md / README to mention `DEVPILOT_GRAPH_GO_BACKEND=native` as the recommended setting after one stable release.
- [ ] Update the Phase 5 design entry: for Go, Phase 5 becomes a **coverage check** (assert every `gopls` `workspace/symbol` entry appears in the native graph; log deltas, no gate). Precision/recall gating stays for TS / Rust where the LSP is genuinely independent.
- [ ] Commit `docs: native-AST Go backend, env-flag opt-in`.

## Task N1.17 (deferred — after the env flag flips by default)

- [ ] Delete `internal/graph/parser/go.go` + tree-sitter Go deps (`github.com/smacker/go-tree-sitter/golang` removal from go.mod via tidy).
- [ ] Delete tree-sitter-specific resolver paths that only existed for Go.
- [ ] Remove the env flag and the registry branch — native is the only Go path.
- [ ] One commit: `feat(graph): remove tree-sitter Go backend after native AST migration`.

---

## Phase N1 acceptance

- [ ] All 16 in-scope tasks have green tests.
- [ ] Native backend emits zero `external::Name` edges for in-module symbols on the testdata fixture AND on devpilot.
- [ ] Parity report (N1.14) committed and human-reviewed; net-new and net-removed counts are explainable.
- [ ] `make lint && make test` clean with `DEVPILOT_GRAPH_GO_BACKEND` unset and set to `native`.
- [ ] Bench (N1.15) within budgets.
- [ ] Phase 5 entry reframed for Go (coverage check, not precision/recall gate).

## Non-acceptance (deferred)

- Rust and TS migrations are out of scope for N1. Their tests + tree-sitter paths must remain unchanged.
- Deletion of `go.go` (Task N1.17) is intentionally a separate release.
