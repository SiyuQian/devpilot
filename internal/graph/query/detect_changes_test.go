package query

import (
	"reflect"
	"sort"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestDetectChanges(t *testing.T) {
	// Graph reflects HEAD state.
	nodes := []store.Node{
		{ID: "api/checkout.go::handleCheckout", Kind: "function", Path: "api/checkout.go",
			Name: "handleCheckout", Language: "go", IsExported: true, SignatureHash: "newhash"},
		{ID: "internal/auth/session.go::Validate", Kind: "function", Path: "internal/auth/session.go",
			Name: "Validate", Language: "go", IsExported: true, SignatureHash: "same"},
	}
	r := newStore(t, nodes, nil)

	// Pretend git returns: M api/checkout.go, A internal/new/file.go, D internal/old/gone.go
	prevGitRun := gitRun
	t.Cleanup(func() { gitRun = prevGitRun })
	gitRun = func(repo string, args ...string) ([]byte, error) {
		switch {
		case len(args) > 0 && args[0] == "diff" && contains(args, "--name-status"):
			return []byte("M\tapi/checkout.go\nA\tinternal/new/file.go\nD\tinternal/old/gone.go\n"), nil
		case len(args) > 0 && args[0] == "show":
			// `git show base:path` and `git show head:path`
			// Return a stub body whose signature hash will differ for the M file
			// and be identical for the U entry (none here).
			if contains(args, "BASE:api/checkout.go") {
				return []byte("old-body"), nil
			}
			if contains(args, "HEAD:api/checkout.go") {
				return []byte("new-body"), nil
			}
			return nil, nil
		}
		return nil, nil
	}

	got, err := DetectChanges(r, "/fake/repo", "BASE", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(got, func(i, j int) bool { return got[i].ID < got[j].ID })

	want := []ChangedSymbol{
		{
			ID:         "api/checkout.go::handleCheckout",
			Kind:       "function",
			IsExported: true,
			IsNew:      false,
			ChangeType: "modified",
		},
		// New file's symbols cannot be enumerated from the graph (they'd be in HEAD).
		// Phase 3 keeps DetectChanges focused on graph-known symbols; a new file
		// surfaces as a file-level entry instead.
		{
			ID:         "internal/new/file.go",
			Kind:       "file",
			ChangeType: "added",
			IsNew:      true,
		},
		{
			ID:         "internal/old/gone.go",
			Kind:       "file",
			ChangeType: "removed",
			IsNew:      false,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got=%+v\nwant=%+v", got, want)
	}
}

func contains(s []string, target string) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}
