package gmail

import (
	"fmt"

	"github.com/siyuqian/devpilot/internal/auth"
)

const (
	gmailClientID     = "xxxx.apps.googleusercontent.com"
	gmailClientSecret = "GOCSPX-xxxx"
	gmailAuthURL      = "https://accounts.google.com/o/oauth2/v2/auth"
	gmailTokenURL     = "https://oauth2.googleapis.com/token"
	gmailScope        = "https://www.googleapis.com/auth/gmail.modify"
)

func init() {
	auth.Register(NewGmailService())
}

type GmailService struct{}

func NewGmailService() *GmailService {
	return &GmailService{}
}

func (g *GmailService) Name() string {
	return "gmail"
}

func (g *GmailService) Login() error {
	cfg := g.oauthConfig()
	token, err := auth.StartFlow(cfg)
	if err != nil {
		return fmt.Errorf("gmail login failed: %w", err)
	}
	if err := auth.SaveOAuthToken(g.Name(), token); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}
	fmt.Println("Logged in to Gmail.")
	return nil
}

func (g *GmailService) Logout() error {
	if err := auth.Remove(g.Name()); err != nil {
		return err
	}
	fmt.Println("Logged out of Gmail.")
	return nil
}

func (g *GmailService) IsLoggedIn() bool {
	_, err := auth.Load(g.Name())
	return err == nil
}

func (g *GmailService) oauthConfig() auth.OAuthConfig {
	return auth.OAuthConfig{
		ProviderName: "gmail",
		AuthURL:      gmailAuthURL,
		TokenURL:     gmailTokenURL,
		ClientID:     gmailClientID,
		ClientSecret: gmailClientSecret,
		Scopes:       []string{gmailScope},
	}
}
