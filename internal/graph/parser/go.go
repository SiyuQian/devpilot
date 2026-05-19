package parser

import (
	"context"
	"fmt"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
	goLang "github.com/smacker/go-tree-sitter/golang"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// GoParser extracts nodes and edges from Go source files.
type GoParser struct{ lang *sitter.Language }

// NewGoParser returns a Parser for Go source files.
func NewGoParser() *GoParser { return &GoParser{lang: goLang.GetLanguage()} }

func (p *GoParser) Language() string     { return "go" }
func (p *GoParser) Extensions() []string { return []string{".go"} }

// Parse extracts the file node and top-level function declarations.
// Methods, types, calls, imports, and tests edges are added in later tasks.
func (p *GoParser) Parse(path string, src []byte) (ParseResult, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(p.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return ParseResult{}, fmt.Errorf("tree-sitter parse %s: %w", path, err)
	}
	defer tree.Close()

	res := ParseResult{}
	res.Nodes = append(res.Nodes, store.Node{
		ID: path, Kind: "file", Path: path, Name: path, Language: "go",
	})

	root := tree.RootNode()
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.Type() == "function_declaration" {
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			name := nameNode.Content(src)
			id := path + "::" + name
			res.Nodes = append(res.Nodes, store.Node{
				ID: id, Kind: "function", Path: path, Name: name, Language: "go",
				StartLine:  int(child.StartPoint().Row) + 1,
				EndLine:    int(child.EndPoint().Row) + 1,
				IsExported: isExportedGo(name),
			})
			res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
		}
	}
	return res, nil
}

func isExportedGo(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper([]rune(name)[0])
}
