package query

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestPreflightShapeMatchesSpec(t *testing.T) {
	// Parse a single Go file from the existing parser testdata.
	p := parser.NewGoParser()
	src := []byte(`package x
func Foo() {}
func Bar() { Foo() }
`)
	r, err := p.Parse("x/x.go", src)
	if err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(t.TempDir() + "/graph.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if err := st.InsertNodes(r.Nodes); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertEdges(r.Edges); err != nil {
		t.Fatal(err)
	}

	prevGitRun := gitRun
	t.Cleanup(func() { gitRun = prevGitRun })
	gitRun = func(repo string, args ...string) ([]byte, error) {
		switch args[0] {
		case "diff":
			return []byte("M\tx/x.go\n"), nil
		case "show":
			if strings.HasSuffix(args[1], "BASE:x/x.go") {
				return []byte("old"), nil
			}
			return []byte("new"), nil
		}
		return nil, nil
	}

	res, err := Preflight(st, PreflightInput{RepoRoot: "/fake", Base: "BASE", Head: "HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"mode"`, `"changed_symbols"`, `"risk_summary"`,
		`"cross_community_edges"`, `"truncated_symbols"`,
		`"callers"`, `"tests"`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing field %s in marshalled payload:\n%s", want, string(b))
		}
	}
}
