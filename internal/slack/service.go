package slack

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/siyuqian/devpilot/internal/auth"
)

const (
	slackAuthURL   = "https://slack.com/oauth/v2/authorize"
	slackTokenURL  = "https://slack.com/api/oauth.v2.access"
	slackScopeChat = "chat:write"
	slackScopeRead = "channels:read"
)

func init() {
	auth.Register(NewSlackService())
}

type SlackService struct{}

func NewSlackService() *SlackService {
	return &SlackService{}
}

func (s *SlackService) Name() string {
	return "slack"
}

func (s *SlackService) Login() error {
	cfg := s.oauthConfig()
	token, err := auth.StartFlow(cfg)
	if err != nil {
		return fmt.Errorf("slack login failed: %w", err)
	}

	// Slack OAuth V2 returns bot token and workspace info in the token response.
	// We need to re-exchange the code to get the full response, but StartFlow
	// already parsed it. The access_token is the bot token (xoxb-...).
	// We also need team_id and team_name from the raw response.
	// Since StartFlow only returns the standard OAuthToken, we store what we have
	// and parse additional fields from a second approach.
	//
	// Actually, Slack's oauth.v2.access response includes access_token at the top level,
	// which StartFlow's parseTokenResponse will capture. We save the bot token directly.
	creds := auth.ServiceCredentials{
		"access_token": token.AccessToken,
	}
	if err := auth.Save(s.Name(), creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println("Logged in to Slack.")
	return nil
}

func (s *SlackService) Logout() error {
	if err := auth.Remove(s.Name()); err != nil {
		return err
	}
	fmt.Println("Logged out of Slack.")
	return nil
}

func (s *SlackService) IsLoggedIn() bool {
	_, err := auth.Load(s.Name())
	return err == nil
}

func (s *SlackService) oauthConfig() auth.OAuthConfig {
	return auth.OAuthConfig{
		ProviderName: "slack",
		AuthURL:      slackAuthURL,
		TokenURL:     slackTokenURL,
		ClientID:     os.Getenv("SLACK_CLIENT_ID"),
		ClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
		Scopes:       []string{slackScopeChat, slackScopeRead},
		UseTLS:       true,
		RedirectPort: 17321,
	}
}

func loadBotToken() (string, error) {
	creds, err := auth.Load("slack")
	if err != nil {
		return "", fmt.Errorf("Not logged in to Slack. Run: devpilot login slack")
	}
	token, ok := creds["access_token"]
	if !ok || token == "" {
		return "", fmt.Errorf("Not logged in to Slack. Run: devpilot login slack")
	}
	return token, nil
}

// parseSlackError extracts an error message from a Slack API error response.
func parseSlackError(body []byte) string {
	var resp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &resp) == nil && resp.Error != "" {
		return resp.Error
	}
	return string(body)
}
