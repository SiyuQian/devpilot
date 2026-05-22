package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrelloVerify_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1/members/me" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "" || r.URL.Query().Get("token") != "" {
			t.Fatalf("credentials must not be in query string: %s", r.URL.RawQuery)
		}
		wantAuth := `OAuth oauth_consumer_key="test-key", oauth_token="test-token"`
		if got := r.Header.Get("Authorization"); got != wantAuth {
			w.WriteHeader(401)
			return
		}
		_, _ = w.Write([]byte(`{"id":"123","fullName":"Test User"}`))
	}))
	defer server.Close()

	svc := &TrelloService{baseURL: server.URL}
	err := svc.verify("test-key", "test-token")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestTrelloVerify_InvalidCreds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()

	svc := &TrelloService{baseURL: server.URL}
	err := svc.verify("bad-key", "bad-token")
	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}
}
