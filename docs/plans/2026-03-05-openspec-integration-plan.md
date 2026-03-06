# OpenSpec Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace `devpilot push` with `devpilot sync` that reads OpenSpec `changes/` directory and syncs to Trello/GitHub Issues; update the runner to use `opsx:apply` for execution.

**Architecture:** New `internal/openspec/` package wraps the OpenSpec CLI and filesystem. `devpilot sync` scans `openspec/changes/`, builds description from proposal.md + tasks.md, and creates/updates cards via existing TaskSource-compatible code. Runner's `buildPrompt()` switches to `/opsx:apply <change-name>` when OpenSpec is detected.

**Tech Stack:** Go, Cobra CLI, OpenSpec npm CLI (external dependency), existing Trello client + gh CLI

---

### Task 1: OpenSpec Directory Scanner

**Files:**
- Create: `internal/openspec/openspec.go`
- Test: `internal/openspec/openspec_test.go`

**Step 1: Write the failing test**

```go
package openspec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanChanges_empty(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "openspec", "changes"), 0755)
	changes, err := ScanChanges(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
}

func TestScanChanges_findsChanges(t *testing.T) {
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "openspec", "changes", "add-auth")
	os.MkdirAll(changeDir, 0755)
	os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("# Add Auth\nAdd authentication"), 0644)
	os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [ ] Task 1\n- [ ] Task 2"), 0644)

	changes, err := ScanChanges(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Name != "add-auth" {
		t.Errorf("expected name add-auth, got %s", changes[0].Name)
	}
	if changes[0].Description == "" {
		t.Error("expected non-empty description")
	}
}

func TestScanChanges_noOpenSpecDir(t *testing.T) {
	dir := t.TempDir()
	_, err := ScanChanges(dir)
	if err == nil {
		t.Error("expected error when openspec/changes/ does not exist")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/openspec/ -v -run TestScanChanges`
Expected: FAIL — package does not exist

**Step 3: Write minimal implementation**

```go
package openspec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Change represents a single OpenSpec change proposal.
type Change struct {
	Name        string // directory name, used as card title and opsx:apply argument
	Description string // combined content of proposal.md + tasks.md
}

// ScanChanges reads openspec/changes/ and returns all change proposals.
func ScanChanges(projectDir string) ([]Change, error) {
	changesDir := filepath.Join(projectDir, "openspec", "changes")
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return nil, fmt.Errorf("read openspec/changes/: %w", err)
	}

	var changes []Change
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		desc := buildDescription(filepath.Join(changesDir, name))
		if desc == "" {
			continue
		}
		changes = append(changes, Change{Name: name, Description: desc})
	}
	return changes, nil
}

// buildDescription concatenates proposal.md and tasks.md content.
func buildDescription(changeDir string) string {
	var parts []string
	for _, file := range []string{"proposal.md", "tasks.md"} {
		data, err := os.ReadFile(filepath.Join(changeDir, file))
		if err != nil {
			continue
		}
		if content := strings.TrimSpace(string(data)); content != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n\n---\n\n")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/openspec/ -v -run TestScanChanges`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/openspec/openspec.go internal/openspec/openspec_test.go
git commit -m "feat(openspec): add change directory scanner"
```

---

### Task 2: OpenSpec Version Checker

**Files:**
- Modify: `internal/openspec/openspec.go`
- Test: `internal/openspec/openspec_test.go`

**Step 1: Write the failing test**

```go
func TestCheckInstalled_notInstalled(t *testing.T) {
	err := CheckInstalled("nonexistent-binary-xyz")
	if err == nil {
		t.Error("expected error when binary not found")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/openspec/ -v -run TestCheckInstalled`
Expected: FAIL — `CheckInstalled` not defined

**Step 3: Write minimal implementation**

Add to `internal/openspec/openspec.go`:

```go
import "os/exec"

// CheckInstalled verifies that the OpenSpec CLI is available.
func CheckInstalled(binary string) error {
	_, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf("openspec CLI not found. Install with: npm install -g @fission-ai/openspec@latest")
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/openspec/ -v -run TestCheckInstalled`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/openspec/openspec.go internal/openspec/openspec_test.go
git commit -m "feat(openspec): add CLI installation checker"
```

---

### Task 3: Trello Client — UpdateCard Method

The sync command needs to update existing cards (idempotency). The Trello client currently has `CreateCard` but no `UpdateCard`.

**Files:**
- Modify: `internal/trello/client.go`
- Modify: `internal/trello/client_test.go`

**Step 1: Write the failing test**

Add to `internal/trello/client_test.go`:

```go
func TestUpdateCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/1/cards/card1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("desc") != "updated body" {
			t.Errorf("expected desc=updated body, got %s", r.URL.Query().Get("desc"))
		}
		fmt.Fprint(w, `{"id":"card1","name":"My Card","desc":"updated body"}`)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))
	err := client.UpdateCard("card1", "updated body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/trello/ -v -run TestUpdateCard`
Expected: FAIL — `UpdateCard` not defined

**Step 3: Write minimal implementation**

Add to `internal/trello/client.go`:

```go
func (c *Client) UpdateCard(cardID, desc string) error {
	params := url.Values{"desc": {desc}}
	_, err := c.put(fmt.Sprintf("/1/cards/%s", cardID), params)
	return err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/trello/ -v -run TestUpdateCard`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/trello/client.go internal/trello/client_test.go
git commit -m "feat(trello): add UpdateCard method for idempotent sync"
```

---

### Task 4: Trello Client — FindCardByName Method

Sync needs to find existing cards by name to decide create vs update.

**Files:**
- Modify: `internal/trello/client.go`
- Modify: `internal/trello/client_test.go`

**Step 1: Write the failing test**

Add to `internal/trello/client_test.go`:

```go
func TestFindCardByName(t *testing.T) {
	cards := []Card{
		{ID: "c1", Name: "add-auth", Desc: "old"},
		{ID: "c2", Name: "fix-bug", Desc: "other"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(cards)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))

	card, err := client.FindCardByName("list1", "add-auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.ID != "c1" {
		t.Errorf("expected c1, got %s", card.ID)
	}

	card, err = client.FindCardByName("list1", "nonexistent")
	if err != nil {
		t.Error("expected nil error for not found")
	}
	if card != nil {
		t.Errorf("expected nil card, got %+v", card)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/trello/ -v -run TestFindCardByName`
Expected: FAIL — `FindCardByName` not defined

**Step 3: Write minimal implementation**

Add to `internal/trello/client.go`:

```go
// FindCardByName searches for a card by name in a list. Returns nil, nil if not found.
func (c *Client) FindCardByName(listID, name string) (*Card, error) {
	cards, err := c.GetListCards(listID)
	if err != nil {
		return nil, err
	}
	for _, card := range cards {
		if card.Name == name {
			return &card, nil
		}
	}
	return nil, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/trello/ -v -run TestFindCardByName`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/trello/client.go internal/trello/client_test.go
git commit -m "feat(trello): add FindCardByName for idempotent sync"
```

---

### Task 5: Sync Logic — Core sync function

**Files:**
- Create: `internal/openspec/sync.go`
- Create: `internal/openspec/sync_test.go`

**Step 1: Write the failing test**

```go
package openspec

import (
	"testing"
)

// SyncTarget abstracts Trello/GitHub for testing.
type mockTarget struct {
	created []struct{ name, desc string }
	updated []struct{ id, desc string }
	cards   map[string]string // name -> id
}

func (m *mockTarget) FindByName(name string) (string, error) {
	if id, ok := m.cards[name]; ok {
		return id, nil
	}
	return "", nil
}

func (m *mockTarget) Create(name, desc string) error {
	m.created = append(m.created, struct{ name, desc string }{name, desc})
	return nil
}

func (m *mockTarget) Update(id, desc string) error {
	m.updated = append(m.updated, struct{ id, desc string }{id, desc})
	return nil
}

func TestSync_createsNew(t *testing.T) {
	target := &mockTarget{cards: map[string]string{}}
	changes := []Change{{Name: "add-auth", Description: "the plan"}}

	results, err := Sync(changes, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Action != "created" {
		t.Errorf("expected 1 created result, got %+v", results)
	}
	if len(target.created) != 1 {
		t.Errorf("expected 1 create call, got %d", len(target.created))
	}
}

func TestSync_updatesExisting(t *testing.T) {
	target := &mockTarget{cards: map[string]string{"add-auth": "card1"}}
	changes := []Change{{Name: "add-auth", Description: "updated plan"}}

	results, err := Sync(changes, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Action != "updated" {
		t.Errorf("expected 1 updated result, got %+v", results)
	}
	if len(target.updated) != 1 || target.updated[0].id != "card1" {
		t.Errorf("expected update to card1, got %+v", target.updated)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/openspec/ -v -run TestSync`
Expected: FAIL — `Sync` and `SyncTarget` not defined

**Step 3: Write minimal implementation**

Create `internal/openspec/sync.go`:

```go
package openspec

// SyncTarget abstracts the task board (Trello or GitHub Issues).
type SyncTarget interface {
	FindByName(name string) (id string, err error) // returns "" if not found
	Create(name, desc string) error
	Update(id, desc string) error
}

// SyncResult describes what happened to each change during sync.
type SyncResult struct {
	Name   string // change name
	Action string // "created" or "updated"
}

// Sync creates or updates board items for each change.
func Sync(changes []Change, target SyncTarget) ([]SyncResult, error) {
	var results []SyncResult
	for _, ch := range changes {
		existingID, err := target.FindByName(ch.Name)
		if err != nil {
			return results, err
		}
		if existingID != "" {
			if err := target.Update(existingID, ch.Description); err != nil {
				return results, err
			}
			results = append(results, SyncResult{Name: ch.Name, Action: "updated"})
		} else {
			if err := target.Create(ch.Name, ch.Description); err != nil {
				return results, err
			}
			results = append(results, SyncResult{Name: ch.Name, Action: "created"})
		}
	}
	return results, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/openspec/ -v -run TestSync`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/openspec/sync.go internal/openspec/sync_test.go
git commit -m "feat(openspec): add sync logic with SyncTarget interface"
```

---

### Task 6: Trello SyncTarget Adapter

**Files:**
- Create: `internal/openspec/trello_target.go`
- Create: `internal/openspec/trello_target_test.go`

**Step 1: Write the failing test**

```go
package openspec

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/siyuqian/devpilot/internal/trello"
)

func TestTrelloTarget_FindByName(t *testing.T) {
	cards := []trello.Card{{ID: "c1", Name: "add-auth"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(cards)
	}))
	defer server.Close()

	client := trello.NewClient("k", "t", trello.WithBaseURL(server.URL))
	target := NewTrelloTarget(client, "list1")

	id, err := target.FindByName("add-auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "c1" {
		t.Errorf("expected c1, got %s", id)
	}

	id, err = target.FindByName("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "" {
		t.Errorf("expected empty id, got %s", id)
	}
}

func TestTrelloTarget_Create(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		fmt.Fprint(w, `{"id":"card99","name":"add-auth"}`)
	}))
	defer server.Close()

	client := trello.NewClient("k", "t", trello.WithBaseURL(server.URL))
	target := NewTrelloTarget(client, "list1")

	err := target.Create("add-auth", "the plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTrelloTarget_Update(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		fmt.Fprint(w, `{"id":"c1"}`)
	}))
	defer server.Close()

	client := trello.NewClient("k", "t", trello.WithBaseURL(server.URL))
	target := NewTrelloTarget(client, "list1")

	err := target.Update("c1", "updated plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/openspec/ -v -run TestTrelloTarget`
Expected: FAIL — `NewTrelloTarget` not defined

**Step 3: Write minimal implementation**

Create `internal/openspec/trello_target.go`:

```go
package openspec

import "github.com/siyuqian/devpilot/internal/trello"

// TrelloTarget adapts trello.Client to the SyncTarget interface.
type TrelloTarget struct {
	client *trello.Client
	listID string
}

func NewTrelloTarget(client *trello.Client, listID string) *TrelloTarget {
	return &TrelloTarget{client: client, listID: listID}
}

func (t *TrelloTarget) FindByName(name string) (string, error) {
	card, err := t.client.FindCardByName(t.listID, name)
	if err != nil {
		return "", err
	}
	if card == nil {
		return "", nil
	}
	return card.ID, nil
}

func (t *TrelloTarget) Create(name, desc string) error {
	_, err := t.client.CreateCard(t.listID, name, desc)
	return err
}

func (t *TrelloTarget) Update(id, desc string) error {
	return t.client.UpdateCard(id, desc)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/openspec/ -v -run TestTrelloTarget`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/openspec/trello_target.go internal/openspec/trello_target_test.go
git commit -m "feat(openspec): add Trello SyncTarget adapter"
```

---

### Task 7: GitHub SyncTarget Adapter

**Files:**
- Create: `internal/openspec/github_target.go`
- Create: `internal/openspec/github_target_test.go`

**Step 1: Write the failing test**

```go
package openspec

import (
	"os/exec"
	"testing"
)

func TestGitHubTarget_interfaceCompliance(t *testing.T) {
	// Verify GitHubTarget implements SyncTarget at compile time.
	var _ SyncTarget = (*GitHubTarget)(nil)
}

func TestGitHubTarget_buildFindCommand(t *testing.T) {
	target := NewGitHubTarget()
	args := target.findArgs("add-auth")
	expected := []string{"issue", "list", "--label", "devpilot", "--state", "open",
		"--search", "add-auth in:title", "--json", "number,title", "--limit", "5"}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, arg := range expected {
		if args[i] != arg {
			t.Errorf("arg[%d]: expected %q, got %q", i, arg, args[i])
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/openspec/ -v -run TestGitHubTarget`
Expected: FAIL — `GitHubTarget` not defined

**Step 3: Write minimal implementation**

Create `internal/openspec/github_target.go`:

```go
package openspec

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// GitHubTarget adapts the gh CLI to the SyncTarget interface.
type GitHubTarget struct{}

func NewGitHubTarget() *GitHubTarget {
	return &GitHubTarget{}
}

func (g *GitHubTarget) findArgs(name string) []string {
	return []string{"issue", "list", "--label", "devpilot", "--state", "open",
		"--search", name + " in:title", "--json", "number,title", "--limit", "5"}
}

func (g *GitHubTarget) FindByName(name string) (string, error) {
	out, err := exec.Command("gh", g.findArgs(name)...).Output()
	if err != nil {
		return "", fmt.Errorf("gh issue list: %w", err)
	}
	var issues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(out, &issues); err != nil {
		return "", fmt.Errorf("parse issues: %w", err)
	}
	for _, issue := range issues {
		if issue.Title == name {
			return fmt.Sprintf("%d", issue.Number), nil
		}
	}
	return "", nil
}

func (g *GitHubTarget) Create(name, desc string) error {
	_, err := exec.Command("gh", "issue", "create",
		"--title", name,
		"--body", desc,
		"--label", "devpilot",
	).Output()
	if err != nil {
		return fmt.Errorf("create issue: %w", err)
	}
	return nil
}

func (g *GitHubTarget) Update(id, desc string) error {
	_, err := exec.Command("gh", "issue", "edit", id, "--body", desc).Output()
	if err != nil {
		return fmt.Errorf("update issue %s: %w", id, err)
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/openspec/ -v -run TestGitHubTarget`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/openspec/github_target.go internal/openspec/github_target_test.go
git commit -m "feat(openspec): add GitHub SyncTarget adapter"
```

---

### Task 8: `devpilot sync` Cobra Command

**Files:**
- Create: `internal/openspec/commands.go`
- Modify: `cmd/devpilot/main.go` (add import + registration)

**Step 1: Write the command**

Create `internal/openspec/commands.go`:

```go
package openspec

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/siyuqian/devpilot/internal/project"
	"github.com/siyuqian/devpilot/internal/trello"
)

func RegisterCommands(parent *cobra.Command) {
	syncCmd.Flags().String("board", "", "Trello board name (required for trello source)")
	syncCmd.Flags().String("source", "", "Task source: trello or github (default from .devpilot.json)")
	syncCmd.Flags().String("list", "Ready", "Target list name (trello only)")
	parent.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync OpenSpec changes to task board",
	Long:  "Scan openspec/changes/ and create or update cards/issues for each change proposal.",
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to get working directory:", err)
			os.Exit(1)
		}

		// Check OpenSpec is installed
		if err := CheckInstalled("openspec"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Scan changes
		changes, err := ScanChanges(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning changes: %v\n", err)
			os.Exit(1)
		}
		if len(changes) == 0 {
			fmt.Println("No changes found in openspec/changes/")
			return
		}

		// Resolve source
		projectCfg, _ := project.Load(dir)
		sourceName, _ := cmd.Flags().GetString("source")
		sourceName = projectCfg.ResolveSource(sourceName)

		boardName, _ := cmd.Flags().GetString("board")
		if boardName == "" && projectCfg.Board != "" {
			boardName = projectCfg.Board
		}

		listName, _ := cmd.Flags().GetString("list")

		// Build target
		var target SyncTarget
		switch sourceName {
		case "trello":
			if boardName == "" {
				fmt.Fprintln(os.Stderr, "Error: --board is required for trello source (or run: devpilot init)")
				os.Exit(1)
			}
			creds, err := auth.Load("trello")
			if err != nil {
				fmt.Fprintln(os.Stderr, "Not logged in to Trello. Run: devpilot login trello")
				os.Exit(1)
			}
			client := trello.NewClient(creds["api_key"], creds["token"])
			board, err := client.FindBoardByName(boardName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			list, err := client.FindListByName(board.ID, listName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			target = NewTrelloTarget(client, list.ID)
		case "github":
			target = NewGitHubTarget()
		default:
			fmt.Fprintf(os.Stderr, "Unknown source %q\n", sourceName)
			os.Exit(1)
		}

		// Sync
		results, err := Sync(changes, target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Sync error: %v\n", err)
			os.Exit(1)
		}

		for _, r := range results {
			fmt.Printf("%s: %s\n", r.Action, r.Name)
		}
		fmt.Printf("\nSynced %d change(s) to %s\n", len(results), sourceName)
	},
}
```

**Step 2: Register in main.go**

Modify `cmd/devpilot/main.go` — add import and registration:

```go
import (
	// ... existing imports ...
	"github.com/siyuqian/devpilot/internal/openspec"
)

// In main(), add after existing RegisterCommands:
openspec.RegisterCommands(rootCmd)
```

**Step 3: Build and verify**

Run: `make build && bin/devpilot sync --help`
Expected: Shows sync command help with --board, --source, --list flags

**Step 4: Commit**

```bash
git add internal/openspec/commands.go cmd/devpilot/main.go
git commit -m "feat: add devpilot sync command"
```

---

### Task 9: Deprecate `devpilot push`

**Files:**
- Modify: `internal/trello/commands.go`

**Step 1: Add deprecation warning**

In `internal/trello/commands.go`, add a deprecation notice at the start of `pushCmd.Run`:

```go
Run: func(cmd *cobra.Command, args []string) {
	fmt.Fprintln(os.Stderr, "WARNING: 'devpilot push' is deprecated. Use OpenSpec + 'devpilot sync' instead.")
	fmt.Fprintln(os.Stderr, "  See: https://github.com/Fission-AI/OpenSpec")
	fmt.Fprintln(os.Stderr, "")
	// ... rest of existing code unchanged ...
```

**Step 2: Build and verify**

Run: `make build && bin/devpilot push --help`
Expected: Command still works, shows in help

**Step 3: Commit**

```bash
git add internal/trello/commands.go
git commit -m "deprecate: add warning to devpilot push, recommend sync"
```

---

### Task 10: Runner — Use opsx:apply When OpenSpec Detected

**Files:**
- Modify: `internal/taskrunner/runner.go`
- Modify: `internal/taskrunner/runner_test.go` (if exists, otherwise create)

**Step 1: Write the failing test**

Create or add to test file:

```go
package taskrunner

import "testing"

func TestBuildPrompt_withOpenSpec(t *testing.T) {
	r := &Runner{config: Config{UseOpenSpec: true}}
	task := Task{Name: "add-auth", Description: "the plan"}
	prompt := r.buildPrompt(task)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	expected := "/opsx:apply add-auth"
	if !strings.Contains(prompt, expected) {
		t.Errorf("expected prompt to contain %q, got:\n%s", expected, prompt)
	}
}

func TestBuildPrompt_withoutOpenSpec(t *testing.T) {
	r := &Runner{config: Config{UseOpenSpec: false}}
	task := Task{Name: "add-auth", Description: "the plan"}
	prompt := r.buildPrompt(task)
	if !strings.Contains(prompt, "the plan") {
		t.Errorf("expected prompt to contain plan description, got:\n%s", prompt)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/taskrunner/ -v -run TestBuildPrompt`
Expected: FAIL — `UseOpenSpec` field does not exist on `Config`

**Step 3: Write minimal implementation**

Modify `internal/taskrunner/runner.go`:

Add `UseOpenSpec bool` to `Config` struct:

```go
type Config struct {
	BoardName     string
	Interval      time.Duration
	Timeout       time.Duration
	ReviewTimeout time.Duration
	Once          bool
	DryRun        bool
	WorkDir       string
	UseOpenSpec   bool
}
```

Replace `buildPrompt`:

```go
func (r *Runner) buildPrompt(task Task) string {
	if r.config.UseOpenSpec {
		return fmt.Sprintf(`Execute the following OpenSpec change autonomously from start to finish. This runs unattended — never stop to ask for feedback, confirmation, or approval.

Run: /opsx:apply %s

Rules:
- Execute ALL tasks without stopping
- Commit after each logical unit of work
- Never ask for user input or feedback
- If a task is blocked, skip it and continue with the next task
- When ALL tasks are complete, push to the current branch`, task.Name)
	}
	return fmt.Sprintf(`Execute the following task plan autonomously from start to finish. This runs unattended — never stop to ask for feedback, confirmation, or approval. Execute ALL steps/batches continuously without pausing.

Use /superpowers:test-driven-development and /superpowers:verification-before-completion skills during execution.

Task: %s

Plan:
%s

Rules:
- Execute ALL steps in the plan without stopping. Do NOT pause between batches or steps for review.
- Commit after each logical unit of work
- Never ask for user input or feedback
- If a step is blocked, skip it and continue with the next step
- When ALL steps are complete, push to the current branch`, task.Name, task.Description)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/taskrunner/ -v -run TestBuildPrompt`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/taskrunner/runner.go
git commit -m "feat(runner): use opsx:apply when UseOpenSpec is enabled"
```

---

### Task 11: Wire UseOpenSpec in Runner Commands

**Files:**
- Modify: `internal/taskrunner/commands.go`

**Step 1: Detect OpenSpec and set config**

In `internal/taskrunner/commands.go`, add OpenSpec detection after loading project config:

```go
import "github.com/siyuqian/devpilot/internal/openspec"

// Inside runCmd.Run, after projectCfg is loaded and before building cfg:
useOpenSpec := false
if openspec.CheckInstalled("openspec") == nil {
	if _, err := openspec.ScanChanges(dir); err == nil {
		useOpenSpec = true
	}
}

// Add to cfg:
cfg := Config{
	// ... existing fields ...
	UseOpenSpec: useOpenSpec,
}
```

**Step 2: Build and verify**

Run: `make build && bin/devpilot run --help`
Expected: Builds successfully, no new flags needed (auto-detected)

**Step 3: Run all tests**

Run: `make test`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/taskrunner/commands.go
git commit -m "feat(runner): auto-detect OpenSpec and enable opsx:apply"
```

---

### Task 12: Config — Add openspecMinVersion

**Files:**
- Modify: `internal/project/config.go`
- Modify: `internal/project/config_test.go`

**Step 1: Write the failing test**

Add to `internal/project/config_test.go`:

```go
func TestConfig_OpenSpecMinVersion(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Board:              "devpilot",
		Source:             "github",
		OpenSpecMinVersion: "1.2.0",
	}
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.OpenSpecMinVersion != "1.2.0" {
		t.Errorf("expected 1.2.0, got %s", loaded.OpenSpecMinVersion)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/project/ -v -run TestConfig_OpenSpecMinVersion`
Expected: FAIL — `OpenSpecMinVersion` not defined

**Step 3: Write minimal implementation**

Add field to `Config` in `internal/project/config.go`:

```go
type Config struct {
	Board              string            `json:"board,omitempty"`
	Source             string            `json:"source,omitempty"`
	Models             map[string]string `json:"models,omitempty"`
	OpenSpecMinVersion string            `json:"openspecMinVersion,omitempty"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/project/ -v -run TestConfig_OpenSpecMinVersion`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/project/config.go internal/project/config_test.go
git commit -m "feat(config): add openspecMinVersion field"
```

---

### Task 13: Final Integration Test — Full Sync Flow

**Files:**
- Create: `internal/openspec/integration_test.go`

**Step 1: Write the integration test**

```go
package openspec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFullSyncFlow(t *testing.T) {
	// Setup: create a mock project with openspec/changes/
	dir := t.TempDir()

	// Create two changes
	for _, name := range []string{"add-auth", "fix-bug"} {
		changeDir := filepath.Join(dir, "openspec", "changes", name)
		os.MkdirAll(changeDir, 0755)
		os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("# "+name+"\nDescription"), 0644)
		os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("- [ ] Task 1"), 0644)
	}

	// Scan
	changes, err := ScanChanges(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	// Sync to mock target — first time creates
	target := &mockTarget{cards: map[string]string{}}
	results, err := Sync(changes, target)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Action != "created" {
			t.Errorf("expected created, got %s for %s", r.Action, r.Name)
		}
	}

	// Sync again with existing cards — should update
	target2 := &mockTarget{cards: map[string]string{"add-auth": "c1", "fix-bug": "c2"}}
	results2, err := Sync(changes, target2)
	if err != nil {
		t.Fatalf("sync2: %v", err)
	}
	for _, r := range results2 {
		if r.Action != "updated" {
			t.Errorf("expected updated, got %s for %s", r.Action, r.Name)
		}
	}
}
```

**Step 2: Run test**

Run: `go test ./internal/openspec/ -v -run TestFullSyncFlow`
Expected: PASS

**Step 3: Run all tests**

Run: `make test`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/openspec/integration_test.go
git commit -m "test(openspec): add full sync flow integration test"
```

---

### Task 14: Update Design Doc and CLAUDE.md

**Files:**
- Modify: `docs/plans/2026-03-05-openspec-integration-design.md` (mark status as Implemented)
- Modify: `CLAUDE.md` (add sync command and OpenSpec info)

**Step 1: Update design doc status**

Change `**Status:** Draft` to `**Status:** Implemented`

**Step 2: Update CLAUDE.md**

Add to the CLI Commands section:

```markdown
devpilot sync                                              # Sync OpenSpec changes to board/issues
devpilot sync --board "Board Name"                         # Override board
devpilot sync --source github                              # Override source
```

Add to the Architecture section, after Task Runner:

```markdown
### OpenSpec Integration

When OpenSpec is installed and `openspec/changes/` exists:
- `devpilot sync` scans changes and creates/updates Trello cards or GitHub Issues
- Card title = change directory name (used as `opsx:apply` argument)
- Card description = full content of proposal.md + tasks.md
- Runner auto-detects OpenSpec and uses `/opsx:apply <change-name>` instead of raw plan text
- Supports resumability: interrupted tasks pick up from last unchecked task
```

**Step 3: Commit**

```bash
git add docs/plans/2026-03-05-openspec-integration-design.md CLAUDE.md
git commit -m "docs: update design doc status and add OpenSpec to CLAUDE.md"
```
