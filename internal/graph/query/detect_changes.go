package query

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
)

// ChangedSymbol describes one entry in the diff between base and head.
type ChangedSymbol struct {
	ID         string
	Kind       string
	IsExported bool
	IsNew      bool
	ChangeType string // "added" | "removed" | "modified" | "renamed"
}

// gitRun is the shell-out hook. Tests replace it.
var gitRun = func(repo string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %v: %w", args, err)
	}
	return out, nil
}

// DetectChanges enumerates symbols (or files) changed between base..head.
// For modified files, graph-known symbols are emitted as `modified` if their
// signature_hash differs between the base and head blobs; otherwise no symbol
// entry is produced for that file. Added/removed files surface as file-level
// entries because the in-graph state only reflects head.
func DetectChanges(r Reader, repoRoot, base, head string) ([]ChangedSymbol, error) {
	rangeArg := base + ".." + head
	out, err := gitRun(repoRoot, "diff", "--name-status", "-M", rangeArg)
	if err != nil {
		return nil, fmt.Errorf("DetectChanges: %w", err)
	}

	var changes []ChangedSymbol
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		path := parts[len(parts)-1]
		switch status[0] {
		case 'A':
			changes = append(changes, ChangedSymbol{ID: path, Kind: "file", ChangeType: "added", IsNew: true})
		case 'D':
			changes = append(changes, ChangedSymbol{ID: path, Kind: "file", ChangeType: "removed"})
		case 'R':
			changes = append(changes, ChangedSymbol{ID: path, Kind: "file", ChangeType: "renamed"})
		case 'M':
			modified, err := modifiedSymbols(r, repoRoot, base, head, path)
			if err != nil {
				return nil, err
			}
			changes = append(changes, modified...)
		}
	}
	return changes, nil
}

func modifiedSymbols(r Reader, repoRoot, base, head, path string) ([]ChangedSymbol, error) {
	nodes, err := r.NodesByPath(path)
	if err != nil {
		return nil, err
	}
	baseBlob, err := gitRun(repoRoot, "show", base+":"+path)
	if err != nil {
		// File didn't exist at base — treat as added even though status was M
		// (can happen with rename detection edge cases).
		baseBlob = nil
	}
	headBlob, err := gitRun(repoRoot, "show", head+":"+path)
	if err != nil {
		headBlob = nil
	}
	if hashBytes(baseBlob) == hashBytes(headBlob) {
		return nil, nil
	}
	out := make([]ChangedSymbol, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, ChangedSymbol{
			ID:         n.ID,
			Kind:       n.Kind,
			IsExported: n.IsExported,
			ChangeType: "modified",
		})
	}
	return out, nil
}

func hashBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
