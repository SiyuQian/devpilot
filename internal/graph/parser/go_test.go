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
