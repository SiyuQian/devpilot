package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTypeScriptParserExtracts(t *testing.T) {
	loadSimple := func(t *testing.T) (string, []byte) {
		t.Helper()
		path := filepath.Join("testdata", "ts", "simple", "main.ts")
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return "simple/main.ts", src
	}

	t.Run("functions", func(t *testing.T) {
		p := NewTypeScriptParser()
		path, src := loadSimple(t)
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		var hasGreet, hasInternal, hasFile bool
		for _, n := range r.Nodes {
			switch n.ID {
			case "simple/main.ts::greet":
				hasGreet = true
				if !n.IsExported {
					t.Error("greet must be exported")
				}
				if n.Kind != "function" {
					t.Errorf("greet kind=%q want function", n.Kind)
				}
			case "simple/main.ts::internalHelper":
				hasInternal = true
				if n.IsExported {
					t.Error("internalHelper must NOT be exported")
				}
			case "simple/main.ts":
				hasFile = true
				if n.Kind != "file" {
					t.Errorf("file kind=%q want file", n.Kind)
				}
			}
		}
		if !hasGreet || !hasInternal || !hasFile {
			t.Fatalf("missing nodes: greet=%v internal=%v file=%v", hasGreet, hasInternal, hasFile)
		}
	})

	t.Run("methods", func(t *testing.T) {
		p := NewTypeScriptParser()
		path, src := loadSimple(t)
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]struct{ exported bool }{
			"simple/main.ts::Greeter.hello":  {exported: true},
			"simple/main.ts::Greeter.silent": {exported: false},
		}
		seen := map[string]bool{}
		for _, n := range r.Nodes {
			if w, ok := want[n.ID]; ok {
				seen[n.ID] = true
				if n.Kind != "method" {
					t.Errorf("%s kind=%q want method", n.ID, n.Kind)
				}
				if n.Container != "Greeter" {
					t.Errorf("%s container=%q want Greeter", n.ID, n.Container)
				}
				if n.IsExported != w.exported {
					t.Errorf("%s exported=%v want %v", n.ID, n.IsExported, w.exported)
				}
			}
		}
		for id := range want {
			if !seen[id] {
				t.Errorf("missing method node: %s", id)
			}
		}
		var hasClass bool
		for _, n := range r.Nodes {
			if n.ID == "simple/main.ts::Greeter" {
				hasClass = true
				if n.Kind != "class" {
					t.Errorf("Greeter kind=%q want class", n.Kind)
				}
				if !n.IsExported {
					t.Error("Greeter must be exported")
				}
			}
		}
		if !hasClass {
			t.Error("missing class node Greeter")
		}
	})
}
