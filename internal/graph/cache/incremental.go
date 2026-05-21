package cache

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/resolver"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

// BuildIncremental applies a delta on top of an existing graph.db by diffing
// prev.HeadSHA against the current HEAD with `git diff --name-status`. It
// falls back to FullBuild when prev.HeadSHA is empty, the git invocation
// fails, prev.HeadSHA is not an ancestor of HEAD (force-push, rebase), or
// the working tree has uncommitted changes (otherwise the cache would
// silently reflect the last-committed state instead of disk).
func (b *Builder) BuildIncremental(prev Meta) (BuildResult, error) {
	currentHead := gitHeadSHA(b.repo)
	if prev.HeadSHA == "" || currentHead == "" || !isAncestor(b.repo, prev.HeadSHA, currentHead) {
		return b.FullBuild()
	}
	if isDirty(b.repo) {
		return b.FullBuild()
	}
	if prev.HeadSHA == currentHead {
		return BuildResult{Mode: "incremental"}, nil
	}

	rel, err := AcquireBuildLock(LockFile(b.home, b.key), 60*time.Second)
	if err != nil {
		return BuildResult{}, fmt.Errorf("acquire build lock: %w", err)
	}
	defer func() { _ = rel() }()

	changed, err := gitChangedFiles(b.repo, prev.HeadSHA, currentHead)
	if err != nil {
		return BuildResult{}, fmt.Errorf("collect changed files: %w", err)
	}

	st, err := store.Open(GraphDB(b.home, b.key))
	if err != nil {
		return BuildResult{}, fmt.Errorf("open graph.db: %w", err)
	}
	defer func() { _ = st.Close() }()

	// Native Go has no file-level incremental path: go/types needs the whole
	// module. If a Go source file (or go.mod/go.sum) changed AND the registered
	// Go parser is a PackageLoader, re-run LoadModule for the entire module,
	// drop every Go-owned row, and inject the fresh native results into the
	// merge. Non-Go files still take the existing per-file path.
	goReload := false
	for _, list := range [][]string{changed.Added, changed.Modified, changed.Deleted} {
		for _, p := range list {
			if filepath.Ext(p) == ".go" || p == "go.mod" || p == "go.sum" {
				goReload = true
				break
			}
		}
		if goReload {
			break
		}
	}
	var nativeResults map[string]parser.ParseResult
	useNative := false
	if goReload {
		if goP := b.reg.ForLanguage("go"); goP != nil {
			if loader, ok := goP.(parser.PackageLoader); ok {
				res, lerr := loadGoModule(loader, b.repo)
				switch {
				case lerr == nil:
					useNative = true
					nativeResults = res
				case errors.Is(lerr, errNoGoModule):
					// Non-module repo: fall through to per-file path.
				default:
					return BuildResult{}, fmt.Errorf("native Go load: %w", lerr)
				}
			}
		}
	}

	// Wipe all nodes (and edges touching them) for modified+deleted files so
	// re-parsed results can be inserted cleanly. Added files have nothing to
	// delete. When the native Go path is active, strip *.go entries — the
	// language-wide DeleteByLanguage below covers them.
	toDelete := append([]string{}, changed.Modified...)
	toDelete = append(toDelete, changed.Deleted...)
	if useNative {
		toDelete = stripGoPaths(toDelete)
	}
	if _, _, err := st.DeleteByPaths(toDelete); err != nil {
		return BuildResult{}, fmt.Errorf("delete stale paths: %w", err)
	}
	if useNative {
		if _, _, err := st.DeleteByLanguage("go"); err != nil {
			return BuildResult{}, fmt.Errorf("delete go rows: %w", err)
		}
	}

	// Re-parse modified + added files; produces ParseResults for the new content.
	toParse := append([]string{}, changed.Modified...)
	toParse = append(toParse, changed.Added...)
	if useNative {
		toParse = stripGoPaths(toParse)
	}
	var newResults []parser.ParseResult
	for _, p := range toParse {
		par := b.reg.ForPath(p)
		if par == nil {
			continue
		}
		src, err := os.ReadFile(filepath.Join(b.repo, p))
		if err != nil {
			if os.IsNotExist(err) {
				continue // raced with delete
			}
			return BuildResult{}, fmt.Errorf("read %s: %w", p, err)
		}
		res, err := par.Parse(p, src)
		if err != nil {
			return BuildResult{}, fmt.Errorf("parse %s: %w", p, err)
		}
		newResults = append(newResults, res)
	}

	// Inject native Go results before the resolver merge. Sort by key first so
	// the ordering is deterministic and matches FullBuild.
	if useNative {
		keys := make([]string, 0, len(nativeResults))
		for k := range nativeResults {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		nativeSorted := make([]parser.ParseResult, 0, len(keys))
		for _, k := range keys {
			nativeSorted = append(nativeSorted, nativeResults[k])
		}
		newResults = append(nativeSorted, newResults...)
	}

	// To let the resolver see existing nodes from the database (so cross-file
	// references in newly parsed files can resolve), wrap the surviving nodes
	// (post-delete) into a synthetic ParseResult with no edges.
	existingNodes, err := st.AllNodes()
	if err != nil {
		return BuildResult{}, fmt.Errorf("load existing nodes: %w", err)
	}
	allResults := append([]parser.ParseResult{{Nodes: existingNodes}}, newResults...)
	allResults = resolver.Resolve(allResults)

	if _, err := os.Stat(filepath.Join(b.repo, "tsconfig.json")); err == nil {
		ts, err := resolver.NewTSConfigResolver(b.repo)
		if err != nil {
			return BuildResult{}, fmt.Errorf("load tsconfig: %w", err)
		}
		for i := range allResults {
			allResults[i].Edges = ts.Rewrite(allResults[i].Edges)
		}
	}

	// Insert nodes and edges from the newly parsed files only (existing nodes
	// already live in the store and were not deleted). allResults[0] is the
	// synthetic existing-nodes wrapper, so skip it.
	var newNodes []store.Node
	var newEdges []store.Edge
	for _, r := range allResults[1:] {
		newNodes = append(newNodes, r.Nodes...)
		newEdges = append(newEdges, r.Edges...)
	}
	// The resolver may also surface implements edges hanging off the synthetic
	// wrapper (where nodes belong to unchanged files but edges were freshly
	// computed). Include those too.
	newEdges = append(newEdges, allResults[0].Edges...)

	if err := st.InsertNodes(newNodes); err != nil {
		return BuildResult{}, fmt.Errorf("insert nodes: %w", err)
	}
	if err := st.InsertEdges(newEdges); err != nil {
		return BuildResult{}, fmt.Errorf("insert edges: %w", err)
	}

	meta := Meta{
		SchemaVersion: CurrentSchemaVersion,
		HeadSHA:       currentHead,
		ParserVersion: parserVersionTag(b.reg),
		Languages:     b.reg.Languages(),
		BuiltAtUnix:   time.Now().Unix(),
	}
	if err := WriteMeta(MetaFile(b.home, b.key), meta); err != nil {
		return BuildResult{}, fmt.Errorf("write meta: %w", err)
	}

	return BuildResult{
		FilesParsed: len(newResults),
		NodesInsert: len(newNodes),
		EdgesInsert: len(newEdges),
		Mode:        "incremental",
	}, nil
}

type changeSet struct {
	Added, Modified, Deleted []string
}

// gitChangedFiles parses `git diff --name-status -z from to` into added,
// modified, and deleted path slices. The -z flag NUL-separates records and
// status from path, which is the only way to handle paths containing spaces
// or newlines correctly. Renames (R) and copies (C) are split into delete+add.
// Typechanges (T) and unmerged (U) are treated as Modified so the file is
// re-parsed rather than silently skipped.
func gitChangedFiles(repo, from, to string) (changeSet, error) {
	// Output() separates stderr from stdout, so git warnings (e.g. CRLF
	// notices) don't pollute the diff record stream.
	cmd := exec.Command("git", "-C", repo, "diff", "--name-status", "-z", from, to)
	out, err := cmd.Output()
	if err != nil {
		return changeSet{}, fmt.Errorf("git diff %s..%s: %w", from, to, err)
	}
	// Each record is `<status>\0<path>` or, for R/C, `<status>\0<old>\0<new>`.
	tokens := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	var cs changeSet
	for i := 0; i < len(tokens); {
		status := tokens[i]
		if status == "" {
			i++
			continue
		}
		i++
		needNew := status[0] == 'R' || status[0] == 'C'
		if i >= len(tokens) {
			break
		}
		path := tokens[i]
		i++
		var newPath string
		if needNew {
			if i >= len(tokens) {
				break
			}
			newPath = tokens[i]
			i++
		}
		switch status[0] {
		case 'A':
			cs.Added = append(cs.Added, filepath.ToSlash(path))
		case 'D':
			cs.Deleted = append(cs.Deleted, filepath.ToSlash(path))
		case 'R', 'C':
			if status[0] == 'R' {
				cs.Deleted = append(cs.Deleted, filepath.ToSlash(path))
			}
			cs.Added = append(cs.Added, filepath.ToSlash(newPath))
		default:
			// M, T, U, and any unknown future status: treat as modified.
			cs.Modified = append(cs.Modified, filepath.ToSlash(path))
		}
	}
	return cs, nil
}

// stripGoPaths filters out *.go, go.mod, and go.sum entries. Used when the
// native Go path takes ownership of all Go-language rows so the per-file
// fanout doesn't double-process them.
func stripGoPaths(paths []string) []string {
	out := paths[:0:0]
	for _, p := range paths {
		if filepath.Ext(p) == ".go" || p == "go.mod" || p == "go.sum" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// isAncestor reports whether commit a is an ancestor of commit b in repo.
func isAncestor(repo, a, b string) bool {
	return exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", a, b).Run() == nil
}

// isDirty reports whether repo has uncommitted changes (staged or unstaged).
// Returns false when git is unavailable so non-git checkouts still incremental.
func isDirty(repo string) bool {
	out, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}
