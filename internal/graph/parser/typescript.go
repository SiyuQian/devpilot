package parser

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	tsLang "github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// TypeScriptParser extracts nodes and edges from .ts source files.
type TypeScriptParser struct{ lang *sitter.Language }

// NewTypeScriptParser returns a Parser for TypeScript source files.
func NewTypeScriptParser() *TypeScriptParser {
	return &TypeScriptParser{lang: tsLang.GetLanguage()}
}

func (p *TypeScriptParser) Language() string     { return "typescript" }
func (p *TypeScriptParser) Extensions() []string { return []string{".ts", ".tsx"} }

func (p *TypeScriptParser) Parse(path string, src []byte) (ParseResult, error) {
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
		ID: path, Kind: "file", Path: path, Name: path, Language: "typescript",
	})

	root := tree.RootNode()
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		exported := false
		decl := child
		if child.Type() == "export_statement" {
			exported = true
			if child.NamedChildCount() > 0 {
				decl = child.NamedChild(0)
			}
		}
		if decl.Type() == "function_declaration" {
			emitFunctionNode(&res, decl, src, path, exported)
		}
		if decl.Type() == "class_declaration" {
			emitClassNode(&res, decl, src, path, exported)
		}
	}
	return res, nil
}

func emitClassNode(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	className := nameNode.Content(src)
	classID := path + "::" + className
	res.Nodes = append(res.Nodes, store.Node{
		ID: classID, Kind: "class", Path: path, Name: className, Language: "typescript",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: classID, Kind: "contains"})

	body := decl.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < int(body.NamedChildCount()); i++ {
		member := body.NamedChild(i)
		if member.Type() != "method_definition" {
			continue
		}
		methodName := member.ChildByFieldName("name")
		if methodName == nil {
			continue
		}
		mName := methodName.Content(src)
		mID := path + "::" + className + "." + mName
		isPrivate := false
		for j := 0; j < int(member.ChildCount()); j++ {
			c := member.Child(j)
			if c.Type() == "accessibility_modifier" && c.Content(src) == "private" {
				isPrivate = true
				break
			}
		}
		res.Nodes = append(res.Nodes, store.Node{
			ID: mID, Kind: "method", Path: path, Name: mName, Container: className,
			Language:   "typescript",
			StartLine:  int(member.StartPoint().Row) + 1,
			EndLine:    int(member.EndPoint().Row) + 1,
			IsExported: !isPrivate,
		})
		res.Edges = append(res.Edges, store.Edge{Src: classID, Kind: "contains", Dst: mID})
	}
}

func emitFunctionNode(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: "function", Path: path, Name: name, Language: "typescript",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
}
