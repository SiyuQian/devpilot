package review

import (
	"strings"
	"testing"
)

func TestChunkDiff_SmallDiff(t *testing.T) {
	diff := "diff --git a/foo.go b/foo.go\n+++ b/foo.go\n@@ -1,3 +1,4 @@\n+new line\n"
	chunks := ChunkDiff(diff)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Diff != diff {
		t.Error("single chunk should contain the full diff")
	}
	if len(chunks[0].Files) != 1 || chunks[0].Files[0] != "foo.go" {
		t.Errorf("expected files=[foo.go], got %v", chunks[0].Files)
	}
}

func TestChunkDiff_LargeDiff(t *testing.T) {
	// Create a diff with two files, each exceeding half the limit
	file1 := "diff --git a/a.go b/a.go\n+++ b/a.go\n@@ -1,1 +1,1 @@\n+" + strings.Repeat("x", 20000) + "\n"
	file2 := "diff --git a/b.go b/b.go\n+++ b/b.go\n@@ -1,1 +1,1 @@\n+" + strings.Repeat("y", 20000) + "\n"
	diff := file1 + file2

	chunks := ChunkDiff(diff)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks for large diff, got %d", len(chunks))
	}

	// Verify all files are represented
	allFiles := make(map[string]bool)
	for _, c := range chunks {
		for _, f := range c.Files {
			allFiles[f] = true
		}
	}
	if !allFiles["a.go"] || !allFiles["b.go"] {
		t.Errorf("expected files a.go and b.go, got %v", allFiles)
	}
}

func TestChunkDiff_ExactlyAtLimit(t *testing.T) {
	diff := strings.Repeat("x", MaxDiffChunkSize)
	chunks := ChunkDiff(diff)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk at exact limit, got %d", len(chunks))
	}
}

func TestFilesInDiff(t *testing.T) {
	diff := "+++ b/foo.go\n+++ b/bar/baz.go\n"
	files := FilesInDiff(diff)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0] != "foo.go" || files[1] != "bar/baz.go" {
		t.Errorf("got %v", files)
	}
}
