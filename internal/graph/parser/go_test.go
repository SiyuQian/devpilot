package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoParserExtractsFunctions(t *testing.T) {
	p := NewGoParser()
	path := filepath.Join("testdata", "go", "simple", "main.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.Parse("simple/main.go", src)
	if err != nil {
		t.Fatal(err)
	}

	var hasGreet, hasMain, hasFile bool
	for _, n := range r.Nodes {
		switch n.ID {
		case "simple/main.go::Greet":
			hasGreet = true
			if !n.IsExported {
				t.Error("Greet must be exported")
			}
		case "simple/main.go::main":
			hasMain = true
		case "simple/main.go":
			hasFile = true
			if n.Kind != "file" {
				t.Errorf("file node kind=%q, want file", n.Kind)
			}
		}
	}
	if !hasGreet || !hasMain || !hasFile {
		t.Fatalf("missing nodes: greet=%v main=%v file=%v", hasGreet, hasMain, hasFile)
	}
}

func TestGoParserExtractsMethods(t *testing.T) {
	p := NewGoParser()
	src, err := os.ReadFile(filepath.Join("testdata", "go", "simple", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.Parse("simple/main.go", src)
	if err != nil {
		t.Fatal(err)
	}

	wantIDs := map[string]bool{
		"simple/main.go::Greeter.Hello":  false,
		"simple/main.go::Greeter.silent": false,
	}
	for _, n := range r.Nodes {
		if _, ok := wantIDs[n.ID]; ok {
			wantIDs[n.ID] = true
			if n.Kind != "method" {
				t.Errorf("%s kind=%q, want method", n.ID, n.Kind)
			}
			if n.Container != "Greeter" {
				t.Errorf("%s container=%q, want Greeter", n.ID, n.Container)
			}
			if n.ID == "simple/main.go::Greeter.Hello" && !n.IsExported {
				t.Errorf("Hello must be exported")
			}
			if n.ID == "simple/main.go::Greeter.silent" && n.IsExported {
				t.Errorf("silent must NOT be exported")
			}
		}
	}
	for id, found := range wantIDs {
		if !found {
			t.Errorf("missing method node: %s", id)
		}
	}
}

func TestGoParserExtractsTypes(t *testing.T) {
	p := NewGoParser()
	src, err := os.ReadFile(filepath.Join("testdata", "go", "simple", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.Parse("simple/main.go", src)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"simple/main.go::Greeter":  "struct",
		"simple/main.go::Greeter2": "struct",
		"simple/main.go::Hello":    "interface",
		"simple/main.go::Alias":    "type",
		"simple/main.go::IntPtr":   "type",
	}
	got := map[string]string{}
	for _, n := range r.Nodes {
		if _, ok := want[n.ID]; ok {
			got[n.ID] = n.Kind
		}
	}
	for id, kind := range want {
		if got[id] != kind {
			t.Errorf("%s: got kind=%q, want %q", id, got[id], kind)
		}
	}
}
