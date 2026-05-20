package envelope

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewAndOK(t *testing.T) {
	e := New("graph.status").OK(map[string]any{"foo": "bar"})
	if !e.OKFlag {
		t.Fatal("expected ok=true after OK")
	}
	if e.Command != "graph.status" {
		t.Errorf("command=%q", e.Command)
	}
	if e.Error != nil {
		t.Errorf("error should be nil, got %+v", e.Error)
	}
}

func TestErr(t *testing.T) {
	e := New("graph.build").Err("repo_not_found", "no such repo: /tmp/x")
	if e.OKFlag {
		t.Fatal("expected ok=false after Err")
	}
	if e.Error == nil || e.Error.Code != "repo_not_found" {
		t.Fatalf("unexpected error: %+v", e.Error)
	}
	if e.Data != nil {
		t.Errorf("data must be nil on error")
	}
}

func TestSuggestAppends(t *testing.T) {
	e := New("graph.detect-changes").OK(nil).Suggest("devpilot graph context --id foo", "devpilot graph preflight")
	if len(e.NextToolSuggestions) != 2 {
		t.Fatalf("want 2 suggestions, got %d", len(e.NextToolSuggestions))
	}
}

func TestWarnAppends(t *testing.T) {
	e := New("graph.build").OK(nil).Warn("slow disk")
	if len(e.Warnings) != 1 || e.Warnings[0] != "slow disk" {
		t.Fatalf("warnings=%+v", e.Warnings)
	}
}

func TestMarshalShape(t *testing.T) {
	e := New("graph.status").OK(map[string]any{"nodes": 12})
	b, err := e.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		`"schema_version":"1"`,
		`"command":"graph.status"`,
		`"ok":true`,
		`"data":{"nodes":12}`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %s", want, s)
		}
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
}
