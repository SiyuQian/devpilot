package gmail

import (
	"os"
	"testing"
	"time"

	"github.com/siyuqian/devpilot/internal/auth"
)

func TestServiceStateAndOAuthConfig(t *testing.T) {
	dir := t.TempDir()
	restore := auth.OverrideConfigDir(dir)
	defer restore()

	svc := NewService()
	if svc.Name() != "gmail" {
		t.Fatalf("Name() = %q, want gmail", svc.Name())
	}
	if svc.IsLoggedIn() {
		t.Fatalf("IsLoggedIn() = true before credentials")
	}
	if err := auth.Save("gmail", auth.ServiceCredentials{
		"client_id":     "id",
		"client_secret": "secret",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !svc.IsLoggedIn() {
		t.Fatalf("IsLoggedIn() = false after credentials")
	}
	cfg := svc.oauthConfig()
	if cfg.ClientID != "id" || cfg.ClientSecret != "secret" || cfg.AuthURL == "" || cfg.TokenURL == "" {
		t.Fatalf("oauthConfig() = %#v", cfg)
	}
	if len(cfg.Scopes) != 1 || cfg.Scopes[0] != gmailScope {
		t.Fatalf("scopes = %v", cfg.Scopes)
	}
	if err := svc.Logout(); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if svc.IsLoggedIn() {
		t.Fatalf("IsLoggedIn() = true after logout")
	}
}

func TestServiceLoginRejectsEmptyInput(t *testing.T) {
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

	if err := NewService().Login(); err == nil {
		t.Fatalf("Login() succeeded with empty input")
	}
}

func TestNewClientFromTokenAndOptions(t *testing.T) {
	token := &auth.OAuthToken{
		AccessToken:  "access",
		RefreshToken: "refresh",
		Expiry:       time.Now().Add(time.Hour),
	}
	client := NewClientFromToken(token, WithBaseURL("https://example.com"))
	if client.accessToken != "access" || client.refreshToken != "refresh" || client.baseURL != "https://example.com" {
		t.Fatalf("client not initialized from token: %#v", client)
	}
}
