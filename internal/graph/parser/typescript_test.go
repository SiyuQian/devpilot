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

	t.Run("calls", func(t *testing.T) {
		p := NewTypeScriptParser()
		path, src := loadSimple(t)
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		want := map[[2]string]bool{
			{"simple/main.ts::internalHelper", "simple/main.ts::greet"}: false,
			{"simple/main.ts::Greeter.hello", "simple/main.ts::greet"}:  false,
		}
		for _, e := range r.Edges {
			if e.Kind != "calls" {
				continue
			}
			key := [2]string{e.Src, e.Dst}
			if _, ok := want[key]; ok {
				want[key] = true
			}
		}
		for k, seen := range want {
			if !seen {
				t.Errorf("missing calls edge %s -> %s", k[0], k[1])
			}
		}
	})

	t.Run("imports", func(t *testing.T) {
		p := NewTypeScriptParser()
		path := "multifile/a.ts"
		src, err := os.ReadFile(filepath.Join("testdata", "ts", "multifile", "a.ts"))
		if err != nil {
			t.Fatal(err)
		}
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		var seen bool
		for _, e := range r.Edges {
			if e.Kind == "imports" && e.Src == "multifile/a.ts" && e.Dst == "external::./b" {
				seen = true
			}
		}
		if !seen {
			t.Fatalf("missing imports edge multifile/a.ts -> external::./b; edges=%v", r.Edges)
		}
	})

	t.Run("tests_edges", func(t *testing.T) {
		p := NewTypeScriptParser()
		path := "simple/main.test.ts"
		src, err := os.ReadFile(filepath.Join("testdata", "ts", "simple", "main.test.ts"))
		if err != nil {
			t.Fatal(err)
		}
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		var count int
		for _, e := range r.Edges {
			if e.Kind == "tests" && e.Src == "simple/main.test.ts" && e.Dst == "external::greet" {
				count++
			}
		}
		if count == 0 {
			t.Fatalf("expected at least one tests edge from main.test.ts to external::greet; edges=%v", r.Edges)
		}
	})

	t.Run("types", func(t *testing.T) {
		p := NewTypeScriptParser()
		path, src := loadSimple(t)
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		var hasIface, hasTypeAlias bool
		for _, n := range r.Nodes {
			if n.ID == "simple/main.ts::Speaker" {
				hasIface = true
				if n.Kind != "interface" {
					t.Errorf("Speaker kind=%q want interface", n.Kind)
				}
				if !n.IsExported {
					t.Error("Speaker must be exported")
				}
			}
			if n.ID == "simple/main.ts::Greeting" {
				hasTypeAlias = true
				if n.Kind != "type" {
					t.Errorf("Greeting kind=%q want type", n.Kind)
				}
			}
		}
		if !hasIface || !hasTypeAlias {
			t.Fatalf("missing: iface=%v typeAlias=%v", hasIface, hasTypeAlias)
		}
		methods, ok := r.InterfaceMethods["simple/main.ts::Speaker"]
		if !ok || len(methods) != 1 || methods[0] != "hello" {
			t.Errorf("InterfaceMethods[Speaker]=%v, want [hello]", methods)
		}
	})

	t.Run("implements_extends", func(t *testing.T) {
		p := NewTypeScriptParser()
		path, src := loadSimple(t)
		r, err := p.Parse(path, src)
		if err != nil {
			t.Fatal(err)
		}
		var hasImpl, hasExt bool
		for _, e := range r.Edges {
			if e.Kind == "implements" && e.Src == "simple/main.ts::Greeter" && e.Dst == "simple/main.ts::Speaker" {
				hasImpl = true
			}
			if e.Kind == "extends" && e.Src == "simple/main.ts::Greeter" && e.Dst == "simple/main.ts::Base" {
				hasExt = true
			}
		}
		if !hasImpl || !hasExt {
			t.Fatalf("missing edges: implements=%v extends=%v", hasImpl, hasExt)
		}
	})
}
