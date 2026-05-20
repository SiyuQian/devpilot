package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJavaScriptParserExtracts(t *testing.T) {
	load := func(t *testing.T) (string, []byte) {
		t.Helper()
		path := filepath.Join("testdata", "js", "simple", "main.js")
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return "simple/main.js", src
	}

	p := NewJavaScriptParser()
	path, src := load(t)
	r, err := p.Parse(path, src)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("nodes", func(t *testing.T) {
		want := map[string]string{
			"simple/main.js":                 "file",
			"simple/main.js::greet":          "function",
			"simple/main.js::internalHelper": "function",
			"simple/main.js::Greeter":        "class",
			"simple/main.js::Greeter.hello":  "method",
			"simple/main.js::Base":           "class",
		}
		seen := map[string]string{}
		for _, n := range r.Nodes {
			if _, ok := want[n.ID]; ok {
				seen[n.ID] = n.Kind
			}
		}
		for id, kind := range want {
			if seen[id] != kind {
				t.Errorf("%s kind=%q want %q", id, seen[id], kind)
			}
		}
	})

	t.Run("calls", func(t *testing.T) {
		want := map[[2]string]bool{
			{"simple/main.js::internalHelper", "simple/main.js::greet"}: false,
			{"simple/main.js::Greeter.hello", "simple/main.js::greet"}:  false,
		}
		for _, e := range r.Edges {
			if e.Kind == "calls" {
				want[[2]string{e.Src, e.Dst}] = true
			}
		}
		for k, seen := range want {
			if !seen {
				t.Errorf("missing calls edge %s -> %s", k[0], k[1])
			}
		}
	})

	t.Run("extends", func(t *testing.T) {
		var ok bool
		for _, e := range r.Edges {
			if e.Kind == "extends" && e.Src == "simple/main.js::Greeter" && e.Dst == "simple/main.js::Base" {
				ok = true
			}
		}
		if !ok {
			t.Error("missing extends edge Greeter -> Base")
		}
	})
}
