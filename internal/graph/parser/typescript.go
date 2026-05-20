package parser

import (
	"context"
	"fmt"
	"strings"

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
	intra := map[string]string{}
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		decl := child
		if child.Type() == "export_statement" && child.NamedChildCount() > 0 {
			decl = child.NamedChild(0)
		}
		switch decl.Type() {
		case "function_declaration":
			if n := decl.ChildByFieldName("name"); n != nil {
				intra[n.Content(src)] = path + "::" + n.Content(src)
			}
		}
	}

	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.Type() == "import_statement" {
			srcNode := child.ChildByFieldName("source")
			if srcNode != nil {
				modulePath := unquote(srcNode.Content(src))
				res.Edges = append(res.Edges, store.Edge{
					Src: path, Dst: "external::" + modulePath, Kind: "imports",
				})
			}
			continue
		}
		exported := false
		decl := child
		if child.Type() == "export_statement" {
			exported = true
			if child.NamedChildCount() > 0 {
				decl = child.NamedChild(0)
			}
		}
		switch decl.Type() {
		case "function_declaration":
			emitFunctionNode(&res, decl, src, path, exported, intra)
		case "class_declaration":
			emitClassNode(&res, decl, src, path, exported, intra)
		case "interface_declaration":
			emitInterfaceNode(&res, decl, src, path, exported)
		case "type_alias_declaration":
			emitTypeAliasNode(&res, decl, src, path, exported)
		}
	}
	if isTestFile(path) {
		res.Edges = append(res.Edges, extractTSTestEdges(root, src, path)...)
	}
	return res, nil
}

func isTestFile(path string) bool {
	for _, suf := range []string{".test.ts", ".spec.ts", ".test.tsx", ".spec.tsx", ".test.js", ".spec.js"} {
		if strings.HasSuffix(path, suf) {
			return true
		}
	}
	return false
}

func extractTSTestEdges(root *sitter.Node, src []byte, path string) []store.Edge {
	var out []store.Edge
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "call_expression" {
			fn := n.ChildByFieldName("function")
			if fn != nil && fn.Type() == "identifier" {
				if name := fn.Content(src); name == "describe" || name == "it" || name == "test" {
					args := n.ChildByFieldName("arguments")
					if args != nil {
						for i := 0; i < int(args.NamedChildCount()); i++ {
							out = append(out, extractCallTargets(args.NamedChild(i), src, path)...)
						}
					}
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(root)
	return out
}

func extractCallTargets(n *sitter.Node, src []byte, path string) []store.Edge {
	var out []store.Edge
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "call_expression" {
			fn := n.ChildByFieldName("function")
			if fn != nil && fn.Type() == "identifier" {
				out = append(out, store.Edge{Src: path, Dst: "external::" + fn.Content(src), Kind: "tests"})
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(n)
	return out
}

func emitClassNode(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool, intra map[string]string) {
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
		res.Edges = append(res.Edges, store.Edge{Src: classID, Dst: mID, Kind: "contains"})
		if body := member.ChildByFieldName("body"); body != nil {
			res.Edges = append(res.Edges, walkTSCalls(body, src, mID, intra)...)
		}
	}
}

func emitFunctionNode(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool, intra map[string]string) {
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
	if body := decl.ChildByFieldName("body"); body != nil {
		res.Edges = append(res.Edges, walkTSCalls(body, src, id, intra)...)
	}
}

func walkTSCalls(body *sitter.Node, src []byte, srcID string, intra map[string]string) []store.Edge {
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
					if dst, ok := intra[name]; ok {
						out = append(out, store.Edge{Src: srcID, Dst: dst, Kind: "calls"})
					} else {
						out = append(out, store.Edge{Src: srcID, Dst: "external::" + name, Kind: "calls"})
					}
				case "member_expression":
					obj := fn.ChildByFieldName("object")
					prop := fn.ChildByFieldName("property")
					if obj != nil && prop != nil {
						out = append(out, store.Edge{Src: srcID, Dst: "external::" + obj.Content(src) + "." + prop.Content(src), Kind: "calls"})
					}
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

func emitInterfaceNode(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: "interface", Path: path, Name: name, Language: "typescript",
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
		member := body.NamedChild(i)
		if member.Type() != "method_signature" {
			continue
		}
		mName := member.ChildByFieldName("name")
		if mName != nil {
			methods = append(methods, mName.Content(src))
		}
	}
	if len(methods) > 0 {
		res.InterfaceMethods[id] = methods
	}
}

func emitTypeAliasNode(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: "type", Path: path, Name: name, Language: "typescript",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
}
