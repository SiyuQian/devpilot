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
	"golang.org/x/sync/errgroup"
)

// defaultMaxWorkers is the parser fanout used when Builder.MaxWorkers is 0.
const defaultMaxWorkers = 4

// Builder owns the cache directory for a single repo and produces graph.db.
type Builder struct {
	home string
	repo string
	key  string
	reg  *parser.Registry
	// MaxWorkers caps the number of concurrent parser goroutines used by
	// FullBuild. Zero means use defaultMaxWorkers. Output is deterministic
	// regardless of this value.
	MaxWorkers int
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
	// SQLite WAL mode leaves -wal and -shm sidecars; remove them with the db
	// so a fresh Open doesn't replay a stale WAL onto the empty database.
	for _, p := range []string{dbPath, dbPath + "-wal", dbPath + "-shm"} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return BuildResult{}, fmt.Errorf("remove %s: %w", p, err)
		}
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

	workers := b.MaxWorkers
	if workers <= 0 {
		workers = defaultMaxWorkers
	}

	// Indexed slice preserves files-order so output is deterministic
	// regardless of worker scheduling. Do NOT switch to a channel/map.
	results := make([]parser.ParseResult, len(files))
	g := new(errgroup.Group)
	g.SetLimit(workers)
	for i, relPath := range files {
		i, relPath := i, relPath
		g.Go(func() error {
			p := b.reg.ForPath(relPath)
			src, err := os.ReadFile(filepath.Join(b.repo, relPath))
			if err != nil {
				return fmt.Errorf("read %s: %w", relPath, err)
			}
			res, err := p.Parse(relPath, src)
			if err != nil {
				return fmt.Errorf("parse %s: %w", relPath, err)
			}
			results[i] = res
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return BuildResult{}, err
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

// Build dispatches to FullBuild or BuildIncremental based on cache state.
// Missing graph.db -> full. Schema-version mismatch -> wipe cache dir, full.
// Otherwise -> incremental (currently stubbed to full; see Task 2.21).
func (b *Builder) Build() (BuildResult, error) {
	_ = SweepPreflight(b.home, 7*24*time.Hour)
	if _, err := os.Stat(GraphDB(b.home, b.key)); os.IsNotExist(err) {
		return b.FullBuild()
	}
	m, err := ReadMeta(MetaFile(b.home, b.key))
	if err != nil {
		return BuildResult{}, err
	}
	if m.SchemaVersion != CurrentSchemaVersion {
		if err := os.RemoveAll(GraphDir(b.home, b.key)); err != nil {
			return BuildResult{}, fmt.Errorf("wipe cache dir: %w", err)
		}
		if err := EnsureDirs(b.home, b.key); err != nil {
			return BuildResult{}, err
		}
		return b.FullBuild()
	}
	return b.BuildIncremental(m)
}

func parserVersionTag(reg *parser.Registry) string {
	return "phase2:" + strings.Join(reg.Languages(), ",")
}
