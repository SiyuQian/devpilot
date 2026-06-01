package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type testService struct {
	name string
}

func TestTrelloServiceLoginRejectsEmptyInput(t *testing.T) {
	old := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.WriteString("\n\n"); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	_ = w.Close()
	os.Stdin = r
	defer func() { os.Stdin = old }()

	if err := (&TrelloService{}).Login(); err == nil {
		t.Fatalf("Login() succeeded with empty input")
	}
}

func (s testService) Name() string     { return s.name }
func (s testService) Login() error     { return nil }
func (s testService) Logout() error    { return nil }
func (s testService) IsLoggedIn() bool { return true }

func TestRegistryGetAndAvailableNames(t *testing.T) {
	old := registry
	registry = map[string]Service{}
	defer func() { registry = old }()

	Register(testService{name: "zeta"})
	Register(testService{name: "alpha"})
	svc, err := Get("alpha")
	if err != nil {
		t.Fatalf("Get(alpha) error = %v", err)
	}
	if svc.Name() != "alpha" {
		t.Fatalf("service name = %q, want alpha", svc.Name())
	}
	if names := AvailableNames(); !strings.Contains(names, "alpha") || !strings.Contains(names, "zeta") {
		t.Fatalf("AvailableNames() = %q", names)
	}
	if _, err := Get("missing"); err == nil {
		t.Fatalf("Get(missing) succeeded, want error")
	}
}

func TestTrelloServiceLoginStateAndVerify(t *testing.T) {
	dir := t.TempDir()
	restore := OverrideConfigDir(dir)
	defer restore()

	svc := &TrelloService{}
	if svc.Name() != "trello" {
		t.Fatalf("Name() = %q, want trello", svc.Name())
	}
	if svc.IsLoggedIn() {
		t.Fatalf("IsLoggedIn() = true before credentials")
	}
	if err := Save("trello", ServiceCredentials{"api_key": "k", "token": "t"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !svc.IsLoggedIn() {
		t.Fatalf("IsLoggedIn() = false after credentials")
	}
	if err := svc.Logout(); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if svc.IsLoggedIn() {
		t.Fatalf("IsLoggedIn() = true after logout")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1/members/me" {
			http.Error(w, "missing", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); !strings.Contains(got, `oauth_consumer_key="k"`) || !strings.Contains(got, `oauth_token="t"`) {
			t.Fatalf("Authorization = %q", got)
		}
	}))
	defer server.Close()

	svc.baseURL = server.URL
	if err := svc.verify("k", "t"); err != nil {
		t.Fatalf("verify() error = %v", err)
	}
	svc.baseURL = server.URL + "/missing"
	if err := svc.verify("k", "t"); err == nil {
		t.Fatalf("verify() succeeded for non-200 response")
	}
}
