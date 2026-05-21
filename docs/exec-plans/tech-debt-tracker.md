# Tech debt tracker

Parked items not scheduled. Promote to an active exec-plan (`docs/exec-plans/active/`) when a real failure or deadline justifies the work. Until then: one-line entry, dated.

## Open

- 2026-05-21 — TS/JS parser pinned to `smacker/go-tree-sitter` (CGO, upstream in maintenance mode), which forces the release pipeline to do a per-platform CGO matrix build. Migrate to official `tree-sitter/go-tree-sitter` v0.25+ (mechanical rename) when convenient; re-evaluate going CGO-free once `malivvan/tree-sitter` (wazero+WASM) tags v1 or `microsoft/typescript-go` exposes a stable AST. Not scheduled — matrix release unblocks shipping today.

## Closed

- _empty_

---

Entry format:

```
- YYYY-MM-DD — <one-line problem> — <why it hasn't been scheduled>
```
