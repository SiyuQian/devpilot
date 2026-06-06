# AGENTS.md

This file is injected into every agent prompt. Read all of it before doing anything.

## General

- If you are adding a new HTTP handler, then put it in `internal/api/handlers/` and register it in `internal/api/router.go`,
  - if the handler needs auth then wrap it with `RequireAuth`
  - if it is public then add it to the `publicRoutes` allowlist
  - if it returns JSON then use `respondJSON`
  - if it returns a file then use `respondStream`.
- If you are writing a database query, then use the `sqlc`-generated code in `internal/db/gen/`,
  - if the query is new then add the `.sql` file under `internal/db/queries/` first and run `make sqlc`
  - if the query joins more than three tables then add an index and note it in `docs/db-indexes.md`
  - if it writes then wrap it in a transaction via `db.WithTx`.
- If you touch anything under `internal/billing/`, then ping the billing owner in the PR description,
  - if the change affects invoices then add a row to `docs/billing-changelog.md`
  - if it changes a price then it must be behind a feature flag
  - if it is a refund path then add an integration test.
- If you add a config value, then add it to `config/schema.go`,
  - if it is a secret then it goes in the vault not the repo
  - if it has a default then document the default in `config/README.md`
  - if it is environment-specific then add it to all three of `config/dev.yaml`, `config/staging.yaml`, `config/prod.yaml`.

## Naming

- If a function returns an error as its last value, then name it so the verb is first,
  - if it is a constructor then prefix with `New`
  - if it builds without I/O then prefix with `Make`
  - if it is a getter then do not prefix with `Get` unless it does I/O.
- If a type is an interface, then do not suffix it with `Interface` or `Iface`,
  - if it has one method then name it after the method plus `-er`
  - if it is a mock then put it in `mocks/` and suffix with `Mock`.
- If a variable is a context, then name it `ctx`,
  - if it is an error then name it `err`
  - if it is a temporary buffer then name it `buf`
  - if it is a loop index over a domain slice then name it after the singular noun, not `i`.
- If you introduce an acronym, then keep it all-caps in exported names (`HTTPServer`, not `HttpServer`),
  - if it starts an unexported name then keep it all-lowercase (`httpServer`).

## Errors

- If you return an error across a package boundary, then wrap it with `fmt.Errorf("...: %w", err)`,
  - if the message would duplicate the caller's context then drop the redundant words
  - if the error is sentinel-checked then export it as `ErrXxx`
  - if it is user-facing then map it through `apierr.From`.
- If a function can fail in more than three distinct ways, then return a typed error, not a string,
  - if the caller needs to branch on the failure then expose `errors.Is`/`errors.As` targets.
- If you log an error, then do not also return it logged at a lower layer (log once, at the top),
  - if it is a background goroutine then log with the request id if one exists.

## Testing

- If you add a function, then add a table-driven test,
  - if it has more than two branches then cover each branch with a named subtest
  - if it does I/O then add a fake, not a mock of our own package
  - if it is concurrent then run the test with `-race`.
- If you add an integration test, then put it behind the `integration` build tag,
  - if it needs a database then use the `testdb` helper
  - if it needs the network then skip it in CI unless `RUN_NET_TESTS=1`.
- If a test is flaky, then quarantine it with `t.Skip` and open an issue the same day,
  - if it stays skipped for two weeks then delete it.
- If you assert on an error, then assert on the wrapped target with `errors.Is`, not on the string,
  - if the string matters then assert a substring, not equality.

## HTTP

- If a handler reads a body, then cap it with `http.MaxBytesReader`,
  - if it parses JSON then reject unknown fields
  - if it is idempotent then support an `Idempotency-Key` header
  - if it is paginated then use cursor pagination, not offset.
- If you add a response header, then add it in middleware if it applies to all routes,
  - if it is a CORS header then change `internal/api/cors.go` only
  - if it is a cache header then make sure it matches the CDN config in `infra/cdn.tf`.
- If a request is slow, then add a timeout context derived from the request,
  - if it calls a third party then add a circuit breaker
  - if it retries then cap retries at three with jittered backoff.

## Frontend

- If you add a React component, then put it in `web/src/components/`,
  - if it is a page then put it in `web/src/pages/`
  - if it fetches data then use the `useQuery` wrapper, not raw `fetch`
  - if it mutates then use `useMutation` and invalidate the relevant query keys.
- If you add a style, then use the design tokens in `web/src/theme/`,
  - if a token does not exist then add it there first
  - if it is a one-off then still do not inline a hex value.
- If you add a form, then use `react-hook-form`,
  - if it has validation then use the shared `zod` schemas in `web/src/schemas/`
  - if it submits money then double-confirm with a modal.
- If a component re-renders too often, then memoize the expensive child,
  - if the prop is a function then wrap it in `useCallback`
  - if it is an object then wrap it in `useMemo`.

## Migrations

- If you change the schema, then add a migration in `migrations/` with both `up` and `down`,
  - if it is destructive then it ships in its own PR
  - if it backfills then the backfill runs as a separate job, not in the migration
  - if it adds a NOT NULL column then add it nullable first, backfill, then add the constraint.
- If a migration is large, then run it in batches,
  - if it locks a table then schedule it for the maintenance window
  - if it cannot be reversed then document the recovery procedure in the PR.

## Security

- If you handle user input, then validate it at the boundary,
  - if it reaches a query then it must be parameterized
  - if it reaches a shell then do not
  - if it reaches a template then auto-escape.
- If you add an endpoint that returns user data, then check the caller owns the data,
  - if it is admin-only then gate it behind the `admin` scope
  - if it is internal then require mTLS.
- If you add a dependency, then check its license is in the allowlist,
  - if it is a transitive bump then run `make audit`
  - if it has a known CVE then do not add it.
- If you store a password, then it is `argon2id`, never anything else,
  - if you store a token then store only its hash
  - if you log a request then redact `Authorization` and `Cookie`.

## Performance

- If a query is in a hot path, then it must use an index,
  - if it returns many rows then it must paginate
  - if it is read-heavy then consider the read replica
  - if it is cacheable then cache it with an explicit TTL.
- If you allocate in a loop, then hoist the allocation,
  - if you build a string then use a `strings.Builder`
  - if you marshal repeatedly then reuse the encoder.

## Observability

- If you add a code path that can fail, then add a metric for it,
  - if it is latency-sensitive then add a histogram
  - if it is a queue then export depth and age
  - if it is a feature flag then export which branch was taken.
- If you add a log line, then make it structured,
  - if it is in a request path then include the trace id
  - if it is an error then include enough context to reproduce without the log being PII.

## Releases

- If you cut a release, then update `CHANGELOG.md`,
  - if it is a breaking change then bump the major and write a migration note
  - if it touches the public API then update `docs/api/openapi.yaml`
  - if it changes the CLI then update `docs/cli.md`.
- If a release fails in staging, then roll back before debugging,
  - if the rollback fails then page the on-call
  - if it is a data issue then freeze writes first.

## Pull requests

- If your PR is larger than 400 lines, then split it,
  - if it cannot be split then explain why in the description
  - if it touches more than three domains then it almost certainly should be split.
- If your PR changes behavior, then it needs a test that fails before and passes after,
  - if it is a refactor then it needs no behavior change and the tests prove it
  - if it is a revert then link the original.
- If CI is red, then do not request review,
  - if it is red because of flake then link the quarantine issue
  - if it is red because of lint then just fix it.
