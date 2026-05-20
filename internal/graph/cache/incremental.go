package cache

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/resolver"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

// BuildIncremental applies a delta on top of an existing graph.db by diffing
// prev.HeadSHA against the current HEAD with `git diff --name-status`. It
// falls back to FullBuild when prev.HeadSHA is empty, the git invocation
// fails, or prev.HeadSHA is not an ancestor of HEAD (force-push, rebase).
func (b *Builder) BuildIncremental(prev Meta) (BuildResult, error) {
	currentHead := gitHeadSHA(b.repo)
	if prev.HeadSHA == "" || currentHead == "" || !isAncestor(b.repo, prev.HeadSHA, currentHead) {
		return b.FullBuild()
	}
	if prev.HeadSHA == currentHead {
		return BuildResult{Mode: "incremental"}, nil
	}

	rel, err := AcquireBuildLock(LockFile(b.home, b.key), 60*time.Second)
	if err != nil {
		return BuildResult{}, err
	}
	defer func() { _ = rel() }()

	changed, err := gitChangedFiles(b.repo, prev.HeadSHA, currentHead)
	if err != nil {
		return BuildResult{}, err
	}

	st, err := store.Open(GraphDB(b.home, b.key))
	if err != nil {
		return BuildResult{}, err
	}
	defer func() { _ = st.Close() }()

	// Wipe all nodes (and edges touching them) for modified+deleted files so
	// re-parsed results can be inserted cleanly. Added files have nothing to
	// delete.
	toDelete := append([]string{}, changed.Modified...)
	toDelete = append(toDelete, changed.Deleted...)
	if _, _, err := st.DeleteByPaths(toDelete); err != nil {
		return BuildResult{}, fmt.Errorf("delete stale paths: %w", err)
	}

	// Re-parse modified + added files; produces ParseResults for the new content.
	toParse := append([]string{}, changed.Modified...)
	toParse = append(toParse, changed.Added...)
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
			return BuildResult{}, err
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
		return BuildResult{}, err
	}
	if err := st.InsertEdges(newEdges); err != nil {
		return BuildResult{}, err
	}

	meta := Meta{
		SchemaVersion: CurrentSchemaVersion,
		HeadSHA:       currentHead,
		ParserVersion: parserVersionTag(b.reg),
		Languages:     b.reg.Languages(),
		BuiltAtUnix:   time.Now().Unix(),
	}
	if err := WriteMeta(MetaFile(b.home, b.key), meta); err != nil {
		return BuildResult{}, err
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

// gitChangedFiles parses `git diff --name-status from to` into added,
// modified, and deleted path slices. Renames are split into delete+add.
func gitChangedFiles(repo, from, to string) (changeSet, error) {
	out, err := exec.Command("git", "-C", repo, "diff", "--name-status", from, to).CombinedOutput()
	if err != nil {
		return changeSet{}, fmt.Errorf("git diff %s..%s: %w (%s)", from, to, err, out)
	}
	var cs changeSet
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		switch status[0] {
		case 'A':
			cs.Added = append(cs.Added, filepath.ToSlash(fields[len(fields)-1]))
		case 'M':
			cs.Modified = append(cs.Modified, filepath.ToSlash(fields[len(fields)-1]))
		case 'D':
			cs.Deleted = append(cs.Deleted, filepath.ToSlash(fields[len(fields)-1]))
		case 'R':
			if len(fields) == 3 {
				cs.Deleted = append(cs.Deleted, filepath.ToSlash(fields[1]))
				cs.Added = append(cs.Added, filepath.ToSlash(fields[2]))
			}
		}
	}
	return cs, nil
}

// isAncestor reports whether commit a is an ancestor of commit b in repo.
func isAncestor(repo, a, b string) bool {
	return exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", a, b).Run() == nil
}
