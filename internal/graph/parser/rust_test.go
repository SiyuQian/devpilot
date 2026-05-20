package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRustParserExtracts(t *testing.T) {
	loadSimple := func(t *testing.T) (string, []byte) {
		t.Helper()
		path := filepath.Join("testdata", "rust", "simple", "lib.rs")
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return "simple/lib.rs", src
	}

	p := NewRustParser()
	path, src := loadSimple(t)
	r, err := p.Parse(path, src)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]struct {
		kind     string
		exported bool
	}{
		"simple/lib.rs":                  {"file", false},
		"simple/lib.rs::greet":           {"function", true},
		"simple/lib.rs::internal_helper": {"function", false},
		"simple/lib.rs::Greeter":         {"struct", true},
		"simple/lib.rs::Mood":            {"enum", true},
		"simple/lib.rs::Greeting":        {"type", true},
	}
	seen := map[string]bool{}
	for _, n := range r.Nodes {
		w, ok := want[n.ID]
		if !ok {
			continue
		}
		seen[n.ID] = true
		if n.Kind != w.kind {
			t.Errorf("%s kind=%q want %q", n.ID, n.Kind, w.kind)
		}
		if n.IsExported != w.exported {
			t.Errorf("%s exported=%v want %v", n.ID, n.IsExported, w.exported)
		}
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("missing node: %s", id)
		}
	}
}
