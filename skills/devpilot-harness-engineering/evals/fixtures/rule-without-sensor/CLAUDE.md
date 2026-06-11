# Project conventions

This is a payments service. A few hard rules:

## Time

**Never call `time.Now()` directly in business logic.** Wall-clock reads inside
domain code make behavior untestable and cause flaky tests. Inject a `Clock`
interface and read the time through it, so tests can pin the clock. The only
place a raw `time.Now()` is allowed is in `cmd/` wiring, where the real clock is
constructed.

## Money

Represent money as integer minor units (cents) via the `Money` type in
`internal/money`. Never use `float64` for amounts.

## Errors

Wrap errors at package boundaries with `fmt.Errorf("...: %w", err)`.
