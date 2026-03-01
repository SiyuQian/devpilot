# Devpilot CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI tool (`devpilot`) that handles authentication for external services, starting with Trello.

**Architecture:** Cobra-based CLI with subcommands (login, logout, status). Each service implements a common interface. Credentials stored as JSON at `~/.config/devpilot/credentials.json`.

**Tech Stack:** Go 1.25, Cobra, net/http (stdlib for HTTP calls)

---

### Task 1: Scaffold Go module and Cobra root command

**Files:**
- Create: `cli/go.mod`
- Create: `cli/main.go`
- Create: `cli/cmd/root.go`

**Step 1: Initialize Go module**

Run:
```bash
cd /Users/siyu/Works/github.com/siyuqian/devpilot && mkdir -p cli && cd cli && go mod init github.com/siyuqian/devpilot/cli
```

**Step 2: Add Cobra dependency**

Run:
```bash
cd /Users/siyu/Works/github.com/siyuqian/devpilot/cli && go get github.com/spf13/cobra@latest
```

**Step 3: Create root command**

Create `cli/cmd/root.go`:
```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "devpilot",
	Short: "Developer toolkit for managing service integrations",
	Long:  "devpilot manages authentication and integrations for external services like Trello, GitHub, and more.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 4: Create main.go**

Create `cli/main.go`:
```go
package main

import "github.com/siyuqian/devpilot/cli/cmd"

func main() {
	cmd.Execute()
}
```

**Step 5: Build and verify help output**

Run:
```bash
cd /Users/siyu/Works/github.com/siyuqian/devpilot/cli && go build -o devpilot . && ./devpilot --help
```
Expected: Cobra help text showing "devpilot" with description.

**Step 6: Commit**

```bash
git add cli/
git commit -m "feat: scaffold devpilot CLI with Cobra root command"
```

---

### Task 2: Credentials config module

**Files:**
- Create: `cli/internal/config/credentials.go`

**Step 1: Write credentials test**

Create `cli/internal/config/credentials_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	// Use a temp dir instead of real ~/.config
	tmpDir := t.TempDir()
	origFunc := configDir
	configDir = func() string { return tmpDir }
	defer func() { configDir = origFunc }()

	creds := ServiceCredentials{
		"api_key": "test-key",
		"token":   "test-token",
	}

	if err := Save("trello", creds); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load("trello")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded["api_key"] != "test-key" || loaded["token"] != "test-token" {
		t.Fatalf("unexpected creds: %v", loaded)
	}
}

func TestLoadMissing(t *testing.T) {
	tmpDir := t.TempDir()
	origFunc := configDir
	configDir = func() string { return tmpDir }
	defer func() { configDir = origFunc }()

	_, err := Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing service")
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	origFunc := configDir
	configDir = func() string { return tmpDir }
	defer func() { configDir = origFunc }()

	creds := ServiceCredentials{"api_key": "k", "token": "t"}
	Save("trello", creds)

	if err := Remove("trello"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	_, err := Load("trello")
	if err == nil {
		t.Fatal("expected error after removal")
	}
}

func TestListServices(t *testing.T) {
	tmpDir := t.TempDir()
	origFunc := configDir
	configDir = func() string { return tmpDir }
	defer func() { configDir = origFunc }()

	// Empty at start
	services := ListServices()
	if len(services) != 0 {
		t.Fatalf("expected 0 services, got %d", len(services))
	}

	Save("trello", ServiceCredentials{"api_key": "k", "token": "t"})
	services = ListServices()
	if len(services) != 1 || services[0] != "trello" {
		t.Fatalf("expected [trello], got %v", services)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/siyu/Works/github.com/siyuqian/devpilot/cli && go test ./internal/config/ -v
```
Expected: FAIL — package doesn't exist yet.

**Step 3: Write credentials module**

Create `cli/internal/config/credentials.go`:
```go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ServiceCredentials is a map of credential key-value pairs for a service.
type ServiceCredentials map[string]string

// AllCredentials is the top-level structure of the credentials file.
type AllCredentials map[string]ServiceCredentials

// configDir returns the devpilot config directory path. Variable for testing.
var configDir = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "devpilot")
}

func credentialsPath() string {
	return filepath.Join(configDir(), "credentials.json")
}

func loadAll() (AllCredentials, error) {
	data, err := os.ReadFile(credentialsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return AllCredentials{}, nil
		}
		return nil, err
	}
	var all AllCredentials
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("corrupt credentials file: %w", err)
	}
	return all, nil
}

func saveAll(all AllCredentials) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(credentialsPath(), data, 0600)
}

// Save stores credentials for a service.
func Save(service string, creds ServiceCredentials) error {
	all, err := loadAll()
	if err != nil {
		return err
	}
	all[service] = creds
	return saveAll(all)
}

// Load retrieves credentials for a service.
func Load(service string) (ServiceCredentials, error) {
	all, err := loadAll()
	if err != nil {
		return nil, err
	}
	creds, ok := all[service]
	if !ok {
		return nil, fmt.Errorf("no credentials found for %s", service)
	}
	return creds, nil
}

// Remove deletes credentials for a service.
func Remove(service string) error {
	all, err := loadAll()
	if err != nil {
		return err
	}
	delete(all, service)
	return saveAll(all)
}

// ListServices returns the names of all services with stored credentials.
func ListServices() []string {
	all, err := loadAll()
	if err != nil {
		return nil
	}
	services := make([]string, 0, len(all))
	for name := range all {
		services = append(services, name)
	}
	sort.Strings(services)
	return services
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
cd /Users/siyu/Works/github.com/siyuqian/devpilot/cli && go test ./internal/config/ -v
```
Expected: All 4 tests PASS.

**Step 5: Commit**

```bash
git add cli/internal/config/
git commit -m "feat: add credentials config module with tests"
```

---

### Task 3: Service interface and Trello service

**Files:**
- Create: `cli/internal/services/service.go`
- Create: `cli/internal/services/trello.go`
- Create: `cli/internal/services/registry.go`

**Step 1: Write Trello service test**

Create `cli/internal/services/trello_test.go`:
```go
package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrelloVerify_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1/members/me" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		key := r.URL.Query().Get("key")
		token := r.URL.Query().Get("token")
		if key != "test-key" || token != "test-token" {
			w.WriteHeader(401)
			return
		}
		w.Write([]byte(`{"id":"123","fullName":"Test User"}`))
	}))
	defer server.Close()

	svc := &TrelloService{baseURL: server.URL}
	err := svc.verify("test-key", "test-token")
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestTrelloVerify_InvalidCreds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()

	svc := &TrelloService{baseURL: server.URL}
	err := svc.verify("bad-key", "bad-token")
	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/siyu/Works/github.com/siyuqian/devpilot/cli && go test ./internal/services/ -v
```
Expected: FAIL — package doesn't exist yet.

**Step 3: Write service interface**

Create `cli/internal/services/service.go`:
```go
package services

// Service defines the interface for an external service integration.
type Service interface {
	Name() string
	Login() error
	Logout() error
	IsLoggedIn() bool
}
```

**Step 4: Write Trello service**

Create `cli/internal/services/trello.go`:
```go
package services

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/siyuqian/devpilot/cli/internal/config"
)

const trelloBaseURL = "https://api.trello.com"

// TrelloService handles Trello authentication.
type TrelloService struct {
	baseURL string // overridable for testing
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
```

**Step 5: Write service registry**

Create `cli/internal/services/registry.go`:
```go
package services

import "fmt"

var registry = map[string]Service{}

func init() {
	Register(NewTrelloService())
}

// Register adds a service to the registry.
func Register(svc Service) {
	registry[svc.Name()] = svc
}

// Get retrieves a service by name.
func Get(name string) (Service, error) {
	svc, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown service: %s\nAvailable services: %s", name, AvailableNames())
	}
	return svc, nil
}

// AvailableNames returns a comma-separated list of registered service names.
func AvailableNames() string {
	names := ""
	for name := range registry {
		if names != "" {
			names += ", "
		}
		names += name
	}
	return names
}
```

**Step 6: Run tests to verify they pass**

Run:
```bash
cd /Users/siyu/Works/github.com/siyuqian/devpilot/cli && go test ./internal/services/ -v
```
Expected: Both tests PASS.

**Step 7: Commit**

```bash
git add cli/internal/services/
git commit -m "feat: add service interface, Trello service, and registry"
```

---

### Task 4: Login, Logout, and Status commands

**Files:**
- Create: `cli/cmd/login.go`
- Create: `cli/cmd/logout.go`
- Create: `cli/cmd/status.go`

**Step 1: Write login command**

Create `cli/cmd/login.go`:
```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/siyuqian/devpilot/cli/internal/services"
)

var loginCmd = &cobra.Command{
	Use:   "login <service>",
	Short: "Log in to a service",
	Long:  fmt.Sprintf("Authenticate with an external service.\n\nAvailable services: %s", services.AvailableNames()),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, err := services.Get(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := svc.Login(); err != nil {
			fmt.Fprintln(os.Stderr, "Login failed:", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
```

**Step 2: Write logout command**

Create `cli/cmd/logout.go`:
```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/siyuqian/devpilot/cli/internal/services"
)

var logoutCmd = &cobra.Command{
	Use:   "logout <service>",
	Short: "Log out of a service",
	Long:  fmt.Sprintf("Remove stored credentials for a service.\n\nAvailable services: %s", services.AvailableNames()),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, err := services.Get(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := svc.Logout(); err != nil {
			fmt.Fprintln(os.Stderr, "Logout failed:", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
```

**Step 3: Write status command**

Create `cli/cmd/status.go`:
```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/siyuqian/devpilot/cli/internal/config"
	"github.com/siyuqian/devpilot/cli/internal/services"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show login status for all services",
	Run: func(cmd *cobra.Command, args []string) {
		loggedIn := config.ListServices()
		if len(loggedIn) == 0 {
			fmt.Println("No services configured.")
			fmt.Printf("Run 'devpilot login <service>' to get started. Available: %s\n", services.AvailableNames())
			return
		}
		for _, name := range loggedIn {
			fmt.Printf("%s: logged in\n", name)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
```

**Step 4: Build and test all commands manually**

Run:
```bash
cd /Users/siyu/Works/github.com/siyuqian/devpilot/cli && go build -o devpilot . && ./devpilot --help && ./devpilot status && ./devpilot login --help && ./devpilot logout --help
```
Expected: Help text shows login, logout, status subcommands. Status shows "No services configured."

**Step 5: Commit**

```bash
git add cli/cmd/
git commit -m "feat: add login, logout, and status commands"
```

---

### Task 5: Remove MCP server and update project config

**Files:**
- Delete: `mcps/trello-mcp-server/` (entire directory)
- Modify: `CLAUDE.md`

**Step 1: Delete the MCP server directory**

Run:
```bash
rm -rf /Users/siyu/Works/github.com/siyuqian/devpilot/mcps/
```

**Step 2: Update CLAUDE.md**

Replace the MCP-related sections in `CLAUDE.md` with the new CLI structure. Key changes:
- Remove `mcps/` from repository structure
- Add `cli/` to repository structure
- Replace MCP build commands with Go build commands
- Update architecture section to describe the CLI instead of MCP servers

The updated CLAUDE.md should reflect:

```markdown
## Repository Structure

- `cli/` — Go CLI tool (`devpilot`) for managing service integrations
- `.claude/skills/` — Built-in development skills
  - `skill-creator/` — Guide + scripts for creating new skills
  - `mcp-builder/` — Guide + scripts for building MCP servers
- `docs/plans/` — Design and planning documents

## Build & Development Commands

### Devpilot CLI (`cli/`)

\```bash
cd cli
go build -o devpilot .    # Build binary
go test ./...           # Run all tests
./devpilot --help         # Show help
./devpilot login trello   # Login to Trello
./devpilot status         # Check auth status
\```
```

**Step 3: Commit**

```bash
git add -A
git commit -m "feat: remove Trello MCP server, update CLAUDE.md for devpilot CLI"
```

---

### Task 6: End-to-end verification

**Step 1: Run all tests**

Run:
```bash
cd /Users/siyu/Works/github.com/siyuqian/devpilot/cli && go test ./... -v
```
Expected: All tests pass.

**Step 2: Build and test full flow**

Run:
```bash
cd /Users/siyu/Works/github.com/siyuqian/devpilot/cli && go build -o devpilot . && ./devpilot status
```
Expected: "No services configured."

**Step 3: Test login help**

Run:
```bash
./devpilot login trello
```
Expected: Prints Trello instructions and prompts for API Key.
(Ctrl+C to exit without entering credentials)

**Step 4: Test unknown service**

Run:
```bash
./devpilot login github
```
Expected: Error message listing available services.

**Step 5: Commit final state if any tweaks were needed**

```bash
git add -A
git commit -m "chore: end-to-end verification complete"
```
