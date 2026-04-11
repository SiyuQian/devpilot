package gmail

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/siyuqian/devpilot/internal/auth"
)

const defaultBaseURL = "https://gmail.googleapis.com"

// Client is a minimal Gmail REST API client with automatic OAuth token refresh.
type Client struct {
	accessToken  string
	refreshToken string
	expiry       time.Time
	baseURL      string
	httpClient   *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the Gmail API base URL (used in tests).
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithHTTPClient overrides the underlying HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient returns a Client authenticated with the given access token.
func NewClient(accessToken string, opts ...Option) *Client {
	c := &Client{
		accessToken: accessToken,
		baseURL:     defaultBaseURL,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewClientFromToken returns a Client seeded with an OAuth token (including
// refresh token and expiry) so that expired access tokens can be refreshed.
func NewClientFromToken(token *auth.OAuthToken, opts ...Option) *Client {
	c := NewClient(token.AccessToken, opts...)
	c.refreshToken = token.RefreshToken
	c.expiry = token.Expiry
	return c
}

// MessageListResponse is the Gmail users.messages.list response envelope.
type MessageListResponse struct {
	Messages      []MessageRef `json:"messages"`
	NextPageToken string       `json:"nextPageToken"`
}

// MessageRef is a lightweight reference to a Gmail message.
type MessageRef struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
}

// Message is a full Gmail message with headers and body payload.
type Message struct {
	ID      string  `json:"id"`
	Payload Payload `json:"payload"`
}

// Payload is a MIME payload tree node for a Gmail message.
type Payload struct {
	Headers  []Header  `json:"headers"`
	Body     Body      `json:"body"`
	Parts    []Payload `json:"parts"`
	MimeType string    `json:"mimeType"`
}

// Header is a single RFC 822 message header.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Body holds the base64url-encoded bytes of a MIME part.
type Body struct {
	Data string `json:"data"`
	Size int    `json:"size"`
}

func (c *Client) refreshIfNeeded() error {
	if c.refreshToken == "" || c.expiry.IsZero() {
		return nil
	}
	if time.Now().Before(c.expiry.Add(-1 * time.Minute)) {
		return nil
	}
	svc := NewService()
	cfg := svc.oauthConfig()
	newToken, err := auth.RefreshToken(cfg, c.refreshToken)
	if err != nil {
		return fmt.Errorf("token refresh failed, run: devpilot login gmail (%w)", err)
	}
	if err := auth.SaveOAuthToken("gmail", newToken); err != nil {
		return fmt.Errorf("failed to save refreshed token: %w", err)
	}
	c.accessToken = newToken.AccessToken
	c.refreshToken = newToken.RefreshToken
	c.expiry = newToken.Expiry
	return nil
}

func (c *Client) doRequest(method, path string, params url.Values) ([]byte, error) {
	if err := c.refreshIfNeeded(); err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized && c.refreshToken != "" {
		svc := NewService()
		cfg := svc.oauthConfig()
		newToken, refreshErr := auth.RefreshToken(cfg, c.refreshToken)
		if refreshErr != nil {
			return nil, fmt.Errorf("not logged in to gmail, run: devpilot login gmail")
		}
		if saveErr := auth.SaveOAuthToken("gmail", newToken); saveErr != nil {
			return nil, fmt.Errorf("failed to save refreshed token: %w", saveErr)
		}
		c.accessToken = newToken.AccessToken
		c.refreshToken = newToken.RefreshToken
		c.expiry = newToken.Expiry

		req2, err := http.NewRequest(method, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create retry request failed: %w", err)
		}
		req2.Header.Set("Authorization", "Bearer "+c.accessToken)
		resp2, err2 := c.httpClient.Do(req2)
		if err2 != nil {
			return nil, fmt.Errorf("retry request failed: %w", err2)
		}
		defer func() { _ = resp2.Body.Close() }()
		body, err = io.ReadAll(resp2.Body)
		if err != nil {
			return nil, fmt.Errorf("read retry body failed: %w", err)
		}
		if resp2.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d: %s", resp2.StatusCode, string(body))
		}
		return body, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) doPost(path string, payload any) ([]byte, error) {
	if err := c.refreshIfNeeded(); err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s%s", c.baseURL, path)

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// ListMessages returns up to limit message references matching the Gmail
// search query. An empty query matches all messages.
func (c *Client) ListMessages(query string, limit int) ([]MessageRef, error) {
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	if limit > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", limit))
	}

	data, err := c.doRequest(http.MethodGet, "/gmail/v1/users/me/messages", params)
	if err != nil {
		return nil, err
	}

	var resp MessageListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse message list: %w", err)
	}
	return resp.Messages, nil
}

// ListAllMessageIDs returns every message ID matching the Gmail search query,
// paginating through all result pages.
func (c *Client) ListAllMessageIDs(query string) ([]string, error) {
	var allIDs []string
	pageToken := ""
	for {
		params := url.Values{}
		if query != "" {
			params.Set("q", query)
		}
		params.Set("maxResults", "500")
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}

		data, err := c.doRequest(http.MethodGet, "/gmail/v1/users/me/messages", params)
		if err != nil {
			return nil, err
		}

		var resp MessageListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parse message list: %w", err)
		}
		for _, m := range resp.Messages {
			allIDs = append(allIDs, m.ID)
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return allIDs, nil
}

// GetMessage fetches the full message payload for the given message ID.
func (c *Client) GetMessage(id string) (*Message, error) {
	params := url.Values{"format": {"full"}}
	data, err := c.doRequest(http.MethodGet, fmt.Sprintf("/gmail/v1/users/me/messages/%s", id), params)
	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("parse message: %w", err)
	}
	return &msg, nil
}

// BatchModify removes the given label IDs from each of the given message IDs.
// Gmail supports up to 1000 message IDs per call.
func (c *Client) BatchModify(ids []string, removeLabelIDs []string) error {
	payload := map[string]any{
		"ids":            ids,
		"removeLabelIds": removeLabelIDs,
	}
	_, err := c.doPost("/gmail/v1/users/me/messages/batchModify", payload)
	return err
}

// GetHeader extracts a header value from a message by name.
func GetHeader(msg *Message, name string) string {
	for _, h := range msg.Payload.Headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

// GetBody extracts the plain text body from a message.
// Falls back to stripping HTML tags if no text/plain part is found.
func GetBody(msg *Message) string {
	if text := findTextPart(&msg.Payload, "text/plain"); text != "" {
		return text
	}
	if html := findTextPart(&msg.Payload, "text/html"); html != "" {
		return stripHTML(html)
	}
	if msg.Payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		if err == nil {
			return string(decoded)
		}
	}
	return ""
}

func findTextPart(payload *Payload, mimeType string) string {
	if payload.MimeType == mimeType && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			return string(decoded)
		}
	}
	for i := range payload.Parts {
		if text := findTextPart(&payload.Parts[i], mimeType); text != "" {
			return text
		}
	}
	return ""
}

var htmlTagRegexp = regexp.MustCompile(`<[^>]*>`)

func stripHTML(html string) string {
	text := htmlTagRegexp.ReplaceAllString(html, "")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	return strings.TrimSpace(text)
}
