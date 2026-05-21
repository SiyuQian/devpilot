package parser

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// Compile-time assertion that GoNativeParser implements Parser.
var _ Parser = (*GoNativeParser)(nil)

func TestGoNativeParserSkeleton(t *testing.T) {
	parser := NewGoNativeParser()
	if parser == nil {
		t.Fatal("NewGoNativeParser() returned nil")
	}

	// Test Language() returns "go"
	if got := parser.Language(); got != "go" {
		t.Errorf("Language() = %q, want %q", got, "go")
	}

	// Test Extensions() returns [".go"]
	exts := parser.Extensions()
	if len(exts) != 1 || exts[0] != ".go" {
		t.Errorf("Extensions() = %v, want [.go]", exts)
	}

	// Test Parse returns empty ParseResult with no error
	src := []byte("package main\n\nfunc main() {}")
	result, err := parser.Parse("foo.go", src)
	if err != nil {
		t.Errorf("Parse() returned error: %v", err)
	}

	// Verify empty ParseResult
	if len(result.Nodes) != 0 {
		t.Errorf("Parse() Nodes = %v, want empty", result.Nodes)
	}
	if len(result.Edges) != 0 {
		t.Errorf("Parse() Edges = %v, want empty", result.Edges)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Parse() Errors = %v, want empty", result.Errors)
	}
	if len(result.InterfaceMethods) != 0 {
		t.Errorf("Parse() InterfaceMethods = %v, want empty", result.InterfaceMethods)
	}
}

// flattenNodes returns all nodes across every ParseResult, keyed by ID.
func flattenNodes(t *testing.T, results map[string]ParseResult) map[string]store.Node {
	t.Helper()
	out := map[string]store.Node{}
	for _, r := range results {
		for _, n := range r.Nodes {
			if existing, ok := out[n.ID]; ok {
				t.Fatalf("duplicate node ID %q across results: first=%+v second=%+v", n.ID, existing, n)
			}
			out[n.ID] = n
		}
	}
	return out
}

func TestLoadModuleProducesNodes(t *testing.T) {
	abs, err := filepath.Abs("testdata/go_native")
	if err != nil {
		t.Fatalf("abs testdata: %v", err)
	}
	results, err := NewGoNativeParser().LoadModule(abs)
	if err != nil {
		t.Fatalf("LoadModule: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected non-empty results")
	}

	nodes := flattenNodes(t, results)

	type want struct {
		id         string
		kind       string
		container  string
		isExported bool
	}
	wants := []want{
		{"pkg/a/a.go::Greet", "function", "", true},
		{"pkg/a/a.go::Run", "function", "", true},
		{"pkg/a/a_test.go::TestGreet", "function", "", true},
		{"pkg/b/b.go::B", "function", "", true},
		{"pkg/impl/impl.go::Console.Speak", "method", "Console", true},
	}
	for _, w := range wants {
		n, ok := nodes[w.id]
		if !ok {
			t.Errorf("missing node %q; have %v", w.id, sortedKeys(nodes))
			continue
		}
		if n.Kind != w.kind {
			t.Errorf("%s: kind = %q, want %q", w.id, n.Kind, w.kind)
		}
		if n.Container != w.container {
			t.Errorf("%s: container = %q, want %q", w.id, n.Container, w.container)
		}
		if n.IsExported != w.isExported {
			t.Errorf("%s: isExported = %v, want %v", w.id, n.IsExported, w.isExported)
		}
		if n.Language != "go" {
			t.Errorf("%s: language = %q, want \"go\"", w.id, n.Language)
		}
	}

	// One representative node has non-zero, sensible line range.
	if greet, ok := nodes["pkg/a/a.go::Greet"]; ok {
		if greet.StartLine == 0 || greet.EndLine == 0 || greet.EndLine < greet.StartLine {
			t.Errorf("pkg/a/a.go::Greet: bad line range start=%d end=%d", greet.StartLine, greet.EndLine)
		}
	}

	// Every non-synthetic result has a file node whose ID equals its key.
	for key, r := range results {
		if key == "" {
			continue
		}
		found := false
		for _, n := range r.Nodes {
			if n.Kind == "file" && n.ID == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("result %q missing file node", key)
		}
	}

	// No external:: IDs and only contains edges in this task.
	for key, r := range results {
		for _, n := range r.Nodes {
			if strings.HasPrefix(n.ID, "external::") {
				t.Errorf("result %q has external node %q", key, n.ID)
			}
		}
		for _, e := range r.Edges {
			if e.Kind != "contains" {
				t.Errorf("result %q has non-contains edge kind %q", key, e.Kind)
			}
			if strings.HasPrefix(e.Dst, "external::") {
				t.Errorf("result %q has external edge dst %q", key, e.Dst)
			}
		}
	}
}

func TestLoadModuleDeterministic(t *testing.T) {
	abs, err := filepath.Abs("testdata/go_native")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	p := NewGoNativeParser()
	a, err := p.LoadModule(abs)
	if err != nil {
		t.Fatalf("first LoadModule: %v", err)
	}
	b, err := p.LoadModule(abs)
	if err != nil {
		t.Fatalf("second LoadModule: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("LoadModule not deterministic\n  a keys: %v\n  b keys: %v",
			sortedResultKeys(a), sortedResultKeys(b))
	}
}

func sortedKeys(m map[string]store.Node) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedResultKeys(m map[string]ParseResult) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
