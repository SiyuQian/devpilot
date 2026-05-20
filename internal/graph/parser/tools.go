//go:build tools

// Package parser pins tree-sitter dependencies that will be imported in
// subsequent tasks. Without this, `go mod tidy` would strip them.
package parser

import (
	_ "github.com/smacker/go-tree-sitter"
	_ "github.com/smacker/go-tree-sitter/golang"
)
