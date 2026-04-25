package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateState(t *testing.T) {
	state1, err := generateState()
	if err != nil {
		t.Fatalf("generateState() error: %v", err)
	}
	if len(state1) != 32 {
		t.Errorf("expected 32 char hex string, got %d chars: %s", len(state1), state1)
	}

	state2, err := generateState()
	if err != nil {
		t.Fatalf("generateState() error: %v", err)
	}
	if state1 == state2 {
		t.Error("two generated states should be different")
	}
}

func callbackPort(listener net.Listener) int {
	return listener.Addr().(*net.TCPAddr).Port
}

func TestCallbackServerValidState(t *testing.T) {
	state := "test-state-123"
	listener, srv, resultCh, err := startCallbackServer(state, false, 0)
	if err != nil {
		t.Fatalf("startCallbackServer() error: %v", err)
	}
	defer func() { _ = srv.Close() }()

	port := callbackPort(listener)
	url := fmt.Sprintf("http://localhost:%d/callback?code=authcode123&state=%s", port, state)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET callback error: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	result := <-resultCh
	if result.err != nil {
		t.Errorf("unexpected error: %v", result.err)
	}
	if result.code != "authcode123" {
		t.Errorf("expected code 'authcode123', got %q", result.code)
	}
}

func TestCallbackServerInvalidState(t *testing.T) {
	state := "correct-state"
	listener, srv, resultCh, err := startCallbackServer(state, false, 0)
	if err != nil {
		t.Fatalf("startCallbackServer() error: %v", err)
	}
	defer func() { _ = srv.Close() }()

	port := callbackPort(listener)
	url := fmt.Sprintf("http://localhost:%d/callback?code=authcode123&state=wrong-state", port)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET callback error: %v", err)
	}
	_ = resp.Body.Close()

	result := <-resultCh
	if result.err == nil {
		t.Error("expected error for invalid state, got nil")
	}
}

func TestCallbackServerDeniedAuth(t *testing.T) {
	state := "test-state"
	listener, srv, resultCh, err := startCallbackServer(state, false, 0)
	if err != nil {
		t.Fatalf("startCallbackServer() error: %v", err)
	}
	defer func() { _ = srv.Close() }()

	port := callbackPort(listener)
	url := fmt.Sprintf("http://localhost:%d/callback?error=access_denied", port)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET callback error: %v", err)
	}
	_ = resp.Body.Close()

	result := <-resultCh
	if result.err != ErrAuthDenied {
		t.Errorf("expected ErrAuthDenied, got %v", result.err)
	}
}

func TestCallbackServerTLS(t *testing.T) {
	state := "tls-test-state"
	listener, srv, resultCh, err := startCallbackServer(state, true, 0)
	if err != nil {
		t.Fatalf("startCallbackServer(TLS) error: %v", err)
	}
	defer func() { _ = srv.Close() }()

	port := callbackPort(listener)
	url := fmt.Sprintf("https://localhost:%d/callback?code=tlscode&state=%s", port, state)

	// Use a client that skips TLS verification (self-signed cert)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET TLS callback error: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	result := <-resultCh
	if result.err != nil {
		t.Errorf("unexpected error: %v", result.err)
	}
	if result.code != "tlscode" {
		t.Errorf("expected code 'tlscode', got %q", result.code)
	}
}

func TestCallbackServerDoubleInvocation(t *testing.T) {
	// Regression test for issue #77: a second request to /callback must not
	// block on the buffered resultCh, otherwise srv.Shutdown deadlocks
	// waiting for the in-flight handler to return.
	tests := []struct {
		name        string
		secondQuery string
	}{
		{name: "second_success", secondQuery: "code=second&state=double-invoke"},
		{name: "second_invalid_state", secondQuery: "code=second&state=wrong"},
		{name: "second_denied", secondQuery: "error=access_denied"},
		{name: "second_missing_code", secondQuery: "state=double-invoke"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := "double-invoke"
			listener, srv, resultCh, err := startCallbackServer(state, false, 0)
			if err != nil {
				t.Fatalf("startCallbackServer() error: %v", err)
			}

			port := callbackPort(listener)
			client := &http.Client{Timeout: 2 * time.Second}

			// First request: a valid success that should populate resultCh.
			firstURL := fmt.Sprintf("http://localhost:%d/callback?code=first&state=%s", port, state)
			resp, err := client.Get(firstURL)
			if err != nil {
				t.Fatalf("first GET error: %v", err)
			}
			_ = resp.Body.Close()

			// Drain the result channel as StartFlow would.
			select {
			case result := <-resultCh:
				if result.err != nil {
					t.Fatalf("first call: unexpected error: %v", result.err)
				}
				if result.code != "first" {
					t.Fatalf("first call: expected code 'first', got %q", result.code)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("first call: timed out waiting for resultCh")
			}

			// Second request: must complete within the client timeout instead
			// of blocking forever on the buffered channel send.
			secondURL := fmt.Sprintf("http://localhost:%d/callback?%s", port, tc.secondQuery)
			resp2, err := client.Get(secondURL)
			if err != nil {
				t.Fatalf("second GET (should not block): %v", err)
			}
			_ = resp2.Body.Close()
			if resp2.StatusCode != http.StatusOK && resp2.StatusCode != http.StatusBadRequest {
				t.Errorf("second call: expected 200 or 400, got %d", resp2.StatusCode)
			}

			// Shutdown must not deadlock. Use a tight context so a regression
			// (handler still blocked on send) surfaces as a context deadline.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				t.Fatalf("srv.Shutdown blocked or errored: %v", err)
			}
		})
	}
}

func TestExchangeCode(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm error: %v", err)
		}
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type=authorization_code, got %s", r.FormValue("grant_type"))
		}
		if r.FormValue("code") != "testcode" {
			t.Errorf("expected code=testcode, got %s", r.FormValue("code"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "access123",
			"refresh_token": "refresh456",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	cfg := OAuthConfig{
		ProviderName: "test",
		TokenURL:     tokenServer.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}

	token, err := exchangeCode(cfg, "testcode", "http://localhost:8080/callback")
	if err != nil {
		t.Fatalf("exchangeCode() error: %v", err)
	}
	if token.AccessToken != "access123" {
		t.Errorf("expected access_token 'access123', got %q", token.AccessToken)
	}
	if token.RefreshToken != "refresh456" {
		t.Errorf("expected refresh_token 'refresh456', got %q", token.RefreshToken)
	}
	if token.TokenType != "Bearer" {
		t.Errorf("expected token_type 'Bearer', got %q", token.TokenType)
	}
	if token.Expiry.IsZero() {
		t.Error("expected non-zero expiry")
	}
}

func TestExchangeCodeError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "code expired",
		})
	}))
	defer tokenServer.Close()

	cfg := OAuthConfig{
		ProviderName: "test",
		TokenURL:     tokenServer.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}

	_, err := exchangeCode(cfg, "badcode", "http://localhost:8080/callback")
	if err == nil {
		t.Error("expected error for invalid_grant, got nil")
	}
}

func TestRefreshToken(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm error: %v", err)
		}
		if r.FormValue("grant_type") != "refresh_token" {
			t.Errorf("expected grant_type=refresh_token, got %s", r.FormValue("grant_type"))
		}
		if r.FormValue("refresh_token") != "refresh456" {
			t.Errorf("expected refresh_token=refresh456, got %s", r.FormValue("refresh_token"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "new-access789",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	cfg := OAuthConfig{
		ProviderName: "test",
		TokenURL:     tokenServer.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}

	token, err := RefreshToken(cfg, "refresh456")
	if err != nil {
		t.Fatalf("RefreshToken() error: %v", err)
	}
	if token.AccessToken != "new-access789" {
		t.Errorf("expected access_token 'new-access789', got %q", token.AccessToken)
	}
	if token.RefreshToken != "refresh456" {
		t.Errorf("expected preserved refresh_token 'refresh456', got %q", token.RefreshToken)
	}
}

func TestRefreshTokenExpired(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "Token has been expired or revoked",
		})
	}))
	defer tokenServer.Close()

	cfg := OAuthConfig{
		ProviderName: "test",
		TokenURL:     tokenServer.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}

	_, err := RefreshToken(cfg, "expired-refresh")
	if err != ErrReauthRequired {
		t.Errorf("expected ErrReauthRequired, got %v", err)
	}
}

func TestGenerateSelfSignedCert(t *testing.T) {
	tlsCert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatalf("generateSelfSignedCert() error: %v", err)
	}
	if len(tlsCert.Certificate) == 0 {
		t.Fatal("expected at least one certificate")
	}

	// Parse the leaf cert and verify it covers localhost
	leaf, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate error: %v", err)
	}
	if err := leaf.VerifyHostname("localhost"); err != nil {
		t.Errorf("cert should be valid for localhost: %v", err)
	}
	if leaf.NotAfter.Before(time.Now()) {
		t.Error("cert should not already be expired")
	}
}

func TestOAuthTokenCredentialsConversion(t *testing.T) {
	dir := t.TempDir()
	origConfigDir := configDir
	configDir = func() string { return dir }
	defer func() { configDir = origConfigDir }()

	expiry := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	original := &OAuthToken{
		AccessToken:  "access123",
		RefreshToken: "refresh456",
		Expiry:       expiry,
		TokenType:    "Bearer",
	}

	if err := SaveOAuthToken("testservice", original); err != nil {
		t.Fatalf("SaveOAuthToken() error: %v", err)
	}

	loaded, err := LoadOAuthToken("testservice")
	if err != nil {
		t.Fatalf("LoadOAuthToken() error: %v", err)
	}

	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken: got %q, want %q", loaded.AccessToken, original.AccessToken)
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken: got %q, want %q", loaded.RefreshToken, original.RefreshToken)
	}
	if loaded.TokenType != original.TokenType {
		t.Errorf("TokenType: got %q, want %q", loaded.TokenType, original.TokenType)
	}
	if !loaded.Expiry.Equal(original.Expiry) {
		t.Errorf("Expiry: got %v, want %v", loaded.Expiry, original.Expiry)
	}
}
