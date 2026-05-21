package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// GoNativeParser extracts nodes and edges from Go source files using the native
// Go backend (whole-module analysis via go/packages).
type GoNativeParser struct{}

// NewGoNativeParser returns a Parser for Go source files using the native backend.
func NewGoNativeParser() *GoNativeParser {
	return &GoNativeParser{}
}

func (p *GoNativeParser) Language() string {
	return "go"
}

func (p *GoNativeParser) Extensions() []string {
	return []string{".go"}
}

// Parse is intentionally a no-op; the native backend produces results via
// LoadModule on the whole module, not per-file Parse.
func (p *GoNativeParser) Parse(path string, src []byte) (ParseResult, error) {
	return ParseResult{}, nil
}

// LoadModule type-checks the whole Go module rooted at repoRoot and returns a
// map keyed by repo-relative file path. Each ParseResult contains the file node
// plus all top-level function and method nodes declared in that file, with
// `contains` edges from file -> declaration.
//
// Partial-package errors (per-package go/packages errors) are surfaced via
// ParseResult.Errors under a synthetic "" key. The function only returns a
// hard error if the loader itself fails or no packages can be loaded at all.
func (p *GoNativeParser) LoadModule(repoRoot string) (map[string]ParseResult, error) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("abs %s: %w", repoRoot, err)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports |
			packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo,
		Dir:   abs,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("packages.Load %s: %w", abs, err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages loaded under %s", abs)
	}

	// Process non-test-binary packages first to ensure source-package ownership
	// wins over the `pkg.test` synthetic variant when a file appears in both.
	type pkgEntry struct {
		pkg       *packages.Package
		isDotTest bool
	}
	entries := make([]pkgEntry, 0, len(pkgs))
	for _, pk := range pkgs {
		entries = append(entries, pkgEntry{pkg: pk, isDotTest: strings.HasSuffix(pk.ID, ".test")})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].isDotTest != entries[j].isDotTest {
			return !entries[i].isDotTest // non-.test first
		}
		return entries[i].pkg.ID < entries[j].pkg.ID
	})

	sep := string(filepath.Separator)
	prefix := abs + sep

	results := map[string]ParseResult{}
	// seen[relPath][declID] = true — dedupe across Tests:true duplication.
	seen := map[string]map[string]bool{}
	// fileEmitted tracks which relPaths already have a file node in their result.
	fileEmitted := map[string]bool{}

	var allErrors []ParseError
	usable := 0

	for _, ent := range entries {
		pk := ent.pkg
		if len(pk.Errors) > 0 {
			for _, e := range pk.Errors {
				pe := ParseError{Message: e.Msg}
				// e.Pos format: "file:line:col" (may be empty).
				if e.Pos != "" {
					parts := strings.SplitN(e.Pos, ":", 3)
					pe.Path = relFromPrefix(parts[0], prefix)
					if len(parts) >= 2 {
						var line int
						if _, scanErr := fmt.Sscanf(parts[1], "%d", &line); scanErr == nil {
							pe.Line = line
						}
					}
				} else {
					pe.Path = pk.PkgPath
				}
				allErrors = append(allErrors, pe)
			}
		}
		// Skip the synthetic test binary; its files are owned by the source/_test pkgs.
		if ent.isDotTest {
			continue
		}
		if pk.Fset == nil || len(pk.Syntax) == 0 {
			continue
		}
		usable++

		// Pair Syntax with CompiledGoFiles by index, but sort by filename for determinism.
		type fileEntry struct {
			file *ast.File
			name string
		}
		files := make([]fileEntry, 0, len(pk.Syntax))
		for _, f := range pk.Syntax {
			pos := pk.Fset.Position(f.Pos())
			files = append(files, fileEntry{file: f, name: pos.Filename})
		}
		sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

		for _, fe := range files {
			fname := fe.name
			if fname == "" || !strings.HasPrefix(fname, prefix) {
				continue // cgo/generated/synthetic
			}
			relPath := filepath.ToSlash(strings.TrimPrefix(fname, prefix))

			res := results[relPath]
			if !fileEmitted[relPath] {
				res.Nodes = append(res.Nodes, store.Node{
					ID: relPath, Kind: "file", Path: relPath, Name: relPath, Language: "go",
				})
				fileEmitted[relPath] = true
			}
			if seen[relPath] == nil {
				seen[relPath] = map[string]bool{}
			}

			for _, decl := range fe.file.Decls {
				// Handle function and method declarations.
				fd, ok := decl.(*ast.FuncDecl)
				if ok && fd.Name != nil {
					name := fd.Name.Name
					var (
						declID    string
						kind      string
						container string
					)
					if fd.Recv != nil && len(fd.Recv.List) == 1 {
						recvType := receiverTypeName(fd.Recv.List[0].Type)
						if recvType != "" {
							container = recvType
							kind = "method"
							declID = relPath + "::" + recvType + "." + name
						}
					}
					if declID == "" {
						kind = "function"
						declID = relPath + "::" + name
					}
					if seen[relPath][declID] {
						continue
					}
					seen[relPath][declID] = true

					startLine := pk.Fset.Position(fd.Pos()).Line
					endLine := pk.Fset.Position(fd.End()).Line
					res.Nodes = append(res.Nodes, store.Node{
						ID: declID, Kind: kind, Path: relPath, Name: name,
						Container: container, Language: "go",
						StartLine: startLine, EndLine: endLine,
						IsExported: ast.IsExported(name),
					})
					res.Edges = append(res.Edges, store.Edge{
						Src: relPath, Dst: declID, Kind: "contains",
					})
					continue
				}

				// Handle type, interface, and struct declarations.
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || ts.Name == nil {
						continue
					}
					name := ts.Name.Name
					obj := pk.TypesInfo.Defs[ts.Name]
					if obj == nil {
						continue
					}

					// Determine kind: alias, interface, struct, or default type.
					var kind string
					if ts.Assign != token.NoPos {
						// type Alias = X (alias form)
						kind = "type"
					} else {
						// Inspect the underlying type.
						switch obj.Type().Underlying().(type) {
						case *types.Interface:
							kind = "interface"
						case *types.Struct:
							kind = "struct"
						default:
							kind = "type"
						}
					}

					declID := relPath + "::" + name
					if seen[relPath][declID] {
						continue
					}
					seen[relPath][declID] = true

					startLine := pk.Fset.Position(ts.Pos()).Line
					endLine := pk.Fset.Position(ts.End()).Line
					res.Nodes = append(res.Nodes, store.Node{
						ID: declID, Kind: kind, Path: relPath, Name: name,
						Language:  "go",
						StartLine: startLine, EndLine: endLine,
						IsExported: ast.IsExported(name),
					})
					res.Edges = append(res.Edges, store.Edge{
						Src: relPath, Dst: declID, Kind: "contains",
					})
				}
			}
			results[relPath] = res
		}
	}

	if usable == 0 {
		return nil, fmt.Errorf("no usable packages loaded under %s (errors: %d)", abs, len(allErrors))
	}

	// Sort nodes/edges within each result for determinism.
	for k, r := range results {
		sort.Slice(r.Nodes, func(i, j int) bool { return r.Nodes[i].ID < r.Nodes[j].ID })
		sort.Slice(r.Edges, func(i, j int) bool {
			if r.Edges[i].Src != r.Edges[j].Src {
				return r.Edges[i].Src < r.Edges[j].Src
			}
			if r.Edges[i].Dst != r.Edges[j].Dst {
				return r.Edges[i].Dst < r.Edges[j].Dst
			}
			return r.Edges[i].Kind < r.Edges[j].Kind
		})
		results[k] = r
	}

	// Attach all package-level errors under the synthetic "" key.
	if len(allErrors) > 0 {
		sort.Slice(allErrors, func(i, j int) bool {
			if allErrors[i].Path != allErrors[j].Path {
				return allErrors[i].Path < allErrors[j].Path
			}
			if allErrors[i].Line != allErrors[j].Line {
				return allErrors[i].Line < allErrors[j].Line
			}
			return allErrors[i].Message < allErrors[j].Message
		})
		errRes := results[""]
		errRes.Errors = append(errRes.Errors, allErrors...)
		results[""] = errRes
	}

	return results, nil
}

// receiverTypeName extracts the bare receiver type name from an ast receiver
// expression, stripping a leading '*' for pointer receivers and any generic
// type-parameter list (e.g. Foo[T] -> Foo).
func receiverTypeName(expr ast.Expr) string {
	t := types.ExprString(expr)
	t = strings.TrimPrefix(t, "*")
	if i := strings.IndexByte(t, '['); i >= 0 {
		t = t[:i]
	}
	return t
}

// relFromPrefix returns path relative to prefix if it lives inside prefix; otherwise
// the original (likely absolute) path is returned unchanged. Used for shaping
// ParseError.Path consistently with node Path values.
func relFromPrefix(path, prefix string) string {
	if strings.HasPrefix(path, prefix) {
		return filepath.ToSlash(strings.TrimPrefix(path, prefix))
	}
	return path
}
