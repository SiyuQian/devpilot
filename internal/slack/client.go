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

type Client struct {
	botToken   string
	baseURL    string
	httpClient *http.Client
}

type Option func(*Client)

func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

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

	return "", fmt.Errorf("Channel not found: %s", name)
}

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
			return fmt.Errorf("Bot is not a member of the channel. Run: /invite @devpilot in the channel.")
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, parseSlackError(body))
	}
	return body, nil
}
