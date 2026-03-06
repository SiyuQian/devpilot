package gmail

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/siyuqian/devpilot/internal/auth"
)

const (
	gmailAuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
	gmailTokenURL = "https://oauth2.googleapis.com/token"
	gmailScope   = "https://www.googleapis.com/auth/gmail.modify"
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
	fmt.Println("Gmail Login")
	fmt.Println("===========")
	fmt.Println()
	fmt.Println("To authenticate, you need a Google OAuth Client ID and Secret:")
	fmt.Println()
	fmt.Println("1. Go to https://console.cloud.google.com/apis/credentials")
	fmt.Println("2. Create an OAuth 2.0 Client ID (type: Desktop app)")
	fmt.Println("3. Copy the Client ID and Client Secret")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Client ID: ")
	clientID, _ := reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)

	fmt.Print("Client Secret: ")
	clientSecret, _ := reader.ReadString('\n')
	clientSecret = strings.TrimSpace(clientSecret)

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("both Client ID and Client Secret are required")
	}

	// Save client credentials first so oauthConfig() can read them.
	creds := auth.ServiceCredentials{
		"client_id":     clientID,
		"client_secret": clientSecret,
	}
	if err := auth.Save(g.Name(), creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

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
	creds, _ := auth.Load(g.Name())
	return auth.OAuthConfig{
		ProviderName: "gmail",
		AuthURL:      gmailAuthURL,
		TokenURL:     gmailTokenURL,
		ClientID:     creds["client_id"],
		ClientSecret: creds["client_secret"],
		Scopes:       []string{gmailScope},
	}
}
