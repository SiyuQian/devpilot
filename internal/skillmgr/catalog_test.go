package skillmgr

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

func TestFetchCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/contents/.claude/skills":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[
				{"type":"dir","name":"pm"},
				{"type":"dir","name":"trello"},
				{"type":"dir","name":"openspec-explore"},
				{"type":"file","name":"README.md"}
			]`)
		case "/repos/o/r/contents/.claude/skills/pm/SKILL.md":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"download_url":"http://%s/raw/pm"}`, r.Host)
		case "/repos/o/r/contents/.claude/skills/trello/SKILL.md":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"download_url":"http://%s/raw/trello"}`, r.Host)
		case "/raw/pm":
			fmt.Fprint(w, "---\nname: pm\ndescription: Product manager skill\n---\n# PM")
		case "/raw/trello":
			fmt.Fprint(w, "---\nname: devpilot:trello\ndescription: Trello integration\n---\n# Trello")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	catalog, err := fetchCatalogFromBase(srv.URL+"/repos/o/r", "v1.0.0")
	if err != nil {
		t.Fatalf("FetchCatalog: %v", err)
	}

	if len(catalog) != 2 {
		t.Fatalf("len(catalog) = %d, want 2", len(catalog))
	}

	sort.Slice(catalog, func(i, j int) bool { return catalog[i].Name < catalog[j].Name })

	if catalog[0].Name != "pm" || catalog[0].Description != "Product manager skill" {
		t.Errorf("catalog[0] = %+v", catalog[0])
	}
	if catalog[1].Name != "trello" || catalog[1].Description != "Trello integration" {
		t.Errorf("catalog[1] = %+v", catalog[1])
	}
}

func TestFetchCatalogExcludesOpenspec(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/contents/.claude/skills":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[
				{"type":"dir","name":"pm"},
				{"type":"dir","name":"openspec-apply-change"},
				{"type":"dir","name":"openspec-archive-change"},
				{"type":"dir","name":"openspec-explore"},
				{"type":"dir","name":"openspec-propose"}
			]`)
		case "/repos/o/r/contents/.claude/skills/pm/SKILL.md":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"download_url":"http://%s/raw/pm"}`, r.Host)
		case "/raw/pm":
			fmt.Fprint(w, "---\nname: pm\ndescription: PM skill\n---")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	catalog, err := fetchCatalogFromBase(srv.URL+"/repos/o/r", "v1.0.0")
	if err != nil {
		t.Fatalf("FetchCatalog: %v", err)
	}

	if len(catalog) != 1 {
		t.Fatalf("len(catalog) = %d, want 1 (openspec-* should be excluded)", len(catalog))
	}
	if catalog[0].Name != "pm" {
		t.Errorf("expected pm, got %q", catalog[0].Name)
	}
}

func TestFetchCatalogSkipsFailedSkills(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/contents/.claude/skills":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[
				{"type":"dir","name":"pm"},
				{"type":"dir","name":"broken"}
			]`)
		case "/repos/o/r/contents/.claude/skills/pm/SKILL.md":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"download_url":"http://%s/raw/pm"}`, r.Host)
		case "/raw/pm":
			fmt.Fprint(w, "---\nname: pm\ndescription: PM\n---")
		case "/repos/o/r/contents/.claude/skills/broken/SKILL.md":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	catalog, err := fetchCatalogFromBase(srv.URL+"/repos/o/r", "v1.0.0")
	if err != nil {
		t.Fatalf("FetchCatalog: %v", err)
	}

	if len(catalog) != 1 {
		t.Fatalf("len(catalog) = %d, want 1 (broken should be skipped)", len(catalog))
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantFM  skillFrontmatter
		wantErr bool
	}{
		{
			name:   "basic",
			input:  "---\nname: pm\ndescription: Product manager\n---\n# Body",
			wantFM: skillFrontmatter{Name: "pm", Description: "Product manager"},
		},
		{
			name:   "multiline description",
			input:  "---\nname: learn\ndescription: >\n  Summarize articles\n  into HTML\n---\n",
			wantFM: skillFrontmatter{Name: "learn", Description: "Summarize articles into HTML"},
		},
		{
			name:    "no frontmatter",
			input:   "# Just a markdown file",
			wantErr: true,
		},
		{
			name:    "unterminated",
			input:   "---\nname: pm\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, err := parseFrontmatter([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fm.Name != tt.wantFM.Name {
				t.Errorf("name = %q, want %q", fm.Name, tt.wantFM.Name)
			}
			if fm.Description != tt.wantFM.Description {
				t.Errorf("description = %q, want %q", fm.Description, tt.wantFM.Description)
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"openspec-explore", true},
		{"openspec-apply-change", true},
		{"pm", false},
		{"trello", false},
		{"my-openspec-thing", false}, // prefix match only
	}
	for _, tt := range tests {
		if got := isExcluded(tt.name); got != tt.want {
			t.Errorf("isExcluded(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
