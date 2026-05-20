package query

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// ContextResult is the return type of Context: the target's source snippet
// plus, optionally, snippets for its direct callers.
type ContextResult struct {
	Target  Snippet
	Callers []Snippet
}

// Snippet captures a node's source-line range from disk.
type Snippet struct {
	ID        string
	Path      string
	StartLine int
	EndLine   int
	Source    string
}

// Context returns the source snippet for id, optionally including caller
// snippets when depth >= 1. repoRoot is the absolute directory that node
// paths are relative to.
func Context(r Reader, id string, depth int, repoRoot string) (ContextResult, error) {
	node, err := r.GetNode(id)
	if err != nil {
		return ContextResult{}, fmt.Errorf("Context: %w", err)
	}
	tgt, err := snippetFromNode(node, repoRoot)
	if err != nil {
		return ContextResult{}, err
	}
	res := ContextResult{Target: tgt}
	if depth < 1 {
		return res, nil
	}
	callers, err := CallersOf(r, id, 1)
	if err != nil {
		return ContextResult{}, err
	}
	for _, c := range callers {
		n, err := r.GetNode(c.ID)
		if err != nil {
			continue // caller without recorded node (synthetic external::...)
		}
		s, err := snippetFromNode(n, repoRoot)
		if err != nil {
			continue
		}
		res.Callers = append(res.Callers, s)
	}
	return res, nil
}

func snippetFromNode(n store.Node, repoRoot string) (Snippet, error) {
	if n.StartLine <= 0 || n.EndLine < n.StartLine {
		return Snippet{ID: n.ID, Path: n.Path, StartLine: n.StartLine, EndLine: n.EndLine}, nil
	}
	abs := filepath.Join(repoRoot, n.Path)
	data, err := os.ReadFile(abs)
	if err != nil {
		return Snippet{}, fmt.Errorf("read %s: %w", abs, err)
	}
	lines := strings.Split(string(data), "\n")
	if n.EndLine > len(lines) {
		return Snippet{}, fmt.Errorf("snippet out of range for %s: end=%d len=%d", n.ID, n.EndLine, len(lines))
	}
	src := strings.Join(lines[n.StartLine-1:n.EndLine], "\n")
	return Snippet{
		ID: n.ID, Path: n.Path,
		StartLine: n.StartLine, EndLine: n.EndLine,
		Source: src,
	}, nil
}
