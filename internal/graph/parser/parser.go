// Package parser defines the common interface for language-specific
// tree-sitter parsers that produce nodes and edges for the code graph.
package parser

import "github.com/siyuqian/devpilot/internal/graph/store"

// Parser parses source files of a single language into ParseResult.
//
// Implementations must not hold mutable state across Parse calls so the
// same Parser value can be invoked from multiple goroutines. In particular,
// implementations that wrap a stateful native parser (e.g. tree-sitter)
// should allocate a fresh instance per Parse rather than caching a shared
// one, or guard the shared instance with explicit synchronization.
type Parser interface {
	Language() string
	Extensions() []string
	Parse(path string, src []byte) (ParseResult, error)
}

// ParseResult is the output of parsing a single source file.
type ParseResult struct {
	Nodes            []store.Node
	Edges            []store.Edge
	Errors           []ParseError
	InterfaceMethods map[string][]string // interfaceNodeID -> method names declared inside
}

// ParseError describes a recoverable parse failure.
type ParseError struct {
	Path    string
	Line    int
	Message string
}
