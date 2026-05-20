package graph

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/envelope"
)

func TestResolveRepoAbs(t *testing.T) {
	got, err := resolveRepo(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("want absolute, got %q", got)
	}
}

func TestResolveRepoMissing(t *testing.T) {
	if _, err := resolveRepo("/definitely/not/here"); err == nil {
		t.Fatal("want error")
	}
}

// captureStdout returns whatever fn writes to os.Stdout.
func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan []byte)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.Bytes()
	}()
	fn()
	_ = w.Close()
	os.Stdout = old
	return <-done
}

func TestEmitValidates(t *testing.T) {
	e := envelope.New("graph.status").OK(map[string]any{
		"repo": "/x", "head_sha": "abc", "languages": []string{"go"},
		"nodes": 1, "edges": 1, "built_at_unix": 1,
	})
	buf := captureStdout(t, func() {
		if code := emit(e, "status.v1.json"); code != 0 {
			t.Errorf("rc=%d", code)
		}
	})
	if !bytes.Contains(buf, []byte(`"ok":true`)) {
		t.Errorf("missing ok=true: %s", buf)
	}
}
