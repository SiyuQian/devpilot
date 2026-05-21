package parser

import (
	"testing"
)

// Compile-time assertion that GoNativeParser implements Parser.
var _ Parser = (*GoNativeParser)(nil)

func TestGoNativeParserSkeleton(t *testing.T) {
	parser := NewGoNativeParser()
	if parser == nil {
		t.Fatal("NewGoNativeParser() returned nil")
	}

	// Test Language() returns "go"
	if got := parser.Language(); got != "go" {
		t.Errorf("Language() = %q, want %q", got, "go")
	}

	// Test Extensions() returns [".go"]
	exts := parser.Extensions()
	if len(exts) != 1 || exts[0] != ".go" {
		t.Errorf("Extensions() = %v, want [.go]", exts)
	}

	// Test Parse returns empty ParseResult with no error
	src := []byte("package main\n\nfunc main() {}")
	result, err := parser.Parse("foo.go", src)
	if err != nil {
		t.Errorf("Parse() returned error: %v", err)
	}

	// Verify empty ParseResult
	if len(result.Nodes) != 0 {
		t.Errorf("Parse() Nodes = %v, want empty", result.Nodes)
	}
	if len(result.Edges) != 0 {
		t.Errorf("Parse() Edges = %v, want empty", result.Edges)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Parse() Errors = %v, want empty", result.Errors)
	}
	if len(result.InterfaceMethods) != 0 {
		t.Errorf("Parse() InterfaceMethods = %v, want empty", result.InterfaceMethods)
	}
}
