package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/cache"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

// seedQueryStore creates a fresh DEVPILOT_HOME, a repo dir, and a seeded
// graph.db at the cache path that runQuery will resolve.
//
// Fixture topology:
//
//	caller  -> target   (calls)
//	target  -> leaf     (calls)
//	tester  -> target   (tests)
//	impl    -> iface    (implements)
//	hub_a   -> hub      (calls)
//	hub_b   -> hub      (calls)
//	hub_c   -> hub      (calls)
//
// target/caller live in src.go with known line ranges so context can read them.
func seedQueryStore(t *testing.T) (repo string) {
	t.Helper()

	home := t.TempDir()
	repo = t.TempDir()
	t.Setenv("DEVPILOT_HOME", home)

	// On-disk source for the "context" pattern: lines 1..6.
	src := "package p\n" + // 1
		"func Target() {\n" + // 2
		"\tprintln(\"t\")\n" + // 3
		"}\n" + // 4
		"func Caller() {\n" + // 5
		"\tTarget()\n" + // 6
		"}\n" // 7
	if err := os.WriteFile(filepath.Join(repo, "src.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	key := cache.RepoKey(repo)
	if err := cache.EnsureDirs(home, key); err != nil {
		t.Fatal(err)
	}
	dbPath := cache.GraphDB(home, key)
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	nodes := []store.Node{
		{ID: "target", Kind: "function", Path: "src.go", Name: "Target", Language: "go", StartLine: 2, EndLine: 4},
		{ID: "caller", Kind: "function", Path: "src.go", Name: "Caller", Language: "go", StartLine: 5, EndLine: 7},
		{ID: "leaf", Kind: "function", Path: "src.go", Name: "Leaf", Language: "go"},
		{ID: "tester", Kind: "function", Path: "src_test.go", Name: "TestTarget", Language: "go"},
		{ID: "iface", Kind: "interface", Path: "src.go", Name: "Iface", Language: "go"},
		{ID: "impl", Kind: "struct", Path: "src.go", Name: "Impl", Language: "go"},
		{ID: "hub", Kind: "function", Path: "h.go", Name: "Hub", Language: "go"},
		{ID: "hub_a", Kind: "function", Path: "h.go", Name: "A", Language: "go"},
		{ID: "hub_b", Kind: "function", Path: "h.go", Name: "B", Language: "go"},
		{ID: "hub_c", Kind: "function", Path: "h.go", Name: "C", Language: "go"},
	}
	edges := []store.Edge{
		{Src: "caller", Dst: "target", Kind: "calls"},
		{Src: "target", Dst: "leaf", Kind: "calls"},
		{Src: "tester", Dst: "target", Kind: "tests"},
		{Src: "impl", Dst: "iface", Kind: "implements"},
		{Src: "hub_a", Dst: "hub", Kind: "calls"},
		{Src: "hub_b", Dst: "hub", Kind: "calls"},
		{Src: "hub_c", Dst: "hub", Kind: "calls"},
	}
	if err := st.InsertNodes(nodes); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertEdges(edges); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	return repo
}

// runAndParse calls runQuery, captures stdout, and decodes the envelope.
func runAndParse(t *testing.T, repo, pattern, target string, opts queryOpts) (rc int, env map[string]any) {
	t.Helper()
	opts.repo = repo
	buf := captureStdout(t, func() {
		rc = runQuery(opts, pattern, target)
	})
	if err := json.Unmarshal(buf, &env); err != nil {
		t.Fatalf("decode envelope (rc=%d): %v\n%s", rc, err, buf)
	}
	return rc, env
}

func okData(t *testing.T, env map[string]any, wantPattern string) (data map[string]any, payload map[string]any) {
	t.Helper()
	if env["ok"] != true {
		t.Fatalf("ok=false envelope: %v", env)
	}
	if env["command"] != "graph.query" {
		t.Errorf("command=%v want graph.query", env["command"])
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatalf("data not object: %v", env["data"])
	}
	if data["pattern"] != wantPattern {
		t.Errorf("pattern=%v want %v", data["pattern"], wantPattern)
	}
	payload, ok = data["pattern_result"].(map[string]any)
	if !ok {
		t.Fatalf("pattern_result not object: %v", data["pattern_result"])
	}
	return data, payload
}

func TestRunQueryPatterns(t *testing.T) {
	repo := seedQueryStore(t)

	t.Run("callers_of", func(t *testing.T) {
		rc, env := runAndParse(t, repo, "callers_of", "target", queryOpts{depth: 2})
		if rc != 0 {
			t.Fatalf("rc=%d env=%v", rc, env)
		}
		_, payload := okData(t, env, "callers_of")
		list, ok := payload["callers"].([]any)
		if !ok {
			t.Fatalf("callers not array: %v", payload)
		}
		if len(list) != 1 {
			t.Fatalf("want 1 caller, got %v", list)
		}
		entry := list[0].(map[string]any)
		if entry["id"] != "caller" {
			t.Errorf("id=%v", entry["id"])
		}
		if _, ok := entry["hop"]; !ok {
			t.Errorf("missing hop field: %v", entry)
		}
	})

	t.Run("callees_of", func(t *testing.T) {
		rc, env := runAndParse(t, repo, "callees_of", "target", queryOpts{depth: 2})
		if rc != 0 {
			t.Fatalf("rc=%d", rc)
		}
		_, payload := okData(t, env, "callees_of")
		list, ok := payload["callees"].([]any)
		if !ok {
			t.Fatalf("callees not array: %v", payload)
		}
		if len(list) != 1 {
			t.Fatalf("want 1 callee, got %v", list)
		}
		entry := list[0].(map[string]any)
		if entry["id"] != "leaf" {
			t.Errorf("id=%v", entry["id"])
		}
		if _, ok := entry["hop"]; !ok {
			t.Errorf("missing hop field: %v", entry)
		}
	})

	t.Run("tests_for", func(t *testing.T) {
		rc, env := runAndParse(t, repo, "tests_for", "target", queryOpts{})
		if rc != 0 {
			t.Fatalf("rc=%d", rc)
		}
		_, payload := okData(t, env, "tests_for")
		list, ok := payload["tests"].([]any)
		if !ok {
			t.Fatalf("tests not array: %v", payload)
		}
		if len(list) != 1 || list[0] != "tester" {
			t.Errorf("tests=%v want [tester]", list)
		}
	})

	t.Run("implementors_of", func(t *testing.T) {
		rc, env := runAndParse(t, repo, "implementors_of", "iface", queryOpts{})
		if rc != 0 {
			t.Fatalf("rc=%d", rc)
		}
		_, payload := okData(t, env, "implementors_of")
		list, ok := payload["implementors"].([]any)
		if !ok {
			t.Fatalf("implementors not array: %v", payload)
		}
		if len(list) != 1 || list[0] != "impl" {
			t.Errorf("implementors=%v want [impl]", list)
		}
	})

	t.Run("hubs", func(t *testing.T) {
		rc, env := runAndParse(t, repo, "hubs", "", queryOpts{threshold: 3})
		if rc != 0 {
			t.Fatalf("rc=%d", rc)
		}
		_, payload := okData(t, env, "hubs")
		list, ok := payload["hubs"].([]any)
		if !ok {
			t.Fatalf("hubs not array: %v", payload)
		}
		if len(list) != 1 {
			t.Fatalf("want 1 hub, got %v", list)
		}
		entry := list[0].(map[string]any)
		if entry["id"] != "hub" {
			t.Errorf("id=%v", entry["id"])
		}
		// caller_count is the canonical key consumed by skills.
		cc, ok := entry["caller_count"]
		if !ok {
			t.Fatalf("missing caller_count: %v", entry)
		}
		// JSON numbers decode as float64.
		if n, _ := cc.(float64); n != 3 {
			t.Errorf("caller_count=%v want 3", cc)
		}
	})

	t.Run("context", func(t *testing.T) {
		rc, env := runAndParse(t, repo, "context", "target", queryOpts{depth: 1})
		if rc != 0 {
			t.Fatalf("rc=%d env=%v", rc, env)
		}
		_, payload := okData(t, env, "context")
		ctx, ok := payload["context"].(map[string]any)
		if !ok {
			t.Fatalf("context not object: %v", payload)
		}
		tgt, ok := ctx["target"].(map[string]any)
		if !ok {
			t.Fatalf("target not object: %v", ctx)
		}
		for _, k := range []string{"id", "path", "start_line", "end_line", "source"} {
			if _, ok := tgt[k]; !ok {
				t.Errorf("target missing %q: %v", k, tgt)
			}
		}
		if tgt["id"] != "target" {
			t.Errorf("target.id=%v", tgt["id"])
		}
		callers, ok := ctx["callers"].([]any)
		if !ok {
			t.Fatalf("callers not array: %v", ctx)
		}
		if len(callers) != 1 {
			t.Fatalf("want 1 caller snippet, got %v", callers)
		}
		c0 := callers[0].(map[string]any)
		if c0["id"] != "caller" {
			t.Errorf("caller.id=%v", c0["id"])
		}
		for _, k := range []string{"id", "path", "start_line", "end_line", "source"} {
			if _, ok := c0[k]; !ok {
				t.Errorf("caller snippet missing %q: %v", k, c0)
			}
		}
	})
}

func TestRunQueryUnknownPattern(t *testing.T) {
	repo := seedQueryStore(t)
	rc, env := runAndParse(t, repo, "no_such_pattern", "x", queryOpts{})
	if rc == 0 {
		t.Errorf("want non-zero rc for unknown pattern")
	}
	if env["ok"] != false {
		t.Errorf("want ok=false, got %v", env["ok"])
	}
	errObj, ok := env["error"].(map[string]any)
	if !ok {
		t.Fatalf("error not object: %v", env["error"])
	}
	if errObj["code"] != "unknown_pattern" {
		t.Errorf("code=%v want unknown_pattern", errObj["code"])
	}
}

func TestRunQueryCacheMissing(t *testing.T) {
	// Fresh DEVPILOT_HOME, repo exists but no graph.db.
	t.Setenv("DEVPILOT_HOME", t.TempDir())
	repo := t.TempDir()
	rc, env := runAndParse(t, repo, "callers_of", "x", queryOpts{})
	if rc == 0 {
		t.Errorf("want non-zero rc")
	}
	errObj, ok := env["error"].(map[string]any)
	if !ok {
		t.Fatalf("error not object: %v", env["error"])
	}
	if errObj["code"] != "cache_missing" {
		t.Errorf("code=%v want cache_missing", errObj["code"])
	}
}

func TestRunQueryRepoInvalid(t *testing.T) {
	t.Setenv("DEVPILOT_HOME", t.TempDir())
	rc, env := runAndParse(t, "/definitely/not/here", "callers_of", "x", queryOpts{})
	if rc == 0 {
		t.Errorf("want non-zero rc")
	}
	errObj, ok := env["error"].(map[string]any)
	if !ok {
		t.Fatalf("error not object: %v", env["error"])
	}
	if errObj["code"] != "repo_invalid" {
		t.Errorf("code=%v want repo_invalid", errObj["code"])
	}
}

// TestEmptyResultNotNull ensures nil slices are normalised to [] in JSON.
func TestRunQueryEmptyResultIsArray(t *testing.T) {
	repo := seedQueryStore(t)
	// hub_a has no inbound edges in the fixture.
	rc, env := runAndParse(t, repo, "callers_of", "hub_a", queryOpts{depth: 2})
	if rc != 0 {
		t.Fatalf("rc=%d", rc)
	}
	_, payload := okData(t, env, "callers_of")
	list, ok := payload["callers"].([]any)
	if !ok {
		t.Fatalf("callers not array (must be [] not null): %v", payload)
	}
	if len(list) != 0 {
		t.Errorf("want empty, got %v", list)
	}
}
