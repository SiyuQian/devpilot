# rule-without-sensor (eval fixture)

A deliberately broken harness fixture for the `devpilot-harness-engineering`
eval suite. The defect: `CLAUDE.md` states a hard rule in prose ("never call
`time.Now()` directly in business logic"), but **nothing mechanical enforces
it** — there is no linter config, no forbidigo rule, no test, no pre-commit
hook. Two source files (`internal/order/order.go`, `internal/billing/invoice.go`)
violate the rule right now.

**Correct advice when pointed at this repo:** a guide with no sensor doesn't
hold — pair the rule with a mechanical check (a `forbidigo`/golangci-lint rule,
an import-restriction lint, or a test) whose output tells the *agent* how to
fix the violation, not just that one exists. Re-wording the doc is not the fix.

Not a real project; only enough Go to make the violation real.
