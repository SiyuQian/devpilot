package cache

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	_ "modernc.org/sqlite"
)

// writeGoMod drops a minimal go.mod into dir so the native Go backend can
// type-check the fixture. The native backend rejects non-module repos.
func writeGoMod(t *testing.T, dir, module string) {
	t.Helper()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module "+module+"\n\ngo 1.22\n")
}

func TestBuildSweepsStalePreflight(t *testing.T) {
	home := t.TempDir()
	preDir := filepath.Join(home, "preflight")
	if err := os.MkdirAll(preDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(preDir, "stale.json")
	if err := os.WriteFile(stale, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	repo := t.TempDir()
	writeGoMod(t, repo, "example.com/sweep")
	mustWrite(t, filepath.Join(repo, "main.go"), "package main\nfunc main() {}\n")

	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Build(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale preflight not swept: err=%v", err)
	}
}

func TestBuilderFullBuildOnTempRepo(t *testing.T) {
	repo := t.TempDir()
	writeGoMod(t, repo, "example.com/temp")
	mustWrite(t, filepath.Join(repo, "main.go"), `package main
func Greet(n string) string { return "hi " + n }
func main() { Greet("x") }
`)
	mustWrite(t, filepath.Join(repo, "ignored.png"), "binary")

	home := t.TempDir()
	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	res, err := b.FullBuild()
	if err != nil {
		t.Fatal(err)
	}
	// Native backend owns Go files via LoadModule, so the per-file fanout
	// only sees zero remaining files (main.go is filtered out).
	if res.NodesInsert == 0 {
		t.Errorf("NodesInsert=%d want > 0", res.NodesInsert)
	}
	if _, err := os.Stat(GraphDB(home, RepoKey(repo))); err != nil {
		t.Errorf("graph.db missing: %v", err)
	}
	meta, err := ReadMeta(MetaFile(home, RepoKey(repo)))
	if err != nil {
		t.Fatal(err)
	}
	if meta.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("meta schema=%d want %d", meta.SchemaVersion, CurrentSchemaVersion)
	}
}

func TestBuilderFullBuildDeterministic(t *testing.T) {
	repo := t.TempDir()
	writeGoMod(t, repo, "example.com/det")
	mustWrite(t, filepath.Join(repo, "main.go"), `package main
func A() {}
func B() { A() }
`)
	home := t.TempDir()
	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.FullBuild(); err != nil {
		t.Fatal(err)
	}
	d1 := dumpDB(t, GraphDB(home, RepoKey(repo)))
	if _, err := b.FullBuild(); err != nil {
		t.Fatal(err)
	}
	d2 := dumpDB(t, GraphDB(home, RepoKey(repo)))
	if d1 != d2 {
		t.Errorf("two full builds produced different dumps:\n%s\n----\n%s", d1, d2)
	}
}

func TestBuilderParallelMatchesSequential(t *testing.T) {
	repo := t.TempDir()
	writeGoMod(t, repo, "example.com/par")
	for i := 0; i < 20; i++ {
		name := filepath.Join(repo, "pkg", "file"+itoa(i)+".go")
		mustWrite(t, name, "package pkg\nfunc F"+itoa(i)+"() {}\n")
	}

	home1 := t.TempDir()
	b1, err := NewBuilder(home1, repo)
	if err != nil {
		t.Fatal(err)
	}
	b1.MaxWorkers = 1
	if _, err := b1.FullBuild(); err != nil {
		t.Fatal(err)
	}
	d1 := dumpDB(t, GraphDB(home1, RepoKey(repo)))

	home8 := t.TempDir()
	b8, err := NewBuilder(home8, repo)
	if err != nil {
		t.Fatal(err)
	}
	b8.MaxWorkers = 8
	if _, err := b8.FullBuild(); err != nil {
		t.Fatal(err)
	}
	d8 := dumpDB(t, GraphDB(home8, RepoKey(repo)))

	if d1 != d8 {
		t.Errorf("parallel build diverged from sequential:\n--- workers=1 ---\n%s\n--- workers=8 ---\n%s", d1, d8)
	}
}

func TestSchemaMismatchRebuilds(t *testing.T) {
	repo := t.TempDir()
	writeGoMod(t, repo, "example.com/schema")
	mustWrite(t, filepath.Join(repo, "main.go"), `package main
func Greet(n string) string { return "hi " + n }
func main() { Greet("x") }
`)
	home := t.TempDir()
	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.FullBuild(); err != nil {
		t.Fatal(err)
	}

	// Poison the on-disk schema version.
	metaPath := MetaFile(home, RepoKey(repo))
	m, err := ReadMeta(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	m.SchemaVersion = 0
	if err := WriteMeta(metaPath, m); err != nil {
		t.Fatal(err)
	}

	res, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != "full" {
		t.Errorf("Mode=%q want full", res.Mode)
	}
	got, err := ReadMeta(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("meta schema=%d want %d", got.SchemaVersion, CurrentSchemaVersion)
	}
	if _, err := os.Stat(GraphDB(home, RepoKey(repo))); err != nil {
		t.Errorf("graph.db missing after rebuild: %v", err)
	}
}

func TestParserVersionMismatchRebuilds(t *testing.T) {
	repo := t.TempDir()
	writeGoMod(t, repo, "example.com/parserversion")
	mustWrite(t, filepath.Join(repo, "main.go"), `package main
func Greet(n string) string { return "hi " + n }
func main() { Greet("x") }
`)
	home := t.TempDir()
	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.FullBuild(); err != nil {
		t.Fatal(err)
	}

	// Poison the on-disk parser version while keeping schema version valid.
	metaPath := MetaFile(home, RepoKey(repo))
	m, err := ReadMeta(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	m.ParserVersion = "phase2:go=imaginary-backend"
	if err := WriteMeta(metaPath, m); err != nil {
		t.Fatal(err)
	}

	res, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != "full" {
		t.Errorf("Mode=%q want full (parser-version mismatch should trigger rebuild)", res.Mode)
	}
	got, err := ReadMeta(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.ParserVersion == "phase2:go=imaginary-backend" {
		t.Errorf("parser version was not refreshed: %q", got.ParserVersion)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParserVersionTagEncodesGoEnv(t *testing.T) {
	// Native backend output depends on GOOS/GOARCH/CGO_ENABLED/GOFLAGS;
	// changing any of these without touching files must still trigger a
	// rebuild. parserVersionTag captures this in an env= suffix.
	t.Setenv("CGO_ENABLED", "1")
	t.Setenv("GOFLAGS", "")
	tag1 := parserVersionTag(parser.DefaultRegistry())

	t.Setenv("CGO_ENABLED", "0")
	tag2 := parserVersionTag(parser.DefaultRegistry())

	if tag1 == tag2 {
		t.Errorf("CGO_ENABLED change must alter the tag; both=%q", tag1)
	}

	t.Setenv("CGO_ENABLED", "1")
	t.Setenv("GOFLAGS", "-tags=integration")
	tag3 := parserVersionTag(parser.DefaultRegistry())
	if tag1 == tag3 {
		t.Errorf("GOFLAGS change must alter the tag; both=%q", tag1)
	}
}

func TestParserVersionTagEncodesNativeBackend(t *testing.T) {
	tag := parserVersionTag(parser.DefaultRegistry())
	if !strings.Contains(tag, "go=native") {
		t.Errorf("parserVersionTag() = %q, want to contain %q", tag, "go=native")
	}
}

// copyGoNativeFixture materialises a small two-package Go module fixture
// into dst. Lives here (rather than referencing the parser package's
// testdata) so the cache tests own their fixtures.
func copyGoNativeFixture(t *testing.T, dst string) {
	t.Helper()
	files := map[string]string{
		"go.mod": "module example.com/native\n\ngo 1.22\n",
		"pkg/a/a.go": `package a

func Greet(name string) string {
	return "hi " + name
}

func Run() string {
	return Greet("world")
}
`,
		"pkg/b/b.go": `package b

import "example.com/native/pkg/a"

func B() string {
	return a.Greet("y")
}
`,
	}
	for rel, content := range files {
		mustWrite(t, filepath.Join(dst, rel), content)
	}
}

func TestBuilderFullBuildNativeGoBackend(t *testing.T) {
	repo := t.TempDir()
	copyGoNativeFixture(t, repo)

	home := t.TempDir()
	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	res, err := b.FullBuild()
	if err != nil {
		t.Fatal(err)
	}
	if res.NodesInsert == 0 {
		t.Fatalf("NodesInsert=0, want > 0")
	}

	db, err := sql.Open("sqlite", GraphDB(home, RepoKey(repo)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM nodes WHERE id = ?`, "pkg/a/a.go::Greet").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected node pkg/a/a.go::Greet, got count=%d", n)
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE src = ? AND dst = ? AND kind = ?`,
		"pkg/b/b.go::B", "pkg/a/a.go::Greet", "calls").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected calls edge pkg/b/b.go::B -> pkg/a/a.go::Greet, got count=%d", n)
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM edges WHERE dst LIKE 'external::%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("expected zero external:: edges, got %d", n)
	}
}

func TestBuilderFullBuildNativeGoDeterministic(t *testing.T) {
	repo := t.TempDir()
	copyGoNativeFixture(t, repo)

	home := t.TempDir()
	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.FullBuild(); err != nil {
		t.Fatal(err)
	}
	d1 := dumpDB(t, GraphDB(home, RepoKey(repo)))
	if _, err := b.FullBuild(); err != nil {
		t.Fatal(err)
	}
	d2 := dumpDB(t, GraphDB(home, RepoKey(repo)))
	if d1 != d2 {
		t.Errorf("two native-backend full builds produced different dumps:\n%s\n----\n%s", d1, d2)
	}
}

func TestBuilderFullBuildNonGoModuleErrors(t *testing.T) {
	repo := t.TempDir()
	// No go.mod, no go.work — the native backend must surface this as a
	// hard error rather than silently dropping the .go file.
	mustWrite(t, filepath.Join(repo, "main.go"), `package main
func Greet(n string) string { return "hi " + n }
func main() { Greet("x") }
`)

	home := t.TempDir()
	b, err := NewBuilder(home, repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.FullBuild(); err == nil {
		t.Fatal("FullBuild succeeded on non-module Go repo; expected hard error")
	}
}

func dumpDB(t *testing.T, path string) string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var lines []string
	rows, err := db.Query(`SELECT id, kind, path, name, container, language FROM nodes`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id, kind, p, name, lang string
		var container sql.NullString
		if err := rows.Scan(&id, &kind, &p, &name, &container, &lang); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, "N "+id+"|"+kind+"|"+p+"|"+name+"|"+container.String+"|"+lang)
	}
	_ = rows.Close()
	rows, err = db.Query(`SELECT src, dst, kind FROM edges`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var s, d, k string
		if err := rows.Scan(&s, &d, &k); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, "E "+s+"|"+d+"|"+k)
	}
	_ = rows.Close()
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}
