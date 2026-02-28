package trello

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://api.trello.com"

type Client struct {
	apiKey  string
	token   string
	baseURL string
	http    *http.Client
}

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
	return func(c *Client) { c.baseURL = url }
}

func NewClient(apiKey, token string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:  apiKey,
		token:   token,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) get(path string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("key", c.apiKey)
	params.Set("token", c.token)
	url := fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())

	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) post(path string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("key", c.apiKey)
	params.Set("token", c.token)
	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())

	resp, err := c.http.Post(reqURL, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) put(path string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("key", c.apiKey)
	params.Set("token", c.token)
	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())

	req, err := http.NewRequest(http.MethodPut, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) GetBoards() ([]Board, error) {
	params := url.Values{"filter": {"open"}}
	data, err := c.get("/1/members/me/boards", params)
	if err != nil {
		return nil, err
	}
	var boards []Board
	if err := json.Unmarshal(data, &boards); err != nil {
		return nil, fmt.Errorf("parse boards: %w", err)
	}
	return boards, nil
}

func (c *Client) GetBoardLists(boardID string) ([]List, error) {
	params := url.Values{"filter": {"open"}}
	data, err := c.get(fmt.Sprintf("/1/boards/%s/lists", boardID), params)
	if err != nil {
		return nil, err
	}
	var lists []List
	if err := json.Unmarshal(data, &lists); err != nil {
		return nil, fmt.Errorf("parse lists: %w", err)
	}
	return lists, nil
}

func (c *Client) GetListCards(listID string) ([]Card, error) {
	data, err := c.get(fmt.Sprintf("/1/lists/%s/cards", listID), nil)
	if err != nil {
		return nil, err
	}
	var cards []Card
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, fmt.Errorf("parse cards: %w", err)
	}
	return cards, nil
}

func (c *Client) MoveCard(cardID, listID string) error {
	params := url.Values{"idList": {listID}}
	_, err := c.put(fmt.Sprintf("/1/cards/%s", cardID), params)
	return err
}

func (c *Client) AddComment(cardID, text string) error {
	params := url.Values{"text": {text}}
	_, err := c.post(fmt.Sprintf("/1/cards/%s/actions/comments", cardID), params)
	return err
}

func (c *Client) CreateCard(listID, name, desc string) (*Card, error) {
	params := url.Values{
		"idList": {listID},
		"name":   {name},
		"desc":   {desc},
	}
	data, err := c.post("/1/cards", params)
	if err != nil {
		return nil, err
	}
	var card Card
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("parse card: %w", err)
	}
	return &card, nil
}

func (c *Client) FindBoardByName(name string) (*Board, error) {
	boards, err := c.GetBoards()
	if err != nil {
		return nil, err
	}
	for _, b := range boards {
		if b.Name == name {
			return &b, nil
		}
	}
	return nil, fmt.Errorf("board not found: %s", name)
}

func (c *Client) FindListByName(boardID, name string) (*List, error) {
	lists, err := c.GetBoardLists(boardID)
	if err != nil {
		return nil, err
	}
	for _, l := range lists {
		if l.Name == name {
			return &l, nil
		}
	}
	return nil, fmt.Errorf("list not found: %s", name)
}
