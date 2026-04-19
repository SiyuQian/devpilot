package slack

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/siyuqian/devpilot/internal/auth"
)

const (
	slackAuthURL   = "https://slack.com/oauth/v2/authorize"
	slackTokenURL  = "https://slack.com/api/oauth.v2.access"
	slackScopeChat = "chat:write"
	slackScopeRead = "channels:read"
)

func init() {
	auth.Register(NewService())
}

// Service implements the auth.Service interface for Slack.
type Service struct{}

// NewService returns a new Slack auth service.
func NewService() *Service {
	return &Service{}
}

// Name returns the service name ("slack").
func (s *Service) Name() string {
	return "slack"
}

// Login runs the interactive Slack OAuth flow and stores credentials.
func (s *Service) Login() error {
	fmt.Println("Slack Login")
	fmt.Println("===========")
	fmt.Println()
	fmt.Println("To authenticate, you need a Slack App Client ID and Secret:")
	fmt.Println()
	fmt.Println("1. Go to https://api.slack.com/apps")
	fmt.Println("2. Create a new app (or use an existing one)")
	fmt.Println("3. Copy the Client ID and Client Secret from Basic Information")
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
	if err := auth.Save(s.Name(), creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	cfg := s.oauthConfig()
	token, err := auth.StartFlow(cfg)
	if err != nil {
		return fmt.Errorf("slack login failed: %w", err)
	}

	// Preserve client credentials alongside the access token.
	creds["access_token"] = token.AccessToken
	if err := auth.Save(s.Name(), creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println("Logged in to Slack.")
	return nil
}

// Logout removes stored Slack credentials.
func (s *Service) Logout() error {
	if err := auth.Remove(s.Name()); err != nil {
		return err
	}
	fmt.Println("Logged out of Slack.")
	return nil
}

// IsLoggedIn reports whether Slack credentials are stored locally.
func (s *Service) IsLoggedIn() bool {
	_, err := auth.Load(s.Name())
	return err == nil
}

func (s *Service) oauthConfig() auth.OAuthConfig {
	creds, _ := auth.Load(s.Name())
	return auth.OAuthConfig{
		ProviderName: "slack",
		AuthURL:      slackAuthURL,
		TokenURL:     slackTokenURL,
		ClientID:     creds["client_id"],
		ClientSecret: creds["client_secret"],
		Scopes:       []string{slackScopeChat, slackScopeRead},
		UseTLS:       true,
		RedirectPort: 17321,
	}
}

func loadBotToken() (string, error) {
	creds, err := auth.Load("slack")
	if err != nil {
		return "", fmt.Errorf("not logged in to Slack, run: devpilot login slack")
	}
	token, ok := creds["access_token"]
	if !ok || token == "" {
		return "", fmt.Errorf("not logged in to Slack, run: devpilot login slack")
	}
	return token, nil
}

// parseSlackError extracts an error message from a Slack API error response.
func parseSlackError(body []byte) string {
	var resp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	// Unmarshal error is intentionally ignored; if parsing fails we fall back to the raw body.
	if json.Unmarshal(body, &resp) == nil && resp.Error != "" {
		return resp.Error
	}
	return string(body)
}
