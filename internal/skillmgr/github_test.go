package skillmgr

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchLatestTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	defer srv.Close()

	tag, err := fetchLatestTagFromURL(srv.URL + "/repos/owner/repo/releases/latest")
	if err != nil {
		t.Fatalf("FetchLatestTag: %v", err)
	}
	if tag != "v1.2.3" {
		t.Errorf("tag = %q, want v1.2.3", tag)
	}
}

func TestFetchLatestTagNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := fetchLatestTagFromURL(srv.URL + "/repos/owner/repo/releases/latest")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestFetchLatestTagRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := fetchLatestTagFromURL(srv.URL + "/repos/owner/repo/releases/latest")
	if err == nil {
		t.Fatal("expected error for rate limit, got nil")
	}
}

func TestFetchSkill(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/contents/.claude/skills/pm":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{"type":"file","name":"SKILL.md","download_url":"` + "http://" + r.Host + `/file/SKILL.md"},
				{"type":"dir","name":"references","download_url":""}
			]`))
		case "/repos/owner/repo/contents/.claude/skills/pm/references":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{"type":"file","name":"guide.md","download_url":"` + "http://" + r.Host + `/file/guide.md"}
			]`))
		case "/file/SKILL.md":
			_, _ = w.Write([]byte("---\nname: pm\n---"))
		case "/file/guide.md":
			_, _ = w.Write([]byte("# Guide"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	files, err := fetchSkillFromBase(srv.URL+"/repos/owner/repo", "pm", "v1.0.0")
	if err != nil {
		t.Fatalf("FetchSkill: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}

	byPath := make(map[string][]byte)
	for _, f := range files {
		byPath[f.Path] = f.Content
	}

	if string(byPath["SKILL.md"]) != "---\nname: pm\n---" {
		t.Errorf("SKILL.md content = %q", string(byPath["SKILL.md"]))
	}
	if string(byPath["references/guide.md"]) != "# Guide" {
		t.Errorf("references/guide.md content = %q", string(byPath["references/guide.md"]))
	}
}

func TestFetchSkillNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := fetchSkillFromBase(srv.URL+"/repos/owner/repo", "nonexistent", "v1.0.0")
	if err == nil {
		t.Fatal("expected error for missing skill, got nil")
	}
}

func TestInstallSkill(t *testing.T) {
	dir := t.TempDir()
	files := []SkillFile{
		{Path: "SKILL.md", Content: []byte("---\nname: pm\n---")},
		{Path: "references/guide.md", Content: []byte("# Guide")},
	}

	if err := InstallSkill(dir, "pm", files); err != nil {
		t.Fatalf("InstallSkill: %v", err)
	}

	skillMD, err := os.ReadFile(filepath.Join(dir, ".claude", "skills", "pm", "SKILL.md"))
	if err != nil {
		t.Fatalf("reading SKILL.md: %v", err)
	}
	if string(skillMD) != "---\nname: pm\n---" {
		t.Errorf("SKILL.md = %q", string(skillMD))
	}

	guide, err := os.ReadFile(filepath.Join(dir, ".claude", "skills", "pm", "references", "guide.md"))
	if err != nil {
		t.Fatalf("reading guide.md: %v", err)
	}
	if string(guide) != "# Guide" {
		t.Errorf("guide.md = %q", string(guide))
	}
}

func TestInstallSkillOverwrites(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".claude", "skills", "pm")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("old content"), 0644)

	files := []SkillFile{
		{Path: "SKILL.md", Content: []byte("new content")},
	}
	if err := InstallSkill(dir, "pm", files); err != nil {
		t.Fatalf("InstallSkill: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if string(data) != "new content" {
		t.Errorf("expected overwrite, got %q", string(data))
	}
}
