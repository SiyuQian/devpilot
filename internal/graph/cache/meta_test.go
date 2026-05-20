package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMetaRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "meta.json")
	want := Meta{
		SchemaVersion: 1,
		HeadSHA:       "abc123",
		ParserVersion: "go=phase2,ts=phase2",
		Languages:     []string{"go", "typescript"},
		BuiltAtUnix:   1700000000,
	}
	if err := WriteMeta(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadMeta(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.HeadSHA != want.HeadSHA || got.SchemaVersion != want.SchemaVersion ||
		got.ParserVersion != want.ParserVersion || got.BuiltAtUnix != want.BuiltAtUnix ||
		len(got.Languages) != len(want.Languages) {
		t.Errorf("ReadMeta=%+v want %+v", got, want)
	}
	for i := range want.Languages {
		if got.Languages[i] != want.Languages[i] {
			t.Errorf("language[%d]=%q want %q", i, got.Languages[i], want.Languages[i])
		}
	}
}

func TestReadMetaMissingReturnsEmpty(t *testing.T) {
	got, err := ReadMeta("/nonexistent/meta.json")
	if err != nil {
		t.Fatal(err)
	}
	if got.HeadSHA != "" {
		t.Errorf("expected empty meta, got %+v", got)
	}
}

func TestWriteMetaAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "meta.json")
	if err := WriteMeta(path, Meta{HeadSHA: "x", SchemaVersion: 1}); err != nil {
		t.Fatal(err)
	}
	// No stray .tmp file should remain.
	matches, _ := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if len(matches) != 0 {
		t.Errorf("leftover tmp files: %v", matches)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
