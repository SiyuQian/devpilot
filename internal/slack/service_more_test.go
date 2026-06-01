package slack

import (
	"os"
	"testing"

	"github.com/siyuqian/devpilot/internal/auth"
)

func TestServiceStateOAuthConfigAndBotToken(t *testing.T) {
	dir := t.TempDir()
	restore := auth.OverrideConfigDir(dir)
	defer restore()

	svc := NewService()
	if svc.Name() != "slack" {
		t.Fatalf("Name() = %q, want slack", svc.Name())
	}
	if svc.IsLoggedIn() {
		t.Fatalf("IsLoggedIn() = true before credentials")
	}
	if _, err := loadBotToken(); err == nil {
		t.Fatalf("loadBotToken() succeeded before login")
	}
	if err := auth.Save("slack", auth.ServiceCredentials{
		"client_id":     "id",
		"client_secret": "secret",
		"access_token":  "bot",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !svc.IsLoggedIn() {
		t.Fatalf("IsLoggedIn() = false after credentials")
	}
	cfg := svc.oauthConfig()
	if cfg.ClientID != "id" || cfg.ClientSecret != "secret" || !cfg.UseTLS || cfg.RedirectPort != 17321 {
		t.Fatalf("oauthConfig() = %#v", cfg)
	}
	token, err := loadBotToken()
	if err != nil {
		t.Fatalf("loadBotToken() error = %v", err)
	}
	if token != "bot" {
		t.Fatalf("token = %q, want bot", token)
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
