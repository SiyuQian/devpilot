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
	defer parser.Close()
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

	// First pass: collect top-level function names for intra-file call resolution.
	intraFileFuncs := map[string]bool{}
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.Type() == "function_declaration" {
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				intraFileFuncs[nameNode.Content(src)] = true
			}
		}
	}

	// Second pass: emit nodes and edges.
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
			if bodyNode := child.ChildByFieldName("body"); bodyNode != nil {
				callEdges := walkCalls(bodyNode, src, path, id, intraFileFuncs)
				res.Edges = append(res.Edges, callEdges...)

				if isGoTestFunc(name, child, src) {
					for _, e := range callEdges {
						res.Edges = append(res.Edges, store.Edge{Src: e.Src, Dst: e.Dst, Kind: "tests"})
					}
				}
			}
		}
		if child.Type() == "type_declaration" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				spec := child.NamedChild(j)
				if spec.Type() != "type_spec" && spec.Type() != "type_alias" {
					continue
				}
				nameNode := spec.ChildByFieldName("name")
				if nameNode == nil {
					continue
				}
				name := nameNode.Content(src)
				kind := classifyGoTypeSpec(spec)
				id := path + "::" + name
				if kind == "interface" {
					typeNode := spec.ChildByFieldName("type")
					if typeNode != nil {
						methods := extractGoInterfaceMethods(typeNode, src)
						if len(methods) > 0 {
							if res.InterfaceMethods == nil {
								res.InterfaceMethods = map[string][]string{}
							}
							res.InterfaceMethods[id] = methods
						}
					}
				}
				res.Nodes = append(res.Nodes, store.Node{
					ID: id, Kind: kind, Path: path, Name: name, Language: "go",
					StartLine:  int(spec.StartPoint().Row) + 1,
					EndLine:    int(spec.EndPoint().Row) + 1,
					IsExported: isExportedGo(name),
				})
				res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
			}
		}
		if child.Type() == "import_declaration" {
			// Walk every import_spec descendant
			for j := 0; j < int(child.NamedChildCount()); j++ {
				sub := child.NamedChild(j)
				switch sub.Type() {
				case "import_spec":
					if pkg := importSpecPath(sub, src); pkg != "" {
						res.Edges = append(res.Edges, store.Edge{
							Src: path, Dst: "external::" + pkg, Kind: "imports",
						})
					}
				case "import_spec_list":
					for k := 0; k < int(sub.NamedChildCount()); k++ {
						spec := sub.NamedChild(k)
						if spec.Type() != "import_spec" {
							continue
						}
						if pkg := importSpecPath(spec, src); pkg != "" {
							res.Edges = append(res.Edges, store.Edge{
								Src: path, Dst: "external::" + pkg, Kind: "imports",
							})
						}
					}
				}
			}
		}
		if child.Type() == "method_declaration" {
			nameNode := child.ChildByFieldName("name")
			recvNode := child.ChildByFieldName("receiver")
			if nameNode == nil || recvNode == nil {
				continue
			}
			name := nameNode.Content(src)
			recvType := extractGoReceiverType(recvNode, src)
			if recvType == "" {
				continue
			}
			id := fmt.Sprintf("%s::%s.%s", path, recvType, name)
			res.Nodes = append(res.Nodes, store.Node{
				ID: id, Kind: "method", Path: path, Name: name, Container: recvType, Language: "go",
				StartLine:  int(child.StartPoint().Row) + 1,
				EndLine:    int(child.EndPoint().Row) + 1,
				IsExported: isExportedGo(name),
			})
			res.Edges = append(res.Edges, store.Edge{Src: path, Dst: id, Kind: "contains"})
			bodyNode := child.ChildByFieldName("body")
			if bodyNode != nil {
				res.Edges = append(res.Edges, walkCalls(bodyNode, src, path, id, intraFileFuncs)...)
			}
		}
	}
	return res, nil
}

// walkCalls walks node's subtree, emitting a "calls" edge from srcID for every
// call_expression encountered. Local symbols (Greet) emit edges to
// path+"::"+Name; selector expressions (pkg.Sym) emit edges to "external::"+sel.
func walkCalls(n *sitter.Node, src []byte, path, srcID string, intraFile map[string]bool) []store.Edge {
	var edges []store.Edge

	var visit func(node *sitter.Node)
	visit = func(node *sitter.Node) {
		if node.Type() == "call_expression" {
			fnNode := node.ChildByFieldName("function")
			if fnNode != nil {
				dst := resolveCallDst(fnNode, src, path, intraFile)
				if dst != "" {
					edges = append(edges, store.Edge{Src: srcID, Dst: dst, Kind: "calls"})
				}
			}
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			visit(node.NamedChild(i))
		}
	}
	visit(n)
	return edges
}

// resolveCallDst maps a call_expression's "function" child to a node ID.
//   - identifier "Foo"      -> path+"::Foo" if Foo is defined intra-file, else external::Foo
//   - selector "pkg.Sym"    -> external::pkg.Sym
//   - all others (closures, etc.) -> ""
func resolveCallDst(fn *sitter.Node, src []byte, path string, intraFile map[string]bool) string {
	switch fn.Type() {
	case "identifier":
		name := fn.Content(src)
		if intraFile[name] {
			return path + "::" + name
		}
		return "external::" + name
	case "selector_expression":
		// pkg.Sym — capture "pkg.Sym" verbatim
		return "external::" + fn.Content(src)
	}
	return ""
}

// extractGoReceiverType returns the Greeter in "(g *Greeter)" or "(g Greeter)".
// Returns "" if the receiver type cannot be determined.
func extractGoReceiverType(recv *sitter.Node, src []byte) string {
	for i := 0; i < int(recv.NamedChildCount()); i++ {
		c := recv.NamedChild(i)
		if c.Type() == "parameter_declaration" {
			typeNode := c.ChildByFieldName("type")
			if typeNode == nil {
				continue
			}
			t := typeNode.Content(src)
			// strip leading '*' for pointer receivers
			if len(t) > 0 && t[0] == '*' {
				t = t[1:]
			}
			// strip generic type parameters: Foo[T] -> Foo
			if idx := indexByteFast(t, '['); idx >= 0 {
				t = t[:idx]
			}
			return t
		}
	}
	return ""
}

func indexByteFast(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// extractGoInterfaceMethods walks an interface_type subtree and returns the
// names of its declared methods (excluding embedded types).
func extractGoInterfaceMethods(ifaceType *sitter.Node, src []byte) []string {
	var names []string
	for i := 0; i < int(ifaceType.NamedChildCount()); i++ {
		c := ifaceType.NamedChild(i)
		if c.Type() == "method_elem" || c.Type() == "method_spec" {
			nameNode := c.ChildByFieldName("name")
			if nameNode != nil {
				names = append(names, nameNode.Content(src))
			}
		}
	}
	return names
}

// classifyGoTypeSpec returns "struct", "interface", or "type" based on the
// underlying form of a type spec (e.g. struct{}, interface{...}, alias).
func classifyGoTypeSpec(spec *sitter.Node) string {
	typeNode := spec.ChildByFieldName("type")
	if typeNode == nil {
		return "type"
	}
	switch typeNode.Type() {
	case "struct_type":
		return "struct"
	case "interface_type":
		return "interface"
	default:
		return "type"
	}
}

// isGoTestFunc returns true if name starts with "Test" and the function has
// a parameter of type *testing.T.
func isGoTestFunc(name string, fn *sitter.Node, src []byte) bool {
	if len(name) < 4 || name[:4] != "Test" {
		return false
	}
	params := fn.ChildByFieldName("parameters")
	if params == nil {
		return false
	}
	for i := 0; i < int(params.NamedChildCount()); i++ {
		p := params.NamedChild(i)
		if p.Type() != "parameter_declaration" {
			continue
		}
		typeNode := p.ChildByFieldName("type")
		if typeNode == nil {
			continue
		}
		t := typeNode.Content(src)
		if t == "*testing.T" {
			return true
		}
	}
	return false
}

func isExportedGo(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper([]rune(name)[0])
}

// importSpecPath extracts the import path from an import_spec, stripping the
// quotes. For aliased imports (e.g. `alias "strings"`) it returns the path,
// not the alias. Returns the final path segment (e.g. "strings" for
// "github.com/foo/strings") to keep external IDs short.
//
// For v1 we keep just the last segment to match the test expectation. Later
// tasks can refine this to keep the full path when needed.
func importSpecPath(spec *sitter.Node, src []byte) string {
	pathNode := spec.ChildByFieldName("path")
	if pathNode == nil {
		return ""
	}
	raw := pathNode.Content(src)
	// strip surrounding quotes
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	// Take final segment
	if idx := lastSlash(raw); idx >= 0 {
		raw = raw[idx+1:]
	}
	return raw
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}
