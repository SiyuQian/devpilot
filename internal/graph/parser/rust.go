package parser

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	rustLang "github.com/smacker/go-tree-sitter/rust"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// RustParser extracts nodes and edges from .rs source files.
type RustParser struct{ lang *sitter.Language }

// NewRustParser returns a Parser for Rust source files.
func NewRustParser() *RustParser { return &RustParser{lang: rustLang.GetLanguage()} }

// Language returns the parser's language identifier.
func (p *RustParser) Language() string { return "rust" }

// Extensions returns the file extensions handled by this parser.
func (p *RustParser) Extensions() []string { return []string{".rs"} }

// Parse extracts file, function, struct, enum, and type alias nodes from src.
func (p *RustParser) Parse(path string, src []byte) (ParseResult, error) {
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(p.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return ParseResult{}, fmt.Errorf("tree-sitter parse %s: %w", path, err)
	}
	defer tree.Close()

	res := ParseResult{InterfaceMethods: map[string][]string{}}
	res.Nodes = append(res.Nodes, store.Node{
		ID: path, Kind: "file", Path: path, Name: path, Language: "rust",
	})
	root := tree.RootNode()
	for i := 0; i < int(root.NamedChildCount()); i++ {
		c := root.NamedChild(i)
		exported := hasRustVisibilityPub(c, src)
		switch c.Type() {
		case "function_item":
			emitRustSymbol(&res, c, src, path, "function", exported)
		case "struct_item":
			emitRustSymbol(&res, c, src, path, "struct", exported)
		case "enum_item":
			emitRustSymbol(&res, c, src, path, "enum", exported)
		case "type_item":
			emitRustSymbol(&res, c, src, path, "type", exported)
		}
	}
	return res, nil
}

func hasRustVisibilityPub(n *sitter.Node, src []byte) bool {
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c.Type() == "visibility_modifier" && c.Content(src) == "pub" {
			return true
		}
	}
	return false
}

func emitRustSymbol(res *ParseResult, decl *sitter.Node, src []byte, path, kind string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: kind, Path: path, Name: name, Language: "rust",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
}
