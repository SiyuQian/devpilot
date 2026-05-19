// Package parser defines the common interface for language-specific
// tree-sitter parsers that produce nodes and edges for the code graph.
package parser

import "github.com/siyuqian/devpilot/internal/graph/store"

// Parser parses source files of a single language into ParseResult.
// Implementations are stateless and safe for concurrent use.
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
