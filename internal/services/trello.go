package services

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/siyuqian/developer-kit/internal/config"
)

const trelloBaseURL = "https://api.trello.com"

type TrelloService struct {
	baseURL string
}

func NewTrelloService() *TrelloService {
	return &TrelloService{baseURL: trelloBaseURL}
}

func (t *TrelloService) Name() string {
	return "trello"
}

func (t *TrelloService) Login() error {
	fmt.Println("Trello Login")
	fmt.Println("============")
	fmt.Println()
	fmt.Println("To authenticate, you need an API Key and a Token:")
	fmt.Println()
	fmt.Println("1. Go to https://trello.com/power-ups/admin")
	fmt.Println("2. Click 'New' to create a Power-Up (or use an existing one)")
	fmt.Println("3. Copy the API Key, then click the Token link to generate a token")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	fmt.Print("Token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	if apiKey == "" || token == "" {
		return fmt.Errorf("both API Key and Token are required")
	}

	fmt.Print("Verifying credentials... ")
	if err := t.verify(apiKey, token); err != nil {
		fmt.Println("failed")
		return err
	}
	fmt.Println("ok")

	creds := config.ServiceCredentials{
		"api_key": apiKey,
		"token":   token,
	}
	if err := config.Save(t.Name(), creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println("Credentials saved. You're logged in to Trello.")
	return nil
}

func (t *TrelloService) Logout() error {
	if err := config.Remove(t.Name()); err != nil {
		return err
	}
	fmt.Println("Logged out of Trello.")
	return nil
}

func (t *TrelloService) IsLoggedIn() bool {
	_, err := config.Load(t.Name())
	return err == nil
}

func (t *TrelloService) verify(apiKey, token string) error {
	url := fmt.Sprintf("%s/1/members/me?key=%s&token=%s", t.getBaseURL(), apiKey, token)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to Trello: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid credentials (HTTP %d)", resp.StatusCode)
	}
	return nil
}

func (t *TrelloService) getBaseURL() string {
	if t.baseURL != "" {
		return t.baseURL
	}
	return trelloBaseURL
}
