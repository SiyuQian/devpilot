package resolver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/parser"
	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestResolveGoIntraModuleCalls(t *testing.T) {
	p := parser.NewGoParser()
	dir := filepath.Join("..", "parser", "testdata", "go", "multifile")
	aSrc, err := os.ReadFile(filepath.Join(dir, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	bSrc, err := os.ReadFile(filepath.Join(dir, "b.go"))
	if err != nil {
		t.Fatal(err)
	}
	rA, err := p.Parse("multifile/a.go", aSrc)
	if err != nil {
		t.Fatal(err)
	}
	rB, err := p.Parse("multifile/b.go", bSrc)
	if err != nil {
		t.Fatal(err)
	}

	resolved := Resolve([]parser.ParseResult{rA, rB})

	// Find the calls edge from A's body. It should now point to b.go::B.
	var foundCall bool
	for _, r := range resolved {
		for _, e := range r.Edges {
			if e.Kind == "calls" && e.Src == "multifile/a.go::A" {
				if e.Dst == "multifile/b.go::B" {
					foundCall = true
				}
			}
		}
	}
	if !foundCall {
		// Surface all calls edges from A for debugging.
		var seen []string
		for _, r := range resolved {
			for _, e := range r.Edges {
				if e.Kind == "calls" && e.Src == "multifile/a.go::A" {
					seen = append(seen, e.Dst)
				}
			}
		}
		t.Fatalf("expected calls edge multifile/a.go::A -> multifile/b.go::B, saw dsts: %v", seen)
	}

	// External calls to fmt.Println should remain external::fmt.Println.
	var foundFmt bool
	for _, r := range resolved {
		for _, e := range r.Edges {
			if e.Kind == "calls" && e.Src == "multifile/a.go::A" && e.Dst == "external::fmt.Println" {
				foundFmt = true
			}
		}
	}
	if !foundFmt {
		t.Errorf("external::fmt.Println edge was wrongly rewritten")
	}

	// Sanity: the resolver returns at least one ParseResult per input.
	if len(resolved) != 2 {
		t.Errorf("expected 2 ParseResults out, got %d", len(resolved))
	}
	_ = store.Node{} // keep store import used
}

func TestResolveEmitsImplementsEdges(t *testing.T) {
	p := parser.NewGoParser()
	dir := filepath.Join("..", "parser", "testdata", "go", "iface")
	src, err := os.ReadFile(filepath.Join(dir, "iface.go"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := p.Parse("iface/iface.go", src)
	if err != nil {
		t.Fatal(err)
	}
	resolved := Resolve([]parser.ParseResult{r})

	wantSrc := "iface/iface.go::Console"
	wantDst := "iface/iface.go::Greeter"
	var have, haveMute bool
	for _, rr := range resolved {
		for _, e := range rr.Edges {
			if e.Kind == "implements" && e.Src == wantSrc && e.Dst == wantDst {
				have = true
			}
			if e.Kind == "implements" && e.Src == "iface/iface.go::Mute" && e.Dst == wantDst {
				haveMute = true
			}
		}
	}
	if !have {
		t.Errorf("missing implements edge Console -> Greeter")
	}
	if haveMute {
		t.Errorf("Mute should NOT implement Greeter (no methods)")
	}
}
