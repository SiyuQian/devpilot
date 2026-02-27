# Task Runner Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `devkit run` command that autonomously picks up Trello cards, executes their plans via `claude -p`, and creates PRs.

**Architecture:** Go CLI loop (poll Trello → spawn Claude → create PR → move card). Trello HTTP client extracted into its own package. Runner logic in `internal/runner/`. Task execution delegated to a Claude skill.

**Tech Stack:** Go + Cobra, `net/http` for Trello API, `os/exec` for `claude -p` and `gh` CLI

---

### Task 1: Trello Client — Types

**Files:**
- Create: `internal/trello/types.go`

**Step 1: Create the types file**

```go
package trello

type Board struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type List struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Card struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	IDList string `json:"idList"`
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/trello/`
Expected: no errors

**Step 3: Commit**

```bash
git add internal/trello/types.go
git commit -m "feat(trello): add Board, List, Card types"
```

---

### Task 2: Trello Client — Core HTTP Client

**Files:**
- Create: `internal/trello/client.go`
- Create: `internal/trello/client_test.go`

**Step 1: Write the failing test for NewClient and GetBoards**

```go
package trello

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetBoards(t *testing.T) {
	boards := []Board{{ID: "board1", Name: "Sprint Board"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1/members/me/boards" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("filter") != "open" {
			t.Error("expected filter=open")
		}
		if r.URL.Query().Get("key") == "" || r.URL.Query().Get("token") == "" {
			t.Error("missing auth params")
		}
		json.NewEncoder(w).Encode(boards)
	}))
	defer server.Close()

	client := NewClient("testkey", "testtoken", WithBaseURL(server.URL))
	result, err := client.GetBoards()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Name != "Sprint Board" {
		t.Errorf("unexpected boards: %+v", result)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/trello/ -run TestGetBoards -v`
Expected: FAIL — `NewClient` not defined

**Step 3: Implement the client**

```go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/trello/ -run TestGetBoards -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/trello/client.go internal/trello/client_test.go
git commit -m "feat(trello): add HTTP client with GetBoards"
```

---

### Task 3: Trello Client — List & Card Operations

**Files:**
- Modify: `internal/trello/client.go`
- Modify: `internal/trello/client_test.go`

**Step 1: Write failing tests for GetBoardLists, GetListCards, MoveCard, AddComment**

```go
func TestGetBoardLists(t *testing.T) {
	lists := []List{{ID: "list1", Name: "Ready"}, {ID: "list2", Name: "Done"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1/boards/board1/lists" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(lists)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))
	result, err := client.GetBoardLists("board1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 || result[0].Name != "Ready" {
		t.Errorf("unexpected lists: %+v", result)
	}
}

func TestGetListCards(t *testing.T) {
	cards := []Card{{ID: "card1", Name: "Fix bug", Desc: "the plan"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1/lists/list1/cards" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(cards)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))
	result, err := client.GetListCards("list1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Desc != "the plan" {
		t.Errorf("unexpected cards: %+v", result)
	}
}

func TestMoveCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/1/cards/card1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id":"card1"}`)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))
	err := client.MoveCard("card1", "list2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/1/cards/card1/actions/comments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))
	err := client.AddComment("card1", "task done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/trello/ -v`
Expected: FAIL — methods not defined

**Step 3: Implement the methods**

Add to `internal/trello/client.go`:

```go
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
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/trello/ -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/trello/client.go internal/trello/client_test.go
git commit -m "feat(trello): add GetBoardLists, GetListCards, MoveCard, AddComment"
```

---

### Task 4: Trello Client — Board Resolution Helper

**Files:**
- Modify: `internal/trello/client.go`
- Modify: `internal/trello/client_test.go`

**Step 1: Write failing test for FindBoardByName and FindListByName**

```go
func TestFindBoardByName(t *testing.T) {
	boards := []Board{{ID: "b1", Name: "Sprint Board"}, {ID: "b2", Name: "Backlog"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(boards)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))

	board, err := client.FindBoardByName("Sprint Board")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if board.ID != "b1" {
		t.Errorf("expected b1, got %s", board.ID)
	}

	_, err = client.FindBoardByName("Nonexistent")
	if err == nil {
		t.Error("expected error for missing board")
	}
}

func TestFindListByName(t *testing.T) {
	lists := []List{{ID: "l1", Name: "Ready"}, {ID: "l2", Name: "Done"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(lists)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))

	list, err := client.FindListByName("board1", "Ready")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.ID != "l1" {
		t.Errorf("expected l1, got %s", list.ID)
	}

	_, err = client.FindListByName("board1", "Nonexistent")
	if err == nil {
		t.Error("expected error for missing list")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/trello/ -run TestFind -v`
Expected: FAIL

**Step 3: Implement**

Add to `internal/trello/client.go`:

```go
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
```

**Step 4: Run tests**

Run: `go test ./internal/trello/ -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/trello/client.go internal/trello/client_test.go
git commit -m "feat(trello): add FindBoardByName and FindListByName helpers"
```

---

### Task 5: Runner — Executor (claude -p wrapper)

**Files:**
- Create: `internal/runner/executor.go`
- Create: `internal/runner/executor_test.go`

**Step 1: Write failing test**

```go
package runner

import (
	"context"
	"testing"
	"time"
)

func TestExecute_Success(t *testing.T) {
	exec := NewExecutor(WithCommand("echo", "hello"))
	result, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", result.ExitCode)
	}
	if result.Stdout == "" {
		t.Error("expected stdout output")
	}
}

func TestExecute_Failure(t *testing.T) {
	exec := NewExecutor(WithCommand("false"))
	result, err := exec.Run(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code")
	}
}

func TestExecute_Timeout(t *testing.T) {
	exec := NewExecutor(WithCommand("sleep", "10"))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	result, err := exec.Run(ctx, "test prompt")
	if err == nil && !result.TimedOut {
		t.Error("expected timeout")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/runner/ -v`
Expected: FAIL

**Step 3: Implement**

```go
package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"
)

type ExecuteResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	TimedOut bool
}

type Executor struct {
	command string
	args    []string
}

type ExecutorOption func(*Executor)

func WithCommand(command string, args ...string) ExecutorOption {
	return func(e *Executor) {
		e.command = command
		e.args = args
	}
}

func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		command: "claude",
		args:    []string{"-p", "--allowedTools", "*"},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *Executor) Run(ctx context.Context, prompt string) (*ExecuteResult, error) {
	args := make([]string, len(e.args))
	copy(args, e.args)

	// Only append prompt if using claude (not test commands)
	if e.command == "claude" {
		args = append(args, prompt)
	}

	cmd := exec.CommandContext(ctx, e.command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &ExecuteResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		return result, fmt.Errorf("execution timed out")
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.Sys().(syscall.WaitStatus).ExitStatus()
			return result, nil
		}
		return result, fmt.Errorf("exec failed: %w", err)
	}

	result.ExitCode = 0
	return result, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/runner/ -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/runner/executor.go internal/runner/executor_test.go
git commit -m "feat(runner): add Executor for claude -p invocation"
```

---

### Task 6: Runner — Git Operations

**Files:**
- Create: `internal/runner/git.go`
- Create: `internal/runner/git_test.go`

**Step 1: Write failing test**

```go
package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %s %v", args, out, err)
		}
	}
	return dir
}

func TestCreateBranch(t *testing.T) {
	dir := setupGitRepo(t)
	git := NewGitOps(dir)

	err := git.CreateBranch("task/abc123-fix-bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify we're on the new branch
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	out, _ := cmd.Output()
	branch := string(out)
	if branch != "task/abc123-fix-bug\n" {
		t.Errorf("expected task/abc123-fix-bug, got %q", branch)
	}
}

func TestCheckoutMain(t *testing.T) {
	dir := setupGitRepo(t)
	git := NewGitOps(dir)

	// Create and switch to a branch
	git.CreateBranch("task/test")

	err := git.CheckoutMain()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	out, _ := cmd.Output()
	// Could be "main" or "master" depending on git version
	branch := string(out)
	if branch != "main\n" && branch != "master\n" {
		t.Errorf("expected main or master, got %q", branch)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Fix auth bug", "fix-auth-bug"},
		{"Add Login Endpoint!!", "add-login-endpoint"},
		{"hello   world", "hello-world"},
	}
	for _, tt := range tests {
		got := Slugify(tt.input)
		if got != tt.expected {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/runner/ -run "TestCreateBranch|TestCheckoutMain|TestSlugify" -v`
Expected: FAIL

**Step 3: Implement**

```go
package runner

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type GitOps struct {
	dir string
}

func NewGitOps(dir string) *GitOps {
	return &GitOps{dir: dir}
}

func (g *GitOps) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s %w", strings.Join(args, " "), string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *GitOps) CreateBranch(name string) error {
	_, err := g.run("checkout", "-b", name)
	return err
}

func (g *GitOps) CheckoutMain() error {
	// Try main first, fall back to master
	if _, err := g.run("checkout", "main"); err != nil {
		_, err = g.run("checkout", "master")
		return err
	}
	return nil
}

func (g *GitOps) Pull() error {
	_, err := g.run("pull", "--ff-only")
	return err
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
```

**Step 4: Run tests**

Run: `go test ./internal/runner/ -run "TestCreateBranch|TestCheckoutMain|TestSlugify" -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/runner/git.go internal/runner/git_test.go
git commit -m "feat(runner): add GitOps for branch management"
```

---

### Task 7: Runner — PR Operations

**Files:**
- Modify: `internal/runner/git.go`
- Modify: `internal/runner/git_test.go`

**Step 1: Write failing test for CreatePR and MergePR**

Note: These call `gh` CLI so we test with a mock command approach. For unit tests, we verify the command construction. Integration testing with real `gh` is out of scope for v1.

```go
func TestBranchName(t *testing.T) {
	git := NewGitOps("/tmp")
	name := git.BranchName("abc123", "Fix auth bug")
	if name != "task/abc123-fix-auth-bug" {
		t.Errorf("unexpected branch name: %s", name)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/runner/ -run TestBranchName -v`
Expected: FAIL

**Step 3: Implement BranchName, Push, CreatePR, MergePR**

Add to `internal/runner/git.go`:

```go
func (g *GitOps) BranchName(cardID, cardName string) string {
	slug := Slugify(cardName)
	// Truncate slug to keep branch name reasonable
	if len(slug) > 40 {
		slug = slug[:40]
		slug = strings.TrimRight(slug, "-")
	}
	return fmt.Sprintf("task/%s-%s", cardID, slug)
}

func (g *GitOps) Push(branch string) error {
	_, err := g.run("push", "-u", "origin", branch)
	return err
}

func (g *GitOps) CreatePR(title, body string) (string, error) {
	cmd := exec.Command("gh", "pr", "create", "--title", title, "--body", body)
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr create: %s %w", string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *GitOps) MergePR() error {
	cmd := exec.Command("gh", "pr", "merge", "--squash", "--auto")
	cmd.Dir = g.dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh pr merge: %s %w", string(out), err)
	}
	return nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/runner/ -v`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/runner/git.go internal/runner/git_test.go
git commit -m "feat(runner): add PR creation and merge via gh CLI"
```

---

### Task 8: Runner — Main Loop

**Files:**
- Create: `internal/runner/runner.go`

**Step 1: Implement the Runner struct and Run method**

```go
package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/siyuqian/developer-kit/internal/trello"
)

type Config struct {
	BoardName   string
	Interval    time.Duration
	Timeout     time.Duration
	Once        bool
	DryRun      bool
	WorkDir     string
}

type Runner struct {
	config   Config
	trello   *trello.Client
	executor *Executor
	git      *GitOps
	logger   *log.Logger

	// Resolved IDs
	boardID      string
	readyListID  string
	inProgListID string
	doneListID   string
	failedListID string
}

func New(cfg Config, trelloClient *trello.Client) *Runner {
	return &Runner{
		config:   cfg,
		trello:   trelloClient,
		executor: NewExecutor(),
		git:      NewGitOps(cfg.WorkDir),
		logger:   log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (r *Runner) init() error {
	r.logger.Printf("Resolving board: %s", r.config.BoardName)
	board, err := r.trello.FindBoardByName(r.config.BoardName)
	if err != nil {
		return fmt.Errorf("find board: %w", err)
	}
	r.boardID = board.ID
	r.logger.Printf("Board found: %s (%s)", board.Name, board.ID)

	listNames := map[string]*string{
		"Ready":       &r.readyListID,
		"In Progress": &r.inProgListID,
		"Done":        &r.doneListID,
		"Failed":      &r.failedListID,
	}
	for name, idPtr := range listNames {
		list, err := r.trello.FindListByName(r.boardID, name)
		if err != nil {
			return fmt.Errorf("find list %q: %w", name, err)
		}
		*idPtr = list.ID
		r.logger.Printf("List %q → %s", name, list.ID)
	}
	return nil
}

func (r *Runner) Run(ctx context.Context) error {
	if err := r.init(); err != nil {
		return err
	}

	r.logger.Println("Runner started. Polling for tasks...")

	for {
		select {
		case <-ctx.Done():
			r.logger.Println("Shutting down.")
			return nil
		default:
		}

		cards, err := r.trello.GetListCards(r.readyListID)
		if err != nil {
			r.logger.Printf("Error polling: %v. Retrying in %s...", err, r.config.Interval)
			time.Sleep(r.config.Interval)
			continue
		}

		if len(cards) == 0 {
			r.logger.Printf("No tasks. Sleeping %s...", r.config.Interval)
			time.Sleep(r.config.Interval)
			continue
		}

		card := cards[0]
		r.processCard(ctx, card)

		if r.config.Once {
			r.logger.Println("--once flag set. Exiting.")
			return nil
		}
	}
}

func (r *Runner) processCard(ctx context.Context, card trello.Card) {
	start := time.Now()
	r.logger.Printf("Processing card: %q (%s)", card.Name, card.ID)

	if card.Desc == "" {
		r.logger.Printf("Card has empty description, marking as failed")
		r.trello.MoveCard(card.ID, r.failedListID)
		r.trello.AddComment(card.ID, "❌ Task failed\nError: Empty plan — card description is empty")
		return
	}

	if r.config.DryRun {
		r.logger.Printf("[DRY RUN] Would process card: %q", card.Name)
		return
	}

	// Move to In Progress
	if err := r.trello.MoveCard(card.ID, r.inProgListID); err != nil {
		r.logger.Printf("Failed to move card to In Progress: %v", err)
	}

	// Git: checkout main, pull, create branch
	branch := r.git.BranchName(card.ID, card.Name)
	if err := r.git.CheckoutMain(); err != nil {
		r.failCard(card, start, fmt.Sprintf("git checkout main: %v", err))
		return
	}
	r.git.Pull() // best-effort
	if err := r.git.CreateBranch(branch); err != nil {
		r.failCard(card, start, fmt.Sprintf("git create branch: %v", err))
		return
	}

	// Build prompt
	prompt := r.buildPrompt(card)

	// Execute
	taskCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	result, err := r.executor.Run(taskCtx, prompt)

	// Save log
	r.saveLog(card.ID, result)

	if err != nil || result.ExitCode != 0 {
		errMsg := "non-zero exit code"
		if result.TimedOut {
			errMsg = "execution timed out"
		} else if result.Stderr != "" {
			errMsg = truncate(result.Stderr, 500)
		}
		r.failCard(card, start, errMsg)
		r.git.CheckoutMain()
		return
	}

	// Push and create PR
	if err := r.git.Push(branch); err != nil {
		r.failCard(card, start, fmt.Sprintf("git push: %v", err))
		r.git.CheckoutMain()
		return
	}

	cardURL := fmt.Sprintf("https://trello.com/c/%s", card.ID)
	prBody := fmt.Sprintf("## Task\n%s\n\n🤖 Executed by devkit runner", cardURL)
	prURL, err := r.git.CreatePR(card.Name, prBody)
	if err != nil {
		r.failCard(card, start, fmt.Sprintf("create PR: %v", err))
		r.git.CheckoutMain()
		return
	}

	if err := r.git.MergePR(); err != nil {
		r.logger.Printf("Auto-merge failed (may need approval): %v", err)
	}

	// Move to Done
	duration := time.Since(start).Round(time.Second)
	r.trello.MoveCard(card.ID, r.doneListID)
	r.trello.AddComment(card.ID, fmt.Sprintf("✅ Task completed by devkit runner\nDuration: %s\nPR: %s", duration, prURL))
	r.logger.Printf("Card %q completed in %s. PR: %s", card.Name, duration, prURL)

	r.git.CheckoutMain()
	r.git.Pull()
}

func (r *Runner) buildPrompt(card trello.Card) string {
	return fmt.Sprintf(`Execute the following task plan. Use /superpowers:test-driven-development and /superpowers:verification-before-completion skills during execution.

Task: %s

Plan:
%s

When done:
- Commit all changes with a descriptive message
- Push to the appropriate branch`, card.Name, card.Desc)
}

func (r *Runner) failCard(card trello.Card, start time.Time, errMsg string) {
	duration := time.Since(start).Round(time.Second)
	logPath := filepath.Join("~/.config/devkit/logs", card.ID+".log")
	comment := fmt.Sprintf("❌ Task failed\nDuration: %s\nError: %s\nSee full log: %s", duration, errMsg, logPath)
	r.trello.MoveCard(card.ID, r.failedListID)
	r.trello.AddComment(card.ID, comment)
	r.logger.Printf("Card %q failed: %s", card.Name, errMsg)
}

func (r *Runner) saveLog(cardID string, result *ExecuteResult) {
	if result == nil {
		return
	}
	logDir := filepath.Join(os.Getenv("HOME"), ".config", "devkit", "logs")
	os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, cardID+".log")
	content := fmt.Sprintf("=== STDOUT ===\n%s\n\n=== STDERR ===\n%s\n", result.Stdout, result.Stderr)
	os.WriteFile(logPath, []byte(content), 0644)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/runner/`
Expected: no errors

**Step 3: Commit**

```bash
git add internal/runner/runner.go
git commit -m "feat(runner): add main Runner loop with card processing"
```

---

### Task 9: CLI Command — `devkit run`

**Files:**
- Create: `internal/cli/run.go`

**Step 1: Implement the Cobra command**

```go
package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/config"
	"github.com/siyuqian/developer-kit/internal/runner"
	"github.com/siyuqian/developer-kit/internal/trello"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Autonomously process tasks from a Trello board",
	Long:  "Poll a Trello board for Ready cards, execute their plans via Claude Code, and create PRs.",
	Run: func(cmd *cobra.Command, args []string) {
		boardName, _ := cmd.Flags().GetString("board")
		interval, _ := cmd.Flags().GetInt("interval")
		timeout, _ := cmd.Flags().GetInt("timeout")
		once, _ := cmd.Flags().GetBool("once")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if boardName == "" {
			fmt.Fprintln(os.Stderr, "Error: --board is required")
			os.Exit(1)
		}

		// Load Trello credentials
		creds, err := config.Load("trello")
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

		cfg := runner.Config{
			BoardName: boardName,
			Interval:  time.Duration(interval) * time.Second,
			Timeout:   time.Duration(timeout) * time.Minute,
			Once:      once,
			DryRun:    dryRun,
			WorkDir:   dir,
		}

		r := runner.New(cfg, trelloClient)

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

func init() {
	runCmd.Flags().String("board", "", "Trello board name (required)")
	runCmd.Flags().Int("interval", 300, "Poll interval in seconds")
	runCmd.Flags().Int("timeout", 30, "Per-task timeout in minutes")
	runCmd.Flags().Bool("once", false, "Process one card and exit")
	runCmd.Flags().Bool("dry-run", false, "Print actions without executing")
	rootCmd.AddCommand(runCmd)
}
```

**Step 2: Verify it compiles**

Run: `go build ./cmd/devkit/`
Expected: no errors

**Step 3: Verify help output**

Run: `go run ./cmd/devkit/ run --help`
Expected: shows usage with --board, --interval, --timeout, --once, --dry-run flags

**Step 4: Commit**

```bash
git add internal/cli/run.go
git commit -m "feat(cli): add devkit run command"
```

---

### Task 10: Task Executor Skill

**Files:**
- Create: `.claude/skills/task-executor/SKILL.md`

**Step 1: Create the skill file**

```markdown
---
name: developerkit:task-executor
description: Executes a task plan autonomously. Used by the devkit runner to process Trello cards. Follows execution plans step-by-step using TDD and verification skills.
---

# Task Executor

Execute implementation plans autonomously. This skill is invoked by `devkit run` via `claude -p`.

## Process

1. **Parse the plan** — Read the provided execution plan and identify each task/step.

2. **Execute step-by-step** — For each step in the plan:
   - If the step involves writing code, use the `superpowers:test-driven-development` skill
   - If the step involves debugging, use the `superpowers:systematic-debugging` skill
   - Follow exact file paths and commands from the plan
   - Commit after each logical unit of work

3. **Verify before completion** — Use the `superpowers:verification-before-completion` skill:
   - Run all tests
   - Run any verification commands specified in the plan
   - Confirm all changes compile and pass

4. **Commit and push** — Create descriptive commit messages for each change. Push to the current branch.

## Rules

- **Follow the plan exactly** — Do not add features, refactor, or deviate from what the plan specifies
- **Fail fast** — If a step is blocked and cannot be resolved, exit with a non-zero code and a clear error message to stderr
- **No interactive prompts** — This runs unattended. Never ask for user input.
- **Commit frequently** — Small, focused commits are better than one large commit
- **Push at the end** — Push all commits to the current branch when all steps are complete
```

**Step 2: Commit**

```bash
git add .claude/skills/task-executor/SKILL.md
git commit -m "feat(skills): add task-executor skill for autonomous plan execution"
```

---

### Task 11: Integration Test — Dry Run

**Files:**
- No new files — manual verification

**Step 1: Build the binary**

Run: `make build`
Expected: binary built to `bin/devkit`

**Step 2: Verify run --help works**

Run: `./bin/devkit run --help`
Expected: shows all flags with descriptions

**Step 3: Verify run without --board fails gracefully**

Run: `./bin/devkit run`
Expected: "Error: --board is required"

**Step 4: Verify run without trello login fails gracefully**

Run: `./bin/devkit run --board "Test Board" --dry-run`
Expected: "Not logged in to Trello. Run: devkit login trello" (unless already logged in)

**Step 5: Run all tests**

Run: `go test ./... -v`
Expected: all PASS

**Step 6: Commit (if any fixes were needed)**

```bash
git commit -am "fix: integration test fixes"
```

---

### Task 12: Update Makefile & CLAUDE.md

**Files:**
- Modify: `Makefile` — ensure `make build` and `make test` cover new packages
- Modify: `CLAUDE.md` — document `devkit run` command

**Step 1: Add run command docs to CLAUDE.md**

Add under Build & Development Commands:

```markdown
### Task Runner

```bash
make run ARGS="run --board 'Sprint Board'"                  # Run task runner
make run ARGS="run --board 'Sprint Board' --once --dry-run" # Test with one card, no execution
```
```

**Step 2: Verify make targets work**

Run: `make build && make test`
Expected: builds and all tests pass

**Step 3: Commit**

```bash
git add Makefile CLAUDE.md
git commit -m "docs: add devkit run command documentation"
```
