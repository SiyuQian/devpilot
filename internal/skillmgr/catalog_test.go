package skillmgr

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"
)

func skillMDBase64(content string) string {
	return base64.StdEncoding.EncodeToString([]byte(content))
}

func TestFetchCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/contents/skills":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `[
				{"type":"dir","name":"pm"},
				{"type":"dir","name":"trello"},
				{"type":"file","name":"README.md"}
			]`)
		case "/repos/o/r/contents/skills/pm/SKILL.md":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"content":"%s","encoding":"base64"}`,
				skillMDBase64("---\nname: pm\ndescription: Product manager skill\n---\n# PM"))
		case "/repos/o/r/contents/skills/trello/SKILL.md":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"content":"%s","encoding":"base64"}`,
				skillMDBase64("---\nname: devpilot:trello\ndescription: Trello integration\n---\n# Trello"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	catalog, err := fetchCatalogFromBase(ctx, srv.URL+"/repos/o/r", "v1.0.0")
	if err != nil {
		t.Fatalf("FetchCatalog: %v", err)
	}

	if len(catalog) != 2 {
		t.Fatalf("len(catalog) = %d, want 2", len(catalog))
	}

	sort.Slice(catalog, func(i, j int) bool { return catalog[i].Name < catalog[j].Name })

	if got, want := catalog[0].Name, "pm"; got != want {
		t.Errorf("catalog[0].Name = %q, want %q", got, want)
	}
	if got, want := catalog[0].Description, "Product manager skill"; got != want {
		t.Errorf("catalog[0].Description = %q, want %q", got, want)
	}
	if got, want := catalog[1].Name, "trello"; got != want {
		t.Errorf("catalog[1].Name = %q, want %q", got, want)
	}
	if got, want := catalog[1].Description, "Trello integration"; got != want {
		t.Errorf("catalog[1].Description = %q, want %q", got, want)
	}
}

func TestFetchCatalogSkipsFailedSkills(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/contents/skills":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `[
				{"type":"dir","name":"pm"},
				{"type":"dir","name":"broken"}
			]`)
		case "/repos/o/r/contents/skills/pm/SKILL.md":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"content":"%s","encoding":"base64"}`,
				skillMDBase64("---\nname: pm\ndescription: PM\n---"))
		case "/repos/o/r/contents/skills/broken/SKILL.md":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	catalog, err := fetchCatalogFromBase(ctx, srv.URL+"/repos/o/r", "v1.0.0")
	if err != nil {
		t.Fatalf("FetchCatalog: %v", err)
	}

	if len(catalog) != 1 {
		t.Fatalf("len(catalog) = %d, want 1 (broken should be skipped)", len(catalog))
	}
}

func TestFetchCatalogRespectsTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/contents/skills":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `[{"type":"dir","name":"slow"}]`)
		case "/repos/o/r/contents/skills/slow/SKILL.md":
			time.Sleep(2 * time.Second)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"content":"%s","encoding":"base64"}`,
				skillMDBase64("---\nname: slow\ndescription: Slow\n---"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	catalog, err := fetchCatalogFromBase(ctx, srv.URL+"/repos/o/r", "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	// The slow skill should be skipped due to timeout
	if len(catalog) != 0 {
		t.Errorf("expected empty catalog due to timeout, got %d entries", len(catalog))
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
