// Nested module so the parent repo's `go build ./...` / `make test` skips this
// eval fixture. The fixture exists to be inspected, not built.
module example.com/bloated-agents-md

go 1.22
