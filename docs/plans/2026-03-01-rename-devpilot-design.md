# Rename: devpilot → DevPilot

**Date:** 2026-03-01
**Status:** Approved

## Goal

Rename the project from "devpilot" / "DevPilot" / "devpilot" to "DevPilot" / "devpilot" across the entire codebase, including Go module path, CLI binary, config paths, skill namespaces, documentation, and CI/CD.

## Decisions

- **Full rename**: GitHub repo will also be renamed (siyuqian/devpilot → siyuqian/devpilot)
- **No backward compatibility**: No migration logic for old config paths or file names
- **All docs updated**: Including historical design/plan documents in docs/
- **Approach**: Single-pass bulk text replacement + directory/file renames

## Naming Mapping

Replacements applied in this order (longest match first to avoid false positives):

| Old | New | Context |
|-----|-----|---------|
| `github.com/siyuqian/devpilot` | `github.com/siyuqian/devpilot` | Go module, imports |
| `siyuqian/devpilot` | `siyuqian/devpilot` | GitHub repo refs |
| `DevPilot` | `DevPilot` | Brand title |
| `devpilot` | `devpilot` | Remaining repo refs |
| `devpilot:` | `devpilot:` | Skill namespace |
| `DevPilot` | `DevPilot` | Brand name in docs |
| `.devpilot.json` | `.devpilot.json` | Project config file |
| `.devpilot/` | `.devpilot/` | Runtime log directory |
| `devpilot` | `devpilot` | CLI name, binary, config paths, all remaining |
| `Devpilot` | `Devpilot` | Title case variants |
| `DevPilot` | `DevPilot` | CamelCase variants |

## File/Directory Renames

| Old Path | New Path |
|----------|----------|
| `cmd/devpilot/` | `cmd/devpilot/` |
| `.devpilot.json` | `.devpilot.json` |

## Scope

- **Go source**: ~13 files (imports + string constants)
- **Config**: 4 files (go.mod, Makefile, .gitignore, .devpilot.json)
- **Docs**: ~22 files (README, CLAUDE.md, docs/plans/*, docs/rejected/*)
- **Skills**: 4 SKILL.md files
- **Scripts/CI**: 3 files (install.sh, release.yml)
- **Tests**: ~3 files

## Verification

1. `go build ./cmd/devpilot` — compiles
2. `go test ./...` — all tests pass
3. `grep -ri "devpilot"` — zero residual matches in tracked files

## Out of Scope

- No migration/compat logic
- No git history rewrite
- GitHub repo rename is a manual step after code changes land
