# Domain-Based Refactoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reorganize `internal/` from technical-layer packages (`cli/`, `config/`, `services/`, `runner/`) to domain-based packages (`auth/`, `trello/`, `taskrunner/`).

**Architecture:** Each domain package owns its code, tests, and CLI commands. `cmd/devkit/main.go` wires domains together via `RegisterCommands()`. No circular dependencies: `auth` → none, `trello` → `auth`, `taskrunner` → `trello` + `auth`.

**Tech Stack:** Go 1.25, Cobra CLI, module `github.com/siyuqian/developer-kit`

---

### Task 1: Create `internal/auth/` — credentials

**Files:**
- Create: `internal/auth/credentials.go`
- Create: `internal/auth/credentials_test.go`

**Step 1: Create `internal/auth/credentials.go`**

Copy from `internal/config/credentials.go`, change package name only:

```go
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type ServiceCredentials map[string]string

type AllCredentials map[string]ServiceCredentials

var configDir = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "devkit")
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

func Save(service string, creds ServiceCredentials) error {
	all, err := loadAll()
	if err != nil {
		return err
	}
	all[service] = creds
	return saveAll(all)
}

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

func Remove(service string) error {
	all, err := loadAll()
	if err != nil {
		return err
	}
	delete(all, service)
	return saveAll(all)
}

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

**Step 2: Create `internal/auth/credentials_test.go`**

Copy from `internal/config/credentials_test.go`, change package name only:

```go
package auth

import (
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
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

**Step 3: Run tests**

Run: `go test ./internal/auth/...`
Expected: all 4 tests PASS

**Step 4: Commit**

```bash
git add internal/auth/credentials.go internal/auth/credentials_test.go
git commit -m "refactor: create auth package with credentials from config"
```

---

### Task 2: Create `internal/auth/` — service interface, registry, trello service

**Files:**
- Create: `internal/auth/service.go`
- Create: `internal/auth/trello_service.go`
- Create: `internal/auth/trello_service_test.go`

**Step 1: Create `internal/auth/service.go`**

Merges `services/service.go` and `services/registry.go` into one file:

```go
package auth

import "fmt"

type Service interface {
	Name() string
	Login() error
	Logout() error
	IsLoggedIn() bool
}

var registry = map[string]Service{}

func init() {
	Register(NewTrelloService())
}

func Register(svc Service) {
	registry[svc.Name()] = svc
}

func Get(name string) (Service, error) {
	svc, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown service: %s\nAvailable services: %s", name, AvailableNames())
	}
	return svc, nil
}

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

**Step 2: Create `internal/auth/trello_service.go`**

From `services/trello.go`. Key change: references to `config.Save`/`config.Load`/`config.Remove`/`config.ServiceCredentials` become local calls (`Save`/`Load`/`Remove`/`ServiceCredentials`) since they're in the same package now.

```go
package auth

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
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

	creds := ServiceCredentials{
		"api_key": apiKey,
		"token":   token,
	}
	if err := Save(t.Name(), creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println("Credentials saved. You're logged in to Trello.")
	return nil
}

func (t *TrelloService) Logout() error {
	if err := Remove(t.Name()); err != nil {
		return err
	}
	fmt.Println("Logged out of Trello.")
	return nil
}

func (t *TrelloService) IsLoggedIn() bool {
	_, err := Load(t.Name())
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

**Step 3: Create `internal/auth/trello_service_test.go`**

From `services/trello_test.go`, change package to `auth`:

```go
package auth

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

**Step 4: Run tests**

Run: `go test ./internal/auth/...`
Expected: all 6 tests PASS (4 credentials + 2 trello verify)

**Step 5: Commit**

```bash
git add internal/auth/service.go internal/auth/trello_service.go internal/auth/trello_service_test.go
git commit -m "refactor: add service interface, registry, and trello service to auth package"
```

---

### Task 3: Create `internal/auth/commands.go` — login, logout, status CLI commands

**Files:**
- Create: `internal/auth/commands.go`

**Step 1: Create `internal/auth/commands.go`**

Merges login, logout, and status commands. Uses `RegisterCommands(parent)` pattern instead of `init()`.

```go
package auth

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func RegisterCommands(parent *cobra.Command) {
	parent.AddCommand(loginCmd)
	parent.AddCommand(logoutCmd)
	parent.AddCommand(statusCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login <service>",
	Short: "Log in to a service",
	Long:  fmt.Sprintf("Authenticate with an external service.\n\nAvailable services: %s", AvailableNames()),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, err := Get(args[0])
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

var logoutCmd = &cobra.Command{
	Use:   "logout <service>",
	Short: "Log out of a service",
	Long:  fmt.Sprintf("Remove stored credentials for a service.\n\nAvailable services: %s", AvailableNames()),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, err := Get(args[0])
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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show login status for all services",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		loggedIn := ListServices()
		if len(loggedIn) == 0 {
			fmt.Println("No services configured.")
			fmt.Printf("Run 'devkit login <service>' to get started. Available: %s\n", AvailableNames())
			return
		}
		for _, name := range loggedIn {
			fmt.Printf("%s: logged in\n", name)
		}
	},
}
```

**Step 2: Verify compilation**

Run: `go build ./internal/auth/...`
Expected: compiles without errors

**Step 3: Commit**

```bash
git add internal/auth/commands.go
git commit -m "refactor: add login, logout, status CLI commands to auth package"
```

---

### Task 4: Add push command to `internal/trello/`

**Files:**
- Create: `internal/trello/commands.go`
- Create: `internal/trello/commands_test.go`

**Step 1: Create `internal/trello/commands.go`**

Push command moves here. Import changes: `config` → `auth`, `trello` references become local.

```go
package trello

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/auth"
)

func RegisterCommands(parent *cobra.Command) {
	pushCmd.Flags().String("board", "", "Trello board name (required)")
	pushCmd.Flags().String("list", "Ready", "Target list name")
	parent.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push <plan-file>",
	Short: "Create a Trello card from a plan file",
	Long:  "Read a plan markdown file and create a Trello card with the title from the first # heading and the full file contents as the description.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		boardName, _ := cmd.Flags().GetString("board")
		listName, _ := cmd.Flags().GetString("list")

		if boardName == "" {
			fmt.Fprintln(os.Stderr, "Error: --board is required")
			os.Exit(1)
		}

		// Read the plan file
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		// Extract title from first # heading
		title := ExtractTitle(string(content))
		if title == "" {
			fmt.Fprintln(os.Stderr, "Error: no # heading found in file")
			os.Exit(1)
		}

		// Load Trello credentials
		creds, err := auth.Load("trello")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Not logged in to Trello. Run: devkit login trello")
			os.Exit(1)
		}

		client := NewClient(creds["api_key"], creds["token"])

		// Resolve board
		board, err := client.FindBoardByName(boardName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Resolve list
		list, err := client.FindListByName(board.ID, listName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Create card
		card, err := client.CreateCard(list.ID, title, string(content))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating card: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created card: %s\n", title)
		if card.ShortURL != "" {
			fmt.Println(card.ShortURL)
		}
	},
}

func ExtractTitle(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if title, ok := strings.CutPrefix(line, "# "); ok {
			return title
		}
	}
	return ""
}
```

Note: `extractTitle` is renamed to `ExtractTitle` (exported) since tests need access to it from `_test.go` in the same package. Actually, since the test file uses the same package, it can stay unexported. But the original was unexported, so let's keep it unexported:

Actually, the test is in the same package (`trello`), so it can access unexported functions. Let me correct: keep it as `extractTitle`.

Replace `ExtractTitle` with `extractTitle` in the above code, and update the call site to `extractTitle`.

**Step 2: Create `internal/trello/commands_test.go`**

```go
package trello

import "testing"

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "standard plan heading",
			content: "# Task Runner Implementation Plan\n\n> For Claude...\n",
			want:    "Task Runner Implementation Plan",
		},
		{
			name:    "heading after blank lines",
			content: "\n\n# My Plan\n\nBody text",
			want:    "My Plan",
		},
		{
			name:    "no heading",
			content: "Just some text\nNo heading here",
			want:    "",
		},
		{
			name:    "ignores ## subheadings",
			content: "## Not This\n# This One\n",
			want:    "This One",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.content)
			if got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

**Step 3: Run tests**

Run: `go test ./internal/trello/...`
Expected: all existing client tests + 4 extractTitle tests PASS

**Step 4: Commit**

```bash
git add internal/trello/commands.go internal/trello/commands_test.go
git commit -m "refactor: add push command to trello package"
```

---

### Task 5: Create `internal/taskrunner/` — rename from runner + add run command

**Files:**
- Create: `internal/taskrunner/executor.go` (from `runner/executor.go`, package rename)
- Create: `internal/taskrunner/executor_test.go` (from `runner/executor_test.go`, package rename)
- Create: `internal/taskrunner/git.go` (from `runner/git.go`, package rename)
- Create: `internal/taskrunner/git_test.go` (from `runner/git_test.go`, package rename)
- Create: `internal/taskrunner/reviewer.go` (from `runner/reviewer.go`, package rename)
- Create: `internal/taskrunner/reviewer_test.go` (from `runner/reviewer_test.go`, package rename)
- Create: `internal/taskrunner/runner.go` (from `runner/runner.go`, package rename + update import)
- Create: `internal/taskrunner/commands.go` (from `cli/run.go`)

**Step 1: Copy all runner files to taskrunner with package rename**

For each file in `internal/runner/`, copy to `internal/taskrunner/` and change `package runner` → `package taskrunner`.

For `internal/taskrunner/runner.go`, also update the import:
- `"github.com/siyuqian/developer-kit/internal/trello"` stays the same (trello package doesn't move)

**Step 2: Create `internal/taskrunner/commands.go`**

From `cli/run.go`. Import changes: `config` → `auth`, `runner` references become local.

```go
package taskrunner

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/auth"
	"github.com/siyuqian/developer-kit/internal/trello"
)

func RegisterCommands(parent *cobra.Command) {
	runCmd.Flags().String("board", "", "Trello board name (required)")
	runCmd.Flags().Int("interval", 300, "Poll interval in seconds")
	runCmd.Flags().Int("timeout", 30, "Per-task timeout in minutes")
	runCmd.Flags().Int("review-timeout", 10, "Code review timeout in minutes (0 to disable)")
	runCmd.Flags().Bool("once", false, "Process one card and exit")
	runCmd.Flags().Bool("dry-run", false, "Print actions without executing")
	parent.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Autonomously process tasks from a Trello board",
	Long:  "Poll a Trello board for Ready cards, execute their plans via Claude Code, and create PRs.",
	Run: func(cmd *cobra.Command, args []string) {
		boardName, _ := cmd.Flags().GetString("board")
		interval, _ := cmd.Flags().GetInt("interval")
		timeout, _ := cmd.Flags().GetInt("timeout")
		reviewTimeout, _ := cmd.Flags().GetInt("review-timeout")
		once, _ := cmd.Flags().GetBool("once")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if boardName == "" {
			fmt.Fprintln(os.Stderr, "Error: --board is required")
			os.Exit(1)
		}

		// Load Trello credentials
		creds, err := auth.Load("trello")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Not logged in to Trello. Run: devkit login trello")
			os.Exit(1)
		}

		trelloClient := trello.NewClient(creds["api_key"], creds["token"])

		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to get working directory:", err)
			os.Exit(1)
		}

		cfg := Config{
			BoardName:     boardName,
			Interval:      time.Duration(interval) * time.Second,
			Timeout:       time.Duration(timeout) * time.Minute,
			ReviewTimeout: time.Duration(reviewTimeout) * time.Minute,
			Once:          once,
			DryRun:        dryRun,
			WorkDir:       dir,
		}

		r := New(cfg, trelloClient)

		// Handle Ctrl+C
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		go func() {
			<-sigCh
			fmt.Println("\nReceived interrupt, finishing current task...")
			cancel()
		}()

		if err := r.Run(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "Runner error:", err)
			os.Exit(1)
		}
	},
}
```

**Step 3: Run tests**

Run: `go test ./internal/taskrunner/...`
Expected: all executor, git, reviewer tests PASS

**Step 4: Commit**

```bash
git add internal/taskrunner/
git commit -m "refactor: create taskrunner package from runner with run command"
```

---

### Task 6: Update `cmd/devkit/main.go` and delete old packages

**Files:**
- Modify: `cmd/devkit/main.go`
- Delete: `internal/cli/` (entire directory)
- Delete: `internal/config/` (entire directory)
- Delete: `internal/services/` (entire directory)
- Delete: `internal/runner/` (entire directory)

**Step 1: Rewrite `cmd/devkit/main.go`**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/auth"
	"github.com/siyuqian/developer-kit/internal/taskrunner"
	"github.com/siyuqian/developer-kit/internal/trello"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "devkit",
		Short: "Developer toolkit for managing service integrations",
		Long:  "devkit manages authentication and integrations for external services like Trello, GitHub, and more.",
	}

	auth.RegisterCommands(rootCmd)
	trello.RegisterCommands(rootCmd)
	taskrunner.RegisterCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 2: Delete old directories**

```bash
rm -rf internal/cli internal/config internal/services internal/runner
```

**Step 3: Run all tests**

Run: `go test ./...`
Expected: ALL tests PASS

**Step 4: Build and verify CLI**

Run: `go build -o /dev/null ./cmd/devkit/`
Expected: compiles successfully

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor: switch to domain-based packages, remove old layout"
```

---

### Task 7: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Update the Repository Structure and Architecture sections**

Update the `## Repository Structure` section to reflect:

```
- `cmd/devkit/` — CLI entry point (wires domain packages)
- `internal/auth/` — Authentication, credentials, service registry, CLI commands (login/logout/status)
- `internal/trello/` — Trello REST API client, types, CLI commands (push)
- `internal/taskrunner/` — Task executor, git ops, code reviewer, runner loop, CLI commands (run)
```

Update the `## Architecture` section:

```
### Devkit CLI

Go CLI tool using Cobra, organized by domain:
- `cmd/devkit/` — Entry point, creates root command, registers domain commands
- `internal/auth/` — Credentials storage (~/.config/devkit/credentials.json), Service interface + registry, Trello auth (login/logout/verify), CLI commands: login, logout, status
- `internal/trello/` — Trello REST API client (boards, lists, cards), CLI commands: push
- `internal/taskrunner/` — Executor (claude -p wrapper), GitOps (branch/PR management), Reviewer (automated code review), Runner (poll loop), CLI commands: run
- Adding a new service: implement the Service interface in `internal/auth/`, register in `service.go` init()
```

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md for domain-based package structure"
```

---

### Task 8: Final verification

**Step 1: Run full test suite**

Run: `go test ./...`
Expected: ALL tests PASS, no compilation errors

**Step 2: Build binary**

Run: `make build`
Expected: binary builds successfully

**Step 3: Verify no leftover references to old packages**

Run: `grep -r "internal/cli\|internal/config\|internal/services\|internal/runner" --include="*.go" .`
Expected: no matches (design doc matches are OK since they're in .md files)
