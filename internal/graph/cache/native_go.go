package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/mod/modfile"

	"github.com/siyuqian/devpilot/internal/graph/parser"
)

// errNoGoModule is the sentinel returned by loadGoModule when the repo has
// neither a go.mod nor a go.work — callers should treat it as a non-error
// signal to fall back to per-file Parse (tree-sitter) for Go files.
var errNoGoModule = errors.New("no go.mod or go.work: falling back to per-file parse")

// loadGoModule invokes the parser's PackageLoader path.
//
// Detection rules:
//   - repoRoot/go.work exists: parse the use directives via x/mod/modfile and
//     call LoadModule once per module directory, merging results. On key
//     collision the first-seen (sorted by use-directive resolved path) wins.
//   - repoRoot/go.mod exists: call LoadModule(repoRoot) once.
//   - neither: return errNoGoModule so the caller falls back to tree-sitter.
func loadGoModule(loader parser.PackageLoader, repoRoot string) (map[string]parser.ParseResult, error) {
	workPath := filepath.Join(repoRoot, "go.work")
	if data, err := os.ReadFile(workPath); err == nil {
		wf, perr := modfile.ParseWork(workPath, data, nil)
		if perr != nil {
			return nil, fmt.Errorf("parse go.work: %w", perr)
		}
		// Resolve and sort use paths for deterministic merge order.
		usePaths := make([]string, 0, len(wf.Use))
		for _, u := range wf.Use {
			p := u.Path
			if !filepath.IsAbs(p) {
				p = filepath.Join(repoRoot, p)
			}
			abs, aerr := filepath.Abs(p)
			if aerr != nil {
				return nil, fmt.Errorf("abs %s: %w", p, aerr)
			}
			usePaths = append(usePaths, abs)
		}
		sort.Strings(usePaths)

		merged := map[string]parser.ParseResult{}
		for _, up := range usePaths {
			sub, lerr := loader.LoadModule(up)
			if lerr != nil {
				return nil, fmt.Errorf("load module %s: %w", up, lerr)
			}
			// Merge: first-seen key wins (sorted iteration over sub for determinism).
			keys := make([]string, 0, len(sub))
			for k := range sub {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if _, ok := merged[k]; ok {
					continue
				}
				merged[k] = sub[k]
			}
		}
		return merged, nil
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
		return loader.LoadModule(repoRoot)
	}

	return nil, errNoGoModule
}
