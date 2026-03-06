package auth

import (
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
	listener, srv, resultCh, err := startCallbackServer(state)
	if err != nil {
		t.Fatalf("startCallbackServer() error: %v", err)
	}
	defer srv.Close()

	port := callbackPort(listener)
	url := fmt.Sprintf("http://localhost:%d/callback?code=authcode123&state=%s", port, state)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET callback error: %v", err)
	}
	resp.Body.Close()

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
	listener, srv, resultCh, err := startCallbackServer(state)
	if err != nil {
		t.Fatalf("startCallbackServer() error: %v", err)
	}
	defer srv.Close()

	port := callbackPort(listener)
	url := fmt.Sprintf("http://localhost:%d/callback?code=authcode123&state=wrong-state", port)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET callback error: %v", err)
	}
	resp.Body.Close()

	result := <-resultCh
	if result.err == nil {
		t.Error("expected error for invalid state, got nil")
	}
}

func TestCallbackServerDeniedAuth(t *testing.T) {
	state := "test-state"
	listener, srv, resultCh, err := startCallbackServer(state)
	if err != nil {
		t.Fatalf("startCallbackServer() error: %v", err)
	}
	defer srv.Close()

	port := callbackPort(listener)
	url := fmt.Sprintf("http://localhost:%d/callback?error=access_denied", port)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET callback error: %v", err)
	}
	resp.Body.Close()

	result := <-resultCh
	if result.err != ErrAuthDenied {
		t.Errorf("expected ErrAuthDenied, got %v", result.err)
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
		json.NewEncoder(w).Encode(map[string]interface{}{
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
		json.NewEncoder(w).Encode(map[string]string{
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
		json.NewEncoder(w).Encode(map[string]interface{}{
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
		json.NewEncoder(w).Encode(map[string]string{
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
