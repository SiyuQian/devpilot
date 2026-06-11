# bloated-agents-md (eval fixture)

A deliberately broken harness fixture for the `devpilot-harness-engineering`
eval suite. The defect: `AGENTS.md` is ~190 lines of nested `if X then Y`
conditional rules — content that is injected into every prompt and that should
live in on-demand skills with progressive disclosure.

**Correct advice when pointed at this repo:** flag that AGENTS.md is injected on
every turn, that it has blown past the ~60/100-line bar, and that the conditional
detail belongs in skills the agent loads on demand — not a flat dump.

Not a real project; `internal/` and `web/` are empty stubs so the repo reads as
plausible.
