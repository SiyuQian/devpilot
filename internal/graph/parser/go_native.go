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

// objIndexKey creates a string key for objIndex from a types.Object.
// For *types.Func (functions and methods):
//   - Functions: "pkg/path::Name"
//   - Methods: "pkg/path::ReceiverTypeName.Name"
//
// For *types.TypeName: "pkg/path::Name"
//
// For methods, the receiver type is extracted from the function's signature,
// with pointer layers stripped. This ensures two methods named Speak on
// different receivers (e.g., Console.Speak and PartialSpeaker.Speak) collide
// if they have different receiver types.
func objIndexKey(obj types.Object) string {
	if obj.Pkg() == nil {
		return ""
	}
	switch t := obj.(type) {
	case *types.Func:
		sig, ok := t.Type().(*types.Signature)
		if ok && sig.Recv() != nil {
			recv := sig.Recv().Type()
			// Strip pointer layers
			for {
				if p, ok := recv.(*types.Pointer); ok {
					recv = p.Elem()
					continue
				}
				break
			}
			// Get the recv type's name (handles named, alias, generic-instantiated)
			if named, ok := recv.(*types.Named); ok {
				return t.Pkg().Path() + "::" + named.Obj().Name() + "." + t.Name()
			}
			// Fallback for non-named receiver (extremely rare; e.g. method on alias of basic).
			return t.Pkg().Path() + "::" + types.TypeString(recv, nil) + "." + t.Name()
		}
		return t.Pkg().Path() + "::" + t.Name()
	case *types.TypeName:
		return t.Pkg().Path() + "::" + t.Name()
	default:
		return ""
	}
}

// isGoTestFuncNative checks if a function declaration is a valid Go test function.
// A test function must:
// - be declared in a file ending in _test.go
// - have a name starting with "Test"
// - have exactly one parameter
// - that parameter's type is *testing.T
//
// The function uses resolved type information (Types.Info.Defs) to confirm
// the parameter type, avoiding fragile string matching on import aliases.
func isGoTestFuncNative(fd *ast.FuncDecl, fname string, pkg *packages.Package) bool {
	if fd.Name == nil || !strings.HasSuffix(fname, "_test.go") {
		return false
	}
	if !strings.HasPrefix(fd.Name.Name, "Test") {
		return false
	}
	if fd.Type == nil || fd.Type.Params == nil || len(fd.Type.Params.List) != 1 {
		return false
	}
	param := fd.Type.Params.List[0]
	if len(param.Names) == 0 {
		return false
	}
	paramIdent := param.Names[0]
	if pkg.TypesInfo == nil {
		return false
	}
	obj := pkg.TypesInfo.Defs[paramIdent]
	if obj == nil {
		return false
	}
	// obj should be a *types.Var; get its type.
	v, ok := obj.(*types.Var)
	if !ok {
		return false
	}
	t := v.Type()
	// Should be *testing.T: a pointer to a named type T in package "testing".
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	tyObj := named.Obj()
	if tyObj == nil {
		return false
	}
	tyPkg := tyObj.Pkg()
	if tyPkg == nil {
		return false
	}
	return tyPkg.Path() == "testing" && tyObj.Name() == "T"
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
	// objIndex maps a (package path, symbol name) to its emitted node ID. Built in
	// pass 1 so pass 2 can resolve cross-package callees to real symbol IDs.
	// Stored as pkg.Path()::Name for functions, and pkg.Path()::ReceiverType.Name for methods.
	objIndex := map[string]string{}
	// inModule is the set of package paths owned by some non-.test package in
	// this load — used to skip call edges targeting deps outside the module.
	inModule := map[string]bool{}
	// Candidate types gathered in pass 1 for the implements pass. Aliases are
	// excluded (skipped). typeFile maps a type-name object to the relPath of
	// the file that declares it.
	var concreteTypes []*types.TypeName
	var interfaceTypes []*types.TypeName
	typeFile := map[*types.TypeName]string{}

	// Each pendingCall is a callsite to resolve in pass 2 once objIndex is full.
	type pendingCall struct {
		relPath string
		srcID   string
		pkg     *packages.Package
		body    *ast.BlockStmt
		isTest  bool
	}
	var pending []pendingCall

	// primaryFile maps pkgPath -> relPath of the lexically smallest non-test .go file.
	// Pre-built before the main loop so imports edges can target it in pass 1.
	primaryFile := map[string]string{}
	for _, ent := range entries {
		if ent.isDotTest || len(ent.pkg.Syntax) == 0 || ent.pkg.PkgPath == "" {
			continue
		}

		// Collect non-test files for this package.
		type pkgFileEntry struct {
			relPath string
		}
		var pkgFiles []pkgFileEntry
		for _, f := range ent.pkg.Syntax {
			pos := ent.pkg.Fset.Position(f.Pos())
			fname := pos.Filename
			if fname == "" || !strings.HasPrefix(fname, prefix) {
				continue
			}
			relPath := filepath.ToSlash(strings.TrimPrefix(fname, prefix))
			// Skip test files.
			if strings.HasSuffix(relPath, "_test.go") {
				continue
			}
			pkgFiles = append(pkgFiles, pkgFileEntry{relPath: relPath})
		}

		// Sort by relPath and pick the first (lexically smallest).
		if len(pkgFiles) > 0 {
			sort.Slice(pkgFiles, func(i, j int) bool {
				return pkgFiles[i].relPath < pkgFiles[j].relPath
			})
			primaryFile[ent.pkg.PkgPath] = pkgFiles[0].relPath
		}
	}

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
		if pk.PkgPath != "" {
			inModule[pk.PkgPath] = true
		}

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

			// Collect imports for this file (deduped).
			seenImport := map[string]bool{}
			for _, imp := range fe.file.Imports {
				importPath := strings.Trim(imp.Path.Value, "\"")
				if seenImport[importPath] {
					continue
				}
				seenImport[importPath] = true
				// Skip if not in this module (stdlib, third-party, etc.)
				if !inModule[importPath] {
					continue
				}
				// Look up the primary file for the imported package.
				dstRel, ok := primaryFile[importPath]
				if !ok || dstRel == "" {
					continue // no primary file for this package
				}
				res.Edges = append(res.Edges, store.Edge{
					Src: relPath, Dst: dstRel, Kind: "imports",
				})
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
					// Record the func/method in the object index so pass 2 can resolve
					// callsites (both qualified `pkg.Foo()` and bare `Foo()`) back to
					// this ID. We index by (pkg.Path(), name) to handle cross-package
					// resolution where different TypesInfo instances produce different
					// *types.Func pointers for the same symbol.
					if pk.TypesInfo != nil {
						if obj := pk.TypesInfo.Defs[fd.Name]; obj != nil {
							key := objIndexKey(obj)
							if key != "" && objIndex[key] == "" {
								objIndex[key] = declID
							}
						}
					}
					// Queue body for pass-2 call-edge extraction. Skip nil bodies
					// (e.g. assembly stubs `func foo()` with no Go body).
					if fd.Body != nil {
						pending = append(pending, pendingCall{
							relPath: relPath, srcID: declID, pkg: pk, body: fd.Body,
							isTest: isGoTestFuncNative(fd, fname, pk),
						})
					}
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

					// Index the type-name object so the implements pass can look
					// up its node ID. Also bucket it for the implements pass
					// (interfaces vs concretes; aliases excluded since they're
					// not subjects of types.Implements).
					tn, isTN := obj.(*types.TypeName)
					if isTN && ts.Assign == token.NoPos {
						key := objIndexKey(obj)
						if key != "" && objIndex[key] == "" {
							objIndex[key] = declID
						}
						typeFile[tn] = relPath
						switch obj.Type().Underlying().(type) {
						case *types.Interface:
							interfaceTypes = append(interfaceTypes, tn)
						default:
							concreteTypes = append(concreteTypes, tn)
						}
					}
				}
			}
			results[relPath] = res
		}
	}

	if usable == 0 {
		return nil, fmt.Errorf("no usable packages loaded under %s (errors: %d)", abs, len(allErrors))
	}

	// Pass 2: walk each queued function body and emit `calls` edges to symbols
	// minted in pass 1. We resolve the callee via types.Info.Uses:
	//   - `Foo()`         -> CallExpr.Fun is *ast.Ident; Uses[ident]
	//   - `pkg.Foo()`     -> CallExpr.Fun is *ast.SelectorExpr; Uses[sel.Sel]
	//   - `recv.Method()` -> same SelectorExpr form; Uses[sel.Sel] yields *types.Func
	// Interface-method calls resolve to the *interface* method's *types.Func
	// (the static reference), not any concrete impl — the eventual `implements`
	// edge (N1.7) bridges interface methods to concrete implementations.
	// We silently skip edges when the callee is nil (unresolvable), a builtin
	// (obj.Pkg() == nil), outside this module, or not present in objIndex.
	//
	// For test functions (TestXxx(*testing.T) in *_test.go files), we additionally
	// emit `tests` edges with the same (Src, Dst) for each resolved call site.
	for _, pc := range pending {
		ti := pc.pkg.TypesInfo
		if ti == nil {
			continue
		}
		// Collect edges for this caller, then dedupe. The key is now (Dst, Kind)
		// so that we can emit both "calls" and "tests" edges for the same destination
		// from a test function, without duplication.
		seenEdge := map[string]map[string]bool{} // seenEdge[Dst][Kind]
		ast.Inspect(pc.body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			var ident *ast.Ident
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				ident = fn
			case *ast.SelectorExpr:
				ident = fn.Sel
			default:
				return true
			}
			if ident == nil {
				return true
			}
			obj := ti.Uses[ident]
			if obj == nil {
				return true
			}
			fnObj, ok := obj.(*types.Func)
			if !ok {
				return true
			}
			pkg := fnObj.Pkg()
			if pkg == nil { // builtin (len, append, new, ...)
				return true
			}
			if !inModule[pkg.Path()] {
				return true
			}
			key := objIndexKey(fnObj)
			dstID, ok := objIndex[key]
			if !ok {
				return true
			}
			res := results[pc.relPath]
			// Emit "calls" edge (always).
			if seenEdge[dstID] == nil {
				seenEdge[dstID] = map[string]bool{}
			}
			if !seenEdge[dstID]["calls"] {
				seenEdge[dstID]["calls"] = true
				res.Edges = append(res.Edges, store.Edge{
					Src: pc.srcID, Dst: dstID, Kind: "calls",
				})
			}
			// Emit "tests" edge if this is a test function.
			if pc.isTest && !seenEdge[dstID]["tests"] {
				seenEdge[dstID]["tests"] = true
				res.Edges = append(res.Edges, store.Edge{
					Src: pc.srcID, Dst: dstID, Kind: "tests",
				})
			}
			results[pc.relPath] = res
			return true
		})
	}

	// Pass 3: emit `implements` edges using types.Implements over the in-module
	// (T, I) cross-product. We treat T as implementing I if either the value
	// method set or the pointer method set satisfies I — matching the
	// convention used by go vet / gopls. Empty interfaces are skipped to avoid
	// emitting noise (every type "implements" any).
	sortTN := func(s []*types.TypeName) {
		sort.Slice(s, func(i, j int) bool {
			pi, pj := "", ""
			if p := s[i].Pkg(); p != nil {
				pi = p.Path()
			}
			if p := s[j].Pkg(); p != nil {
				pj = p.Path()
			}
			if pi != pj {
				return pi < pj
			}
			return s[i].Name() < s[j].Name()
		})
	}
	sortTN(concreteTypes)
	sortTN(interfaceTypes)
	for _, T := range concreteTypes {
		if T.Pkg() == nil || !inModule[T.Pkg().Path()] {
			continue
		}
		tKey := objIndexKey(T)
		srcID, ok := objIndex[tKey]
		if !ok {
			continue
		}
		relPath, ok := typeFile[T]
		if !ok {
			continue
		}
		Tt := T.Type()
		Tp := types.NewPointer(Tt)
		for _, I := range interfaceTypes {
			if I.Pkg() == nil || !inModule[I.Pkg().Path()] {
				continue
			}
			iface, ok := I.Type().Underlying().(*types.Interface)
			if !ok {
				continue
			}
			if iface.NumMethods() == 0 {
				continue
			}
			if !types.Implements(Tt, iface) && !types.Implements(Tp, iface) {
				continue
			}
			iKey := objIndexKey(I)
			dstID, ok := objIndex[iKey]
			if !ok {
				continue
			}
			res := results[relPath]
			res.Edges = append(res.Edges, store.Edge{
				Src: srcID, Dst: dstID, Kind: "implements",
			})
			results[relPath] = res
		}
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
