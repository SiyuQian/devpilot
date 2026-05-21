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

// ErrNoGoModule is the sentinel returned by loadGoModule when the repo has
// neither a go.mod nor a go.work. The native Go backend requires a module
// boundary; callers surface this as a hard build error. Exported so the CLI
// layer can use errors.Is to surface a distinct envelope code.
var ErrNoGoModule = errors.New("no go.mod or go.work")

// isGoOwnedPath reports whether p is a path the native Go backend takes
// ownership of (Go sources plus the module-shape files). Used by both the
// incremental reload trigger and the per-file fanout filter so the two stay
// in lockstep.
func isGoOwnedPath(p string) bool {
	if filepath.Ext(p) == ".go" {
		return true
	}
	switch p {
	case "go.mod", "go.sum", "go.work", "go.work.sum":
		return true
	}
	return false
}

// loadGoModule invokes the parser's PackageLoader path.
//
// Detection rules:
//   - repoRoot/go.work exists: parse the use directives via x/mod/modfile and
//     call LoadModule once per module directory, merging results. On key
//     collision the first-seen (sorted by use-directive resolved path) wins.
//   - repoRoot/go.mod exists: call LoadModule(repoRoot) once.
//   - neither: return errNoGoModule; callers escalate to a hard build error (no fallback parser).
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
		var workspaceErrs []parser.ParseError
		for _, up := range usePaths {
			sub, lerr := loader.LoadModule(up)
			if lerr != nil {
				// One empty use-module (e.g. go.mod with zero Go files) must not
				// brick the rest of the workspace. Record the per-module failure
				// and continue. Hard config errors (e.g. malformed go.mod) still
				// surface this way; callers can choose to escalate.
				workspaceErrs = append(workspaceErrs, parser.ParseError{
					Path:    up,
					Message: "load workspace module: " + lerr.Error(),
				})
				continue
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
		if len(workspaceErrs) > 0 {
			// Stitch workspace errors into the synthetic "" key alongside any
			// per-package errors LoadModule already collected there.
			existing := merged[""]
			existing.Errors = append(existing.Errors, workspaceErrs...)
			merged[""] = existing
		}
		return merged, nil
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
		return loader.LoadModule(repoRoot)
	}

	return nil, ErrNoGoModule
}
