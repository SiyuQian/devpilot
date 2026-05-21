package query

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestPreflightShapeMatchesSpec(t *testing.T) {
	// Construct a minimal graph: file x/x.go containing two functions Foo and Bar,
	// with Bar calling Foo. Hand-rolled rather than parsed so this test does not
	// depend on any specific parser backend.
	nodes := []store.Node{
		{ID: "x/x.go", Kind: "file", Path: "x/x.go", Name: "x/x.go", Language: "go"},
		{ID: "x/x.go::Foo", Kind: "function", Path: "x/x.go", Name: "Foo", Language: "go", StartLine: 2, EndLine: 2, IsExported: true},
		{ID: "x/x.go::Bar", Kind: "function", Path: "x/x.go", Name: "Bar", Language: "go", StartLine: 3, EndLine: 3, IsExported: true},
	}
	edges := []store.Edge{
		{Src: "x/x.go", Dst: "x/x.go::Foo", Kind: "contains"},
		{Src: "x/x.go", Dst: "x/x.go::Bar", Kind: "contains"},
		{Src: "x/x.go::Bar", Dst: "x/x.go::Foo", Kind: "calls"},
	}

	st, err := store.Open(t.TempDir() + "/graph.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if err := st.InsertNodes(nodes); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertEdges(edges); err != nil {
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
