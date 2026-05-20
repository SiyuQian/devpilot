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

	// Pre-pass: collect top-level function names for intra-file call resolution.
	intra := map[string]string{}
	for i := 0; i < int(root.NamedChildCount()); i++ {
		c := root.NamedChild(i)
		if c.Type() == "function_item" {
			if n := c.ChildByFieldName("name"); n != nil {
				name := n.Content(src)
				intra[name] = path + "::" + name
			}
		}
	}

	for i := 0; i < int(root.NamedChildCount()); i++ {
		c := root.NamedChild(i)
		exported := hasRustVisibilityPub(c, src)
		switch c.Type() {
		case "function_item":
			emitRustSymbol(&res, c, src, path, "function", exported)
			if body := c.ChildByFieldName("body"); body != nil {
				if n := c.ChildByFieldName("name"); n != nil {
					id := path + "::" + n.Content(src)
					res.Edges = append(res.Edges, walkRustCalls(body, src, id, intra)...)
				}
			}
		case "struct_item":
			emitRustSymbol(&res, c, src, path, "struct", exported)
		case "enum_item":
			emitRustSymbol(&res, c, src, path, "enum", exported)
		case "type_item":
			emitRustSymbol(&res, c, src, path, "type", exported)
		case "trait_item":
			emitRustTrait(&res, c, src, path, exported)
		case "impl_item":
			emitRustImpl(&res, c, src, path, intra)
		case "use_declaration":
			if c.NamedChildCount() > 0 {
				usePath := flattenUseTree(c.NamedChild(0), src)
				if usePath != "" {
					res.Edges = append(res.Edges, store.Edge{
						Src: path, Dst: "external::" + usePath, Kind: "imports",
					})
				}
			}
		}
	}
	return res, nil
}

// flattenUseTree returns the scoped path text of a use_declaration's inner node.
// For "use std::fmt::Display;" the inner node is a scoped_identifier whose
// Content() already yields "std::fmt::Display".
func flattenUseTree(n *sitter.Node, src []byte) string {
	if n == nil {
		return ""
	}
	return n.Content(src)
}

// walkRustCalls emits "calls" edges from srcID for every call_expression in body.
// Identifier-form calls resolve via intra first; on miss they fall through to
// external::Name (no dot) so resolver.Resolve can rewrite them to real targets
// in other files of the same module. Scoped/field calls always go external.
func walkRustCalls(body *sitter.Node, src []byte, srcID string, intra map[string]string) []store.Edge {
	var out []store.Edge
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "call_expression" {
			fn := n.ChildByFieldName("function")
			if fn != nil {
				switch fn.Type() {
				case "identifier":
					name := fn.Content(src)
					if id, ok := intra[name]; ok {
						out = append(out, store.Edge{Src: srcID, Dst: id, Kind: "calls"})
					} else {
						out = append(out, store.Edge{Src: srcID, Dst: "external::" + name, Kind: "calls"})
					}
				case "scoped_identifier", "field_expression":
					out = append(out, store.Edge{Src: srcID, Dst: "external::" + fn.Content(src), Kind: "calls"})
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(body)
	return out
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

func emitRustTrait(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: "interface", Path: path, Name: name, Language: "rust",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})

	body := decl.ChildByFieldName("body")
	if body == nil {
		return
	}
	var methods []string
	for i := 0; i < int(body.NamedChildCount()); i++ {
		m := body.NamedChild(i)
		if m.Type() == "function_signature_item" {
			if mn := m.ChildByFieldName("name"); mn != nil {
				methods = append(methods, mn.Content(src))
			}
		}
	}
	if len(methods) > 0 {
		res.InterfaceMethods[id] = methods
	}
}

func emitRustImpl(res *ParseResult, decl *sitter.Node, src []byte, path string, intra map[string]string) {
	traitNode := decl.ChildByFieldName("trait")
	typeNode := decl.ChildByFieldName("type")
	if typeNode == nil {
		return
	}
	typeName := typeNode.Content(src)
	typeID := path + "::" + typeName
	if traitNode != nil {
		traitName := traitNode.Content(src)
		res.Edges = append(res.Edges, store.Edge{
			Src: typeID, Dst: path + "::" + traitName, Kind: "implements",
		})
	}
	body := decl.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < int(body.NamedChildCount()); i++ {
		fn := body.NamedChild(i)
		if fn.Type() != "function_item" {
			continue
		}
		nameNode := fn.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		mName := nameNode.Content(src)
		mID := path + "::" + typeName + "." + mName
		res.Nodes = append(res.Nodes, store.Node{
			ID: mID, Kind: "method", Path: path, Name: mName, Container: typeName,
			Language:   "rust",
			StartLine:  int(fn.StartPoint().Row) + 1,
			EndLine:    int(fn.EndPoint().Row) + 1,
			IsExported: hasRustVisibilityPub(fn, src),
		})
		res.Edges = append(res.Edges, store.Edge{Src: typeID, Dst: mID, Kind: "contains"})
		if body := fn.ChildByFieldName("body"); body != nil {
			res.Edges = append(res.Edges, walkRustCalls(body, src, mID, intra)...)
		}
	}
}
