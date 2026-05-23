# Import / Dependency Reality Check (Agent F)

Agent F is the **mechanical** member of the fanout. It does no judgment — it checks that the artifacts the diff *names* (added imports, added dependencies, new package references) actually exist on their public registry. This is the cheapest, highest-signal class of finding in AI-authored code: hallucinated APIs / fake packages / "slopsquatting" survive every other agent's text-based review because they are syntactically perfect Go / TypeScript / Rust / Python.

The dispatcher pre-extracts the candidate identifiers from the diff and hands them to Agent F as a structured list. Agent F runs the per-ecosystem verification command and returns one finding per *unresolved* artifact.

## What the dispatcher pre-extracts (input to Agent F)

Before dispatching Agent F, the main session walks the diff and assembles:

```json
{
  "go": [
    {"manifest_line": "go.mod:12", "module": "github.com/anthropics/agent-utils-v2", "version": "v0.4.1"},
    {"import_line": "internal/scoring/score.go:4", "module": "github.com/anthropics/agent-utils-v2/ranking", "version": ""}
  ],
  "npm": [
    {"manifest_line": "package.json:18", "package": "react-llm-toolkit", "version": "^3.2.1"}
  ],
  "python": [
    {"manifest_line": "requirements.txt:7", "package": "anthropic-codetools", "version": "==0.9.0"}
  ],
  "rust": [
    {"manifest_line": "Cargo.toml:14", "crate": "claude-tool-bridge", "version": "0.2"}
  ]
}
```

**Extraction rules:**

- **Go** — every NEW line in `go.mod`'s `require` block; every NEW or MODIFIED `import "..."` line in `.go` files whose module path is not already present in the base-ref `go.mod`.
- **npm** — every NEW key under `dependencies` / `devDependencies` / `peerDependencies` in `package.json`; every NEW `import ... from "<pkg>"` / `require("<pkg>")` whose package is not already in base-ref `package.json` (scoped packages count).
- **Python** — every NEW line in `requirements*.txt` / `pyproject.toml [tool.poetry.dependencies]` / `setup.py install_requires`.
- **Rust** — every NEW key under `[dependencies]` / `[dev-dependencies]` in `Cargo.toml`.

Skip standard library imports (`fmt`, `os`, `sort` for Go; `react`, `node:fs` for npm; `os`, `sys`, `typing` for Python). The dispatcher's extractor maintains a small allowlist per language; everything else goes through Agent F.

If the diff touches no manifest and no new third-party imports, **skip Agent F dispatch entirely** — there's nothing to check. Record `agent_f: skipped (no new dependencies)` in the body sweep summary.

## Per-ecosystem verification commands

Each must complete in < 5 s per artifact; cache the result by `(ecosystem, name, version)` within the review session to avoid duplicate calls when the same package appears in multiple lines.

### Go

```bash
# Module-level existence (works without checkout):
go list -m -json <module>@<version>          # version="" → use "latest"
# Exit code 0 + JSON with Path field → exists.
# Exit code != 0 + stderr containing "module ... not found" / "unknown revision" → does not exist.
```

Fallback if `go list` is unavailable in the sandbox:

```bash
curl -fsSI "https://proxy.golang.org/<module>/@v/list"     # 404 → does not exist
```

For sub-packages (`github.com/foo/bar/sub`), trim the path to the module root by matching against `go.mod`'s `require` block — registries are module-rooted, not path-rooted.

### npm

```bash
npm view "<package>@<version>" name --json   # exits 0 + JSON → exists
# Exit code != 0 → does not exist; stderr "404 Not Found" is definitive.
```

Fallback:

```bash
curl -fsS "https://registry.npmjs.org/<package>"   # 404 → not on registry
```

For scoped packages (`@org/pkg`), URL-encode the `/` (`%2F`) only in the curl form.

### Python

```bash
curl -fsS "https://pypi.org/pypi/<package>/json" | jq -e '.info.name'
# 404 → not on PyPI; check version exists in .releases keys.
```

### Rust

```bash
curl -fsS "https://crates.io/api/v1/crates/<crate>"   # 404 → not on crates.io
```

## Finding shape

For each unresolved or suspicious artifact:

```yaml
- agent: F
  path: <manifest_line.path>
  line: <manifest_line.line>
  side: RIGHT
  title: "Dependency does not exist on registry — <name>@<version>"
  severity: Blocking
  confidence: 100
  behavior: "`<name>@<version>` is referenced by this PR (manifest at <path>:<line>, import at <import_line.path>:<line>). The package was queried against <registry> and returned 404 / not-found."
  why: "Hallucinated dependencies fail builds, expose the project to supply-chain compromise if a malicious actor later registers the typo'd name (slopsquatting), and indicate the change was written without execution. This must be resolved before merge."
  fix: "Verify the intended package name. Candidates: <typo_candidates if generated>. If the package legitimately exists on a private registry, document the private-registry config in the PR description and `not_applicable` this check explicitly."
```

**Severity rules:**

- Package not on registry at all → `Blocking, 100`.
- Package exists but exact version not published → `Should-fix, 95`.
- Package exists; name is within Levenshtein distance ≤ 2 of a top-1000 download package on the same registry (potential typosquat) → `Should-fix, 80`.
- Package exists, version exists, no typo neighbors → no finding; record `<name>@<version>: ok` in coverage block.

## Coverage block

Agent F returns alongside findings:

```yaml
coverage:
  dependencies:
    artifacts_checked: <int>
    artifacts_unresolved: <int>
    artifacts_typo_flagged: <int>
    skipped: false     # or true with reason
```

The dispatcher uses `artifacts_checked` to populate the body's "Dependency reality: N/N artifacts verified" line in the Unknown-Unknowns Sweep section.

## What Agent F does NOT do

- Does not judge whether a *real* dependency is a good choice. License, maintainership, popularity, security advisories — out of scope. License-and-CVE auditing is a separate skill we'd plug in later.
- Does not chase transitive dependencies. Only directly-added artifacts in this PR's diff.
- Does not run the package. Existence on registry, not behavior, is the contract.
- Does not interact with private registries. If a name resolves locally via a private mirror but 404s on the public registry, Agent F reports `unresolved` and the author must explicitly mark `not_applicable (private registry: <reason>)`.

## Failure modes & fallback

- Network unavailable in the review sandbox → Agent F returns `coverage.dependencies.skipped: true (reason: no network)`. The body sweep shows `Dependency reality: not verified (no network)`; review event is downgraded to at most `COMMENT` if any new third-party dependency was added.
- Registry rate-limit → wait 2 s, retry once; on second 429 record `skipped (rate-limited)`.
- Ambiguous response (registry returns 200 with redirect to a deprecated/squatted package) → emit a `Should-fix, 70` finding and let the author confirm.

Never block a review on Agent F infra failure — distinguish "network said no" from "registry said no" in the finding text.
