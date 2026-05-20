package cache

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestBuilderFullBuildOnTempRepo(t *testing.T) {
	repo := t.TempDir()
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
	if res.FilesParsed != 1 {
		t.Errorf("FilesParsed=%d want 1", res.FilesParsed)
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
