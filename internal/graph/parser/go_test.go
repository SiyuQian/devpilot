package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoParserExtracts(t *testing.T) {
	mainSrc := func(t *testing.T) (string, []byte) {
		t.Helper()
		path := filepath.Join("testdata", "go", "simple", "main.go")
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return "simple/main.go", src
	}

	tests := []struct {
		name  string
		file  string
		src   func(t *testing.T) (string, []byte)
		check func(t *testing.T, r ParseResult)
	}{
		{
			name: "functions",
			check: func(t *testing.T, r ParseResult) {
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
			},
		},
		{
			name: "methods",
			check: func(t *testing.T, r ParseResult) {
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
			},
		},
		{
			name: "types",
			check: func(t *testing.T, r ParseResult) {
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
			},
		},
		{
			name: "calls",
			check: func(t *testing.T, r ParseResult) {
				calls := map[[2]string]bool{}
				for _, e := range r.Edges {
					if e.Kind == "calls" {
						calls[[2]string{e.Src, e.Dst}] = true
					}
				}
				if !calls[[2]string{"simple/main.go::Greet", "external::fmt.Sprintf"}] {
					t.Error("missing calls edge Greet -> external::fmt.Sprintf")
				}
				if !calls[[2]string{"simple/main.go::main", "simple/main.go::Greet"}] {
					t.Error("missing calls edge main -> Greet (intra-file)")
				}
				if !calls[[2]string{"simple/main.go::main", "external::fmt.Println"}] {
					t.Error("missing calls edge main -> external::fmt.Println")
				}
			},
		},
		{
			name: "imports",
			check: func(t *testing.T, r ParseResult) {
				imports := map[[2]string]bool{}
				for _, e := range r.Edges {
					if e.Kind == "imports" {
						imports[[2]string{e.Src, e.Dst}] = true
					}
				}
				want := [][2]string{
					{"simple/main.go", "external::fmt"},
					{"simple/main.go", "external::strings"},
				}
				for _, w := range want {
					if !imports[w] {
						t.Errorf("missing imports edge %s -> %s", w[0], w[1])
					}
				}
			},
		},
		{
			name: "tests_edges",
			src: func(t *testing.T) (string, []byte) {
				path := filepath.Join("testdata", "go", "simple", "main_test.go")
				src, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				return "simple/main_test.go", src
			},
			check: func(t *testing.T, r ParseResult) {
				hasTestNode := false
				for _, n := range r.Nodes {
					if n.ID == "simple/main_test.go::TestGreet" {
						hasTestNode = true
					}
				}
				if !hasTestNode {
					t.Fatal("missing TestGreet function node")
				}

				hasTestsEdge := false
				for _, e := range r.Edges {
					if e.Kind == "tests" && e.Src == "simple/main_test.go::TestGreet" && e.Dst == "external::Greet" {
						hasTestsEdge = true
					}
				}
				if !hasTestsEdge {
					var got []string
					for _, e := range r.Edges {
						if e.Kind == "tests" {
							got = append(got, e.Src+"->"+e.Dst)
						}
					}
					t.Fatalf("missing tests edge TestGreet -> external::Greet; got tests edges: %v", got)
				}
			},
		},
	}

	p := NewGoParser()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srcFn := tc.src
			if srcFn == nil {
				srcFn = mainSrc
			}
			filePath, src := srcFn(t)
			r, err := p.Parse(filePath, src)
			if err != nil {
				t.Fatal(err)
			}
			tc.check(t, r)
		})
	}
}
