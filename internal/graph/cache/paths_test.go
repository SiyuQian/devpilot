package cache

import (
	"strings"
	"testing"
)

func TestRepoKey(t *testing.T) {
	k1 := RepoKey("/Users/x/code/foo")
	k2 := RepoKey("/Users/x/code/foo")
	k3 := RepoKey("/Users/x/code/bar")
	if k1 != k2 {
		t.Errorf("RepoKey not deterministic: %q vs %q", k1, k2)
	}
	if k1 == k3 {
		t.Errorf("RepoKey not unique across paths: %q == %q", k1, k3)
	}
	if len(k1) != 12 {
		t.Errorf("RepoKey length=%d, want 12", len(k1))
	}
	if strings.ContainsAny(k1, "/. ") {
		t.Errorf("RepoKey contains illegal chars: %q", k1)
	}
}

func TestPathLayout(t *testing.T) {
	k := "abcdef012345"
	got := GraphDB("/tmp/devpilot-home", k)
	want := "/tmp/devpilot-home/graphs/abcdef012345/graph.db"
	if got != want {
		t.Errorf("GraphDB=%q want %q", got, want)
	}
	if !strings.HasPrefix(PreflightFile("/tmp/devpilot-home", k), "/tmp/devpilot-home/preflight/abcdef012345-") {
		t.Errorf("PreflightFile prefix mismatch")
	}
}
