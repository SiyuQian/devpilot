package parser

import "testing"

func TestRegistryForLanguage(t *testing.T) {
	r := DefaultRegistry()
	if p := r.ForLanguage("go"); p == nil || p.Language() != "go" {
		t.Fatalf("ForLanguage(go) = %v", p)
	}
	if p := r.ForLanguage("missing"); p != nil {
		t.Fatalf("ForLanguage(missing) = %v", p)
	}
}

func TestRelFromPrefix(t *testing.T) {
	if got := relFromPrefix("/repo/pkg/a.go", "/repo"); got != "/pkg/a.go" {
		t.Fatalf("relFromPrefix = %q", got)
	}
	if got := relFromPrefix("pkg/a.go", "/repo"); got != "pkg/a.go" {
		t.Fatalf("relFromPrefix relative = %q", got)
	}
}
