package project

import "testing"

func TestResolveSource(t *testing.T) {
	cfg := &Config{Source: "github"}
	if got := cfg.ResolveSource(""); got != "github" {
		t.Fatalf("ResolveSource empty = %q, want github", got)
	}
	if got := cfg.ResolveSource("trello"); got != "trello" {
		t.Fatalf("ResolveSource override = %q, want trello", got)
	}
	if got := (&Config{}).ResolveSource(""); got != "trello" {
		t.Fatalf("ResolveSource default = %q, want trello", got)
	}
}
