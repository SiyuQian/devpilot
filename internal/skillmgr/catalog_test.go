package skillmgr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"
)

func TestFetchCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/o/r/v1.0.0/skills/index.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"skills":[
			{"name":"pm","description":"Product manager skill","files":["SKILL.md"]},
			{"name":"trello","description":"Trello integration","files":["SKILL.md"]}
		]}`))
	}))
	defer srv.Close()

	origBase := rawBaseURL
	defer func() { setRawBaseURL(origBase) }()
	setRawBaseURL(srv.URL)

	ctx := context.Background()
	catalog, err := FetchCatalog(ctx, "o", "r", "v1.0.0")
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
}

func TestFetchCatalogSkipsEmptyNames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"skills":[
			{"name":"pm","description":"PM","files":["SKILL.md"]},
			{"name":"","description":"broken","files":[]}
		]}`))
	}))
	defer srv.Close()

	origBase := rawBaseURL
	defer func() { setRawBaseURL(origBase) }()
	setRawBaseURL(srv.URL)

	catalog, err := FetchCatalog(context.Background(), "o", "r", "v1.0.0")
	if err != nil {
		t.Fatalf("FetchCatalog: %v", err)
	}
	if len(catalog) != 1 {
		t.Fatalf("len(catalog) = %d, want 1", len(catalog))
	}
}

func TestFetchCatalogRespectsTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"skills":[{"name":"slow","description":"Slow","files":["SKILL.md"]}]}`))
	}))
	defer srv.Close()

	origBase := rawBaseURL
	defer func() { setRawBaseURL(origBase) }()
	setRawBaseURL(srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := FetchCatalog(ctx, "o", "r", "v1.0.0")
	if err == nil {
		t.Fatal("expected error due to timeout, got nil")
	}
}
