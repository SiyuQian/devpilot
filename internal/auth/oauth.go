package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type OAuthConfig struct {
	ProviderName string
	AuthURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
	RedirectPort int // 0 means random available port
}

type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	TokenType    string
}

func SaveOAuthToken(serviceName string, token *OAuthToken) error {
	creds := ServiceCredentials{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_type":    token.TokenType,
	}
	if !token.Expiry.IsZero() {
		creds["expiry"] = token.Expiry.Format(time.RFC3339)
	}
	return Save(serviceName, creds)
}

func LoadOAuthToken(serviceName string) (*OAuthToken, error) {
	creds, err := Load(serviceName)
	if err != nil {
		return nil, err
	}
	token := &OAuthToken{
		AccessToken:  creds["access_token"],
		RefreshToken: creds["refresh_token"],
		TokenType:    creds["token_type"],
	}
	if expiryStr, ok := creds["expiry"]; ok && expiryStr != "" {
		token.Expiry, err = time.Parse(time.RFC3339, expiryStr)
		if err != nil {
			return nil, fmt.Errorf("invalid expiry format: %w", err)
		}
	}
	return token, nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

type callbackResult struct {
	code string
	err  error
}

func startCallbackServer(state string) (net.Listener, *http.Server, <-chan callbackResult, error) {
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><body><h2>Authorization denied.</h2><p>You can close this window.</p></body></html>")
			resultCh <- callbackResult{err: ErrAuthDenied}
			return
		}

		if r.URL.Query().Get("state") != state {
			w.Header().Set("Content-Type", "text/html")
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			resultCh <- callbackResult{err: fmt.Errorf("oauth: invalid state parameter (possible CSRF)")}
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			resultCh <- callbackResult{err: fmt.Errorf("oauth: missing authorization code in callback")}
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body><h2>Authorization successful!</h2><p>You can close this window.</p></body></html>")
		resultCh <- callbackResult{code: code}
	})

	srv := &http.Server{Handler: mux}

	var listener net.Listener
	var err error
	for range 3 {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to bind to a local port after 3 attempts: %w", err)
	}

	go srv.Serve(listener) //nolint:errcheck

	return listener, srv, resultCh, nil
}

func openBrowser(url string) error {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return exec.Command(cmd, url).Start()
}

func buildAuthURL(cfg OAuthConfig, redirectURI, state string) string {
	params := url.Values{
		"client_id":     {cfg.ClientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"state":         {state},
	}
	if len(cfg.Scopes) > 0 {
		params.Set("scope", strings.Join(cfg.Scopes, " "))
	}
	return cfg.AuthURL + "?" + params.Encode()
}

func exchangeCode(cfg OAuthConfig, code, redirectURI string) (*OAuthToken, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
	}

	resp, err := http.PostForm(cfg.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("token exchange failed: %s: %s", errResp.Error, errResp.Description)
		}
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	return parseTokenResponse(body)
}

func parseTokenResponse(body []byte) (*OAuthToken, error) {
	var raw struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		Expiry       string `json:"expiry"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	token := &OAuthToken{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		TokenType:    raw.TokenType,
	}

	if raw.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(raw.ExpiresIn) * time.Second)
	} else if raw.Expiry != "" {
		var err error
		token.Expiry, err = time.Parse(time.RFC3339, raw.Expiry)
		if err != nil {
			return nil, fmt.Errorf("invalid expiry format: %w", err)
		}
	}

	return token, nil
}

func RefreshToken(cfg OAuthConfig, refreshToken string) (*OAuthToken, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
	}

	resp, err := http.PostForm(cfg.TokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			if errResp.Error == "invalid_grant" {
				return nil, ErrReauthRequired
			}
			return nil, fmt.Errorf("token refresh failed: %s: %s", errResp.Error, errResp.Description)
		}
		return nil, fmt.Errorf("token refresh failed with status %d", resp.StatusCode)
	}

	token, err := parseTokenResponse(body)
	if err != nil {
		return nil, err
	}

	// Preserve the original refresh token if the provider didn't issue a new one
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}

	return token, nil
}

func StartFlow(cfg OAuthConfig) (*OAuthToken, error) {
	state, err := generateState()
	if err != nil {
		return nil, err
	}

	listener, srv, resultCh, err := startCallbackServer(state)
	if err != nil {
		return nil, err
	}
	defer srv.Shutdown(context.Background()) //nolint:errcheck

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
	authURL := buildAuthURL(cfg, redirectURI, state)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically.\nPlease open this URL in your browser:\n\n  %s\n\n", authURL)
	}

	select {
	case result := <-resultCh:
		if result.err != nil {
			return nil, result.err
		}
		return exchangeCode(cfg, result.code, redirectURI)
	case <-time.After(2 * time.Minute):
		return nil, fmt.Errorf("oauth: authorization timed out after 2 minutes")
	}
}
