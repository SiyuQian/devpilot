package auth

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOAuthCallbackServerBranches(t *testing.T) {
	listener, srv, resultCh, err := startCallbackServer("state", false, 0)
	if err != nil {
		t.Fatalf("startCallbackServer: %v", err)
	}
	defer func() { _ = srv.Close() }()
	base := "http://" + listener.Addr().String() + "/callback"

	resp, err := http.Get(base + "?state=bad&code=x")
	if err != nil {
		t.Fatalf("invalid state get: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid state status = %d", resp.StatusCode)
	}
	if got := <-resultCh; got.err == nil || !strings.Contains(got.err.Error(), "invalid state") {
		t.Fatalf("invalid state result = %#v", got)
	}

	resp, err = http.Get(base + "?state=state&code=abc")
	if err != nil {
		t.Fatalf("success get: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("success status = %d", resp.StatusCode)
	}
	if got := <-resultCh; got.code != "abc" || got.err != nil {
		t.Fatalf("success result = %#v", got)
	}

	listener, srv, resultCh, err = startCallbackServer("state", false, 0)
	if err != nil {
		t.Fatalf("startCallbackServer second: %v", err)
	}
	defer func() { _ = srv.Close() }()
	resp, err = http.Get("http://" + listener.Addr().String() + "/callback?error=access_denied")
	if err != nil {
		t.Fatalf("denied get: %v", err)
	}
	_ = resp.Body.Close()
	if got := <-resultCh; !errors.Is(got.err, ErrAuthDenied) {
		t.Fatalf("denied result = %#v", got)
	}
}

func TestOAuthURLAndTokenExchange(t *testing.T) {
	authURL := buildAuthURL(OAuthConfig{
		AuthURL:  "https://auth.example/authorize",
		ClientID: "client",
		Scopes:   []string{"a", "b"},
	}, "http://localhost/callback", "state")
	for _, want := range []string{"client_id=client", "redirect_uri=http%3A%2F%2Flocalhost%2Fcallback", "scope=a+b", "state=state"} {
		if !strings.Contains(authURL, want) {
			t.Fatalf("authURL missing %q: %s", want, authURL)
		}
	}

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			_, _ = io.WriteString(w, `{"access_token":"access","refresh_token":"refresh","token_type":"Bearer","expires_in":60}`)
		case "refresh_token":
			_, _ = io.WriteString(w, `{"access_token":"new","token_type":"Bearer"}`)
		default:
			http.Error(w, "bad grant", http.StatusBadRequest)
		}
	}))
	defer tokenServer.Close()

	cfg := OAuthConfig{TokenURL: tokenServer.URL, ClientID: "id", ClientSecret: "secret"}
	token, err := exchangeCode(cfg, "code", "http://localhost/callback")
	if err != nil {
		t.Fatalf("exchangeCode: %v", err)
	}
	if token.AccessToken != "access" || token.RefreshToken != "refresh" || token.Expiry.IsZero() {
		t.Fatalf("token = %#v", token)
	}
	refreshed, err := RefreshToken(cfg, "existing-refresh")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if refreshed.AccessToken != "new" || refreshed.RefreshToken != "existing-refresh" {
		t.Fatalf("refreshed = %#v", refreshed)
	}
}

func TestOAuthTokenErrors(t *testing.T) {
	if _, err := parseTokenResponse([]byte(`{"access_token":"x","expiry":"bad"}`)); err == nil {
		t.Fatalf("parseTokenResponse bad expiry succeeded")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"invalid_grant","error_description":"expired"}`)
	}))
	defer server.Close()
	if _, err := RefreshToken(OAuthConfig{TokenURL: server.URL}, "r"); !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("RefreshToken error = %v, want ErrReauthRequired", err)
	}

	token := &OAuthToken{AccessToken: "a", RefreshToken: "r", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour).UTC()}
	dir := t.TempDir()
	restore := OverrideConfigDir(dir)
	defer restore()
	if err := SaveOAuthToken("svc", token); err != nil {
		t.Fatalf("SaveOAuthToken: %v", err)
	}
	loaded, err := LoadOAuthToken("svc")
	if err != nil {
		t.Fatalf("LoadOAuthToken: %v", err)
	}
	if loaded.AccessToken != "a" || loaded.RefreshToken != "r" {
		t.Fatalf("loaded = %#v", loaded)
	}
}
