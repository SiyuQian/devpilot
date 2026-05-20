package parser

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	jsLang "github.com/smacker/go-tree-sitter/javascript"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// JavaScriptParser extracts nodes and edges from .js/.mjs/.cjs source files.
type JavaScriptParser struct{ lang *sitter.Language }

// NewJavaScriptParser returns a Parser for JavaScript source files.
func NewJavaScriptParser() *JavaScriptParser {
	return &JavaScriptParser{lang: jsLang.GetLanguage()}
}

func (p *JavaScriptParser) Language() string     { return "javascript" }
func (p *JavaScriptParser) Extensions() []string { return []string{".js", ".mjs", ".cjs"} }

func (p *JavaScriptParser) Parse(path string, src []byte) (ParseResult, error) {
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
		ID: path, Kind: "file", Path: path, Name: path, Language: "javascript",
	})
	root := tree.RootNode()

	intra := map[string]string{}
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		decl := child
		if child.Type() == "export_statement" && child.NamedChildCount() > 0 {
			decl = child.NamedChild(0)
		}
		if name := nameOf(decl, src); name != "" {
			intra[name] = path + "::" + name
		}
	}

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
		switch decl.Type() {
		case "function_declaration":
			emitJSFunction(&res, decl, src, path, exported, intra)
		case "class_declaration":
			emitJSClass(&res, decl, src, path, exported, intra)
		case "import_statement":
			if s := decl.ChildByFieldName("source"); s != nil {
				res.Edges = append(res.Edges, store.Edge{
					Src: path, Dst: "external::" + unquote(s.Content(src)), Kind: "imports",
				})
			}
		}
	}

	if isJSTestFile(path) {
		res.Edges = append(res.Edges, extractTSTestEdges(root, src, path)...)
	}
	return res, nil
}

func nameOf(decl *sitter.Node, src []byte) string {
	if decl == nil {
		return ""
	}
	n := decl.ChildByFieldName("name")
	if n == nil {
		return ""
	}
	return n.Content(src)
}

func emitJSFunction(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool, intra map[string]string) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := nameNode.Content(src)
	id := path + "::" + name
	res.Nodes = append(res.Nodes, store.Node{
		ID: id, Kind: "function", Path: path, Name: name, Language: "javascript",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
	if body := decl.ChildByFieldName("body"); body != nil {
		res.Edges = append(res.Edges, walkTSCalls(body, src, id, intra)...)
	}
}

func emitJSClass(res *ParseResult, decl *sitter.Node, src []byte, path string, exported bool, intra map[string]string) {
	nameNode := decl.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	className := nameNode.Content(src)
	classID := path + "::" + className
	res.Nodes = append(res.Nodes, store.Node{
		ID: classID, Kind: "class", Path: path, Name: className, Language: "javascript",
		StartLine:  int(decl.StartPoint().Row) + 1,
		EndLine:    int(decl.EndPoint().Row) + 1,
		IsExported: exported,
	})
	res.Edges = append(res.Edges, store.Edge{Src: path, Dst: classID, Kind: "contains"})

	for i := 0; i < int(decl.ChildCount()); i++ {
		c := decl.Child(i)
		if c.Type() == "class_heritage" {
			for j := 0; j < int(c.NamedChildCount()); j++ {
				name := c.NamedChild(j).Content(src)
				res.Edges = append(res.Edges, store.Edge{
					Src: classID, Dst: resolveIntra(name, intra), Kind: "extends",
				})
			}
		}
	}

	body := decl.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < int(body.NamedChildCount()); i++ {
		member := body.NamedChild(i)
		if member.Type() != "method_definition" {
			continue
		}
		mNameNode := member.ChildByFieldName("name")
		if mNameNode == nil {
			continue
		}
		mName := mNameNode.Content(src)
		mID := path + "::" + className + "." + mName
		res.Nodes = append(res.Nodes, store.Node{
			ID: mID, Kind: "method", Path: path, Name: mName, Container: className,
			Language: "javascript", IsExported: true,
			StartLine: int(member.StartPoint().Row) + 1,
			EndLine:   int(member.EndPoint().Row) + 1,
		})
		res.Edges = append(res.Edges, store.Edge{Src: classID, Dst: mID, Kind: "contains"})
		if mBody := member.ChildByFieldName("body"); mBody != nil {
			res.Edges = append(res.Edges, walkTSCalls(mBody, src, mID, intra)...)
		}
	}
}

func isJSTestFile(path string) bool {
	for _, suf := range []string{".test.js", ".spec.js", ".test.mjs", ".spec.mjs"} {
		if strings.HasSuffix(path, suf) {
			return true
		}
	}
	return false
}
