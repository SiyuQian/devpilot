// Package slack provides a minimal Slack Web API client and the
// devpilot CLI commands that drive it.
package slack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://slack.com/api"

// Client is a minimal Slack Web API client.
type Client struct {
	botToken   string
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the Slack API base URL. It is primarily used in tests.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// WithHTTPClient overrides the HTTP client used to make requests.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient returns a Client that authenticates with the given Slack bot token.
func NewClient(botToken string, opts ...Option) *Client {
	c := &Client{
		botToken:   botToken,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Channel represents a Slack conversation (channel or DM).
type Channel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type conversationsListResponse struct {
	OK               bool      `json:"ok"`
	Error            string    `json:"error"`
	Channels         []Channel `json:"channels"`
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

type postMessageResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

type conversationsOpenResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Channel struct {
		ID string `json:"id"`
	} `json:"channel"`
}

// ListConversations returns all public, non-archived channels visible to the bot.
func (c *Client) ListConversations() ([]Channel, error) {
	var all []Channel
	cursor := ""

	for {
		params := url.Values{
			"types":            {"public_channel"},
			"exclude_archived": {"true"},
			"limit":            {"200"},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		body, err := c.doGet("/conversations.list", params)
		if err != nil {
			return nil, err
		}

		var resp conversationsListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse conversations.list response: %w", err)
		}
		if !resp.OK {
			return nil, fmt.Errorf("conversations.list failed: %s", resp.Error)
		}

		all = append(all, resp.Channels...)

		if resp.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = resp.ResponseMetadata.NextCursor
	}

	return all, nil
}

// ResolveChannel looks up a channel by name (with or without a leading "#")
// and returns its Slack channel ID.
func (c *Client) ResolveChannel(name string) (string, error) {
	name = strings.TrimPrefix(name, "#")

	channels, err := c.ListConversations()
	if err != nil {
		return "", err
	}

	for _, ch := range channels {
		if ch.Name == name {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("channel not found: %q", name)
}

// OpenConversation opens a direct-message channel with the given user ID and
// returns the resulting channel ID.
func (c *Client) OpenConversation(userID string) (string, error) {
	params := url.Values{
		"users": {userID},
	}

	body, err := c.doPost("/conversations.open", params)
	if err != nil {
		return "", err
	}

	var resp conversationsOpenResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse conversations.open response: %w", err)
	}
	if !resp.OK {
		return "", fmt.Errorf("conversations.open failed: %s", resp.Error)
	}

	return resp.Channel.ID, nil
}

// PostMessage posts text to the given Slack channel ID.
func (c *Client) PostMessage(channelID, text string) error {
	params := url.Values{
		"channel": {channelID},
		"text":    {text},
	}

	body, err := c.doPost("/chat.postMessage", params)
	if err != nil {
		return err
	}

	var resp postMessageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse chat.postMessage response: %w", err)
	}
	if !resp.OK {
		if resp.Error == "not_in_channel" || resp.Error == "channel_not_found" {
			return fmt.Errorf("bot is not a member of the channel, run: /invite @devpilot in the channel")
		}
		return fmt.Errorf("chat.postMessage failed: %s", resp.Error)
	}

	return nil
}

func (c *Client) doGet(path string, params url.Values) ([]byte, error) {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, parseSlackError(body))
	}
	return body, nil
}

func (c *Client) doPost(path string, params url.Values) ([]byte, error) {
	reqURL := c.baseURL + path

	req, err := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.botToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, parseSlackError(body))
	}
	return body, nil
}
