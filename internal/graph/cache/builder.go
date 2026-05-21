package cache

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/resolver"
	"github.com/siyuqian/devpilot/internal/graph/store"
	"golang.org/x/sync/errgroup"
)

// defaultMaxWorkers is the parser fanout used when Builder.MaxWorkers is 0.
const defaultMaxWorkers = 4

// buildLockTimeout returns the build-lock acquisition deadline. The native
// Go backend holds the lock for the entire packages.Load duration, which is
// seconds-to-minutes on large modules; bump the timeout when it's selected
// so concurrent invocations (e.g. preflight during an interactive build) get
// a fair chance to wait instead of timing out at 60s.
func buildLockTimeout(reg *parser.Registry) time.Duration {
	if reg != nil && reg.GoBackend() == "native" {
		return 5 * time.Minute
	}
	return 60 * time.Second
}

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
	rel, err := AcquireBuildLock(LockFile(b.home, b.key), buildLockTimeout(b.reg))
	if err != nil {
		return BuildResult{}, fmt.Errorf("acquire build lock: %w", err)
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

	// Native Go backend dispatch: when the registered Go parser implements
	// PackageLoader (the native backend), call LoadModule once per module and
	// strip Go files from the per-file fanout. On non-module repos we silently
	// fall back to per-file Parse so the build doesn't crash on stray inputs.
	var nativeResults map[string]parser.ParseResult
	var nativeErrs []parser.ParseError
	useNative := false
	if goP := b.reg.ForLanguage("go"); goP != nil {
		if loader, ok := goP.(parser.PackageLoader); ok {
			res, lerr := loadGoModule(loader, b.repo)
			switch {
			case lerr == nil:
				useNative = true
				nativeResults = res
			case errors.Is(lerr, errNoGoModule):
				// Non-module repo: record a synthetic ParseError so the signal
				// surfaces in results, but keep going via the tree-sitter fanout.
				// Use a distinct sentinel path so this is not conflated with
				// go_native.go's own synthetic errors (which key on "").
				nativeErrs = append(nativeErrs, parser.ParseError{
					Path:    nonModuleErrorPath,
					Message: "native Go backend skipped: " + lerr.Error(),
				})
			default:
				return BuildResult{}, fmt.Errorf("native Go load: %w", lerr)
			}
		}
	}
	// fallbackGo is used for Go files when useNative is false but the registered
	// Go parser is the native (no-op Parse) backend — i.e. on non-module repos.
	// We route Go files through a tree-sitter parser so the build still produces
	// nodes. nil means "use the registry parser as-is".
	var fallbackGo parser.Parser
	if useNative {
		nonGo := files[:0:0]
		for _, p := range files {
			if filepath.Ext(p) != ".go" {
				nonGo = append(nonGo, p)
			}
		}
		files = nonGo
	} else if len(nativeErrs) > 0 {
		fallbackGo = parser.NewGoParser()
	}

	st, err := store.Open(dbPath)
	if err != nil {
		return BuildResult{}, fmt.Errorf("open graph.db: %w", err)
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
			if fallbackGo != nil && filepath.Ext(relPath) == ".go" {
				p = fallbackGo
			}
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
		return BuildResult{}, fmt.Errorf("parse fanout: %w", err)
	}

	// Merge native Go results (sorted by key) ahead of the non-Go per-file
	// results so the final ordering is deterministic. The non-Go results are
	// already in files-order from the fanout slice.
	if useNative || len(nativeErrs) > 0 {
		keys := make([]string, 0, len(nativeResults))
		for k := range nativeResults {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		merged := make([]parser.ParseResult, 0, len(keys)+len(results))
		for _, k := range keys {
			merged = append(merged, nativeResults[k])
		}
		if len(nativeErrs) > 0 {
			merged = append(merged, parser.ParseResult{Errors: nativeErrs})
		}
		merged = append(merged, results...)
		results = merged
	}

	results = resolver.Resolve(results)

	if _, err := os.Stat(filepath.Join(b.repo, "tsconfig.json")); err == nil {
		ts, err := resolver.NewTSConfigResolver(b.repo)
		if err != nil {
			return BuildResult{}, fmt.Errorf("load tsconfig: %w", err)
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
		return BuildResult{}, fmt.Errorf("insert nodes: %w", err)
	}
	if err := st.InsertEdges(allEdges); err != nil {
		return BuildResult{}, fmt.Errorf("insert edges: %w", err)
	}

	meta := Meta{
		SchemaVersion: CurrentSchemaVersion,
		HeadSHA:       gitHeadSHA(b.repo),
		ParserVersion: parserVersionTag(b.reg),
		Languages:     b.reg.Languages(),
		BuiltAtUnix:   time.Now().Unix(),
	}
	if err := WriteMeta(MetaFile(b.home, b.key), meta); err != nil {
		return BuildResult{}, fmt.Errorf("write meta: %w", err)
	}

	return BuildResult{
		FilesParsed: len(files),
		NodesInsert: len(allNodes),
		EdgesInsert: len(allEdges),
		Mode:        "full",
	}, nil
}

// Build dispatches to FullBuild or BuildIncremental based on cache state.
// Missing graph.db -> full. Schema-version OR parser-version mismatch -> wipe
// cache dir and full-build. Otherwise -> incremental.
func (b *Builder) Build() (BuildResult, error) {
	_ = SweepPreflight(b.home, 7*24*time.Hour)
	if _, err := os.Stat(GraphDB(b.home, b.key)); os.IsNotExist(err) {
		return b.FullBuild()
	}
	m, err := ReadMeta(MetaFile(b.home, b.key))
	if err != nil {
		return BuildResult{}, fmt.Errorf("read meta: %w", err)
	}
	if m.SchemaVersion != CurrentSchemaVersion || m.ParserVersion != parserVersionTag(b.reg) {
		if err := os.RemoveAll(GraphDir(b.home, b.key)); err != nil {
			return BuildResult{}, fmt.Errorf("wipe cache dir: %w", err)
		}
		if err := EnsureDirs(b.home, b.key); err != nil {
			return BuildResult{}, fmt.Errorf("ensure cache dirs: %w", err)
		}
		return b.FullBuild()
	}
	return b.BuildIncremental(m)
}

func parserVersionTag(reg *parser.Registry) string {
	langs := reg.Languages()
	parts := make([]string, len(langs))
	for i, lang := range langs {
		if lang == "go" {
			parts[i] = "go=" + reg.GoBackend()
		} else {
			parts[i] = lang
		}
	}
	tag := "phase2:" + strings.Join(parts, ",")
	// The native Go backend's output depends on the ambient Go toolchain:
	// packages.Load honors GOOS/GOARCH/CGO_ENABLED/GOFLAGS, so a graph built
	// on macOS sees *_darwin.go and a Linux CI run sees *_linux.go. Encode
	// those selectors in the tag so Build() rebuilds when the toolchain shape
	// changes, even if no file changed.
	if reg.GoBackend() == "native" {
		tag += ";env=" + goToolchainEnvSignature()
	}
	return tag
}

// goToolchainEnvSignature captures the four env selectors that materially
// alter packages.Load output. Kept short and deterministic so it round-trips
// through meta.json.
func goToolchainEnvSignature() string {
	return fmt.Sprintf("goos=%s,goarch=%s,cgo=%s,goflags=%s",
		runtime.GOOS, runtime.GOARCH,
		os.Getenv("CGO_ENABLED"), os.Getenv("GOFLAGS"))
}
