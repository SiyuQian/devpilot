package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/resolver"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

// Builder owns the cache directory for a single repo and produces graph.db.
type Builder struct {
	home string
	repo string
	key  string
	reg  *parser.Registry
}

// NewBuilder validates the repo path and constructs a Builder.
func NewBuilder(home, repo string) (*Builder, error) {
	abs, err := filepath.Abs(repo)
	if err != nil {
		return nil, fmt.Errorf("abs(%s): %w", repo, err)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("repo %s is not a directory", abs)
	}
	key := RepoKey(abs)
	if err := EnsureDirs(home, key); err != nil {
		return nil, err
	}
	return &Builder{home: home, repo: abs, key: key, reg: parser.DefaultRegistry()}, nil
}

// BuildResult summarises a build for callers and tests.
type BuildResult struct {
	FilesParsed int
	NodesInsert int
	EdgesInsert int
	Mode        string // "full" | "incremental"
}

// FullBuild parses every supported file in the repo, runs the resolver,
// inserts into graph.db, and writes meta.json. It deletes any prior graph.db
// first so two consecutive calls produce identical output.
func (b *Builder) FullBuild() (BuildResult, error) {
	rel, err := AcquireBuildLock(LockFile(b.home, b.key), 60*time.Second)
	if err != nil {
		return BuildResult{}, err
	}
	defer func() { _ = rel() }()

	dbPath := GraphDB(b.home, b.key)
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return BuildResult{}, fmt.Errorf("remove %s: %w", dbPath, err)
	}

	files, err := WalkRepo(b.repo)
	if err != nil {
		return BuildResult{}, fmt.Errorf("walk %s: %w", b.repo, err)
	}
	files = FilterByParser(files, func(p string) bool {
		return b.reg.ForPath(p) != nil
	})

	st, err := store.Open(dbPath)
	if err != nil {
		return BuildResult{}, err
	}
	defer func() { _ = st.Close() }()

	results := make([]parser.ParseResult, 0, len(files))
	for _, relPath := range files {
		p := b.reg.ForPath(relPath)
		src, err := os.ReadFile(filepath.Join(b.repo, relPath))
		if err != nil {
			return BuildResult{}, fmt.Errorf("read %s: %w", relPath, err)
		}
		res, err := p.Parse(relPath, src)
		if err != nil {
			return BuildResult{}, fmt.Errorf("parse %s: %w", relPath, err)
		}
		results = append(results, res)
	}

	results = resolver.Resolve(results)

	if _, err := os.Stat(filepath.Join(b.repo, "tsconfig.json")); err == nil {
		ts, err := resolver.NewTSConfigResolver(b.repo)
		if err != nil {
			return BuildResult{}, err
		}
		for i := range results {
			results[i].Edges = ts.Rewrite(results[i].Edges)
		}
	}

	var allNodes []store.Node
	var allEdges []store.Edge
	for _, r := range results {
		allNodes = append(allNodes, r.Nodes...)
		allEdges = append(allEdges, r.Edges...)
	}

	if err := st.InsertNodes(allNodes); err != nil {
		return BuildResult{}, err
	}
	if err := st.InsertEdges(allEdges); err != nil {
		return BuildResult{}, err
	}

	meta := Meta{
		SchemaVersion: CurrentSchemaVersion,
		HeadSHA:       gitHeadSHA(b.repo),
		ParserVersion: parserVersionTag(b.reg),
		Languages:     b.reg.Languages(),
		BuiltAtUnix:   time.Now().Unix(),
	}
	if err := WriteMeta(MetaFile(b.home, b.key), meta); err != nil {
		return BuildResult{}, err
	}

	return BuildResult{
		FilesParsed: len(files),
		NodesInsert: len(allNodes),
		EdgesInsert: len(allEdges),
		Mode:        "full",
	}, nil
}

func parserVersionTag(reg *parser.Registry) string {
	return "phase2:" + strings.Join(reg.Languages(), ",")
}
