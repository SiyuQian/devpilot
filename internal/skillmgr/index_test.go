package skillmgr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseIndex(t *testing.T) {
	data := []byte(`{
		"skills": [
			{"name": "pm", "description": "Product manager", "files": ["SKILL.md", "references/guide.md"]},
			{"name": "trello", "description": "Trello integration", "files": ["SKILL.md"]}
		]
	}`)

	entries, err := ParseIndex(data)
	if err != nil {
		t.Fatalf("ParseIndex: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
	if entries[0].Name != "pm" {
		t.Errorf("entries[0].Name = %q, want %q", entries[0].Name, "pm")
	}
	if entries[0].Description != "Product manager" {
		t.Errorf("entries[0].Description = %q", entries[0].Description)
	}
	if len(entries[0].Files) != 2 {
		t.Errorf("entries[0].Files len = %d, want 2", len(entries[0].Files))
	}
	if entries[1].Name != "trello" {
		t.Errorf("entries[1].Name = %q", entries[1].Name)
	}
}

func TestParseIndexEmpty(t *testing.T) {
	data := []byte(`{"skills": []}`)
	entries, err := ParseIndex(data)
	if err != nil {
		t.Fatalf("ParseIndex: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len = %d, want 0", len(entries))
	}
}

func TestParseIndexSkipsEmptyNames(t *testing.T) {
	data := []byte(`{"skills": [{"name": "pm", "description": "PM", "files": ["SKILL.md"]}, {"name": "", "description": "bad", "files": []}]}`)
	entries, err := ParseIndex(data)
	if err != nil {
		t.Fatalf("ParseIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len = %d, want 1 (empty name should be filtered)", len(entries))
	}
	if entries[0].Name != "pm" {
		t.Errorf("Name = %q, want %q", entries[0].Name, "pm")
	}
}

func TestParseIndexInvalidJSON(t *testing.T) {
	_, err := ParseIndex([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFetchIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/siyuqian/devpilot/v1.0.0/skills/index.json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"skills":[{"name":"pm","description":"PM skill","files":["SKILL.md"]}]}`))
	}))
	defer srv.Close()

	entries, err := fetchIndexFromBase(context.Background(), srv.URL, "siyuqian", "devpilot", "v1.0.0")
	if err != nil {
		t.Fatalf("FetchIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len = %d, want 1", len(entries))
	}
	if entries[0].Name != "pm" {
		t.Errorf("Name = %q, want %q", entries[0].Name, "pm")
	}
}

func TestFetchIndexNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := fetchIndexFromBase(context.Background(), srv.URL, "o", "r", "v1.0.0")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}
