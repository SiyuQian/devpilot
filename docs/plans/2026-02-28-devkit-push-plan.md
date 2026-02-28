# devkit push Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `devkit push` CLI command that reads a plan markdown file and creates a Trello card from it, feeding into the `devkit run` workflow.

**Architecture:** New `CreateCard` method on the existing Trello client, plus a new Cobra command in `internal/cli/push.go` that reads a file, extracts the title from the first `# Heading`, resolves board/list by name, and creates the card.

**Tech Stack:** Go + Cobra, existing `internal/trello` client, existing `internal/config` credential loader

---

### Task 1: Add `URL` field to Card struct

**Files:**
- Modify: `internal/trello/types.go:13-18`

**Step 1: Add the ShortURL field**

Edit `internal/trello/types.go` — add a `ShortURL` field to the `Card` struct:

```go
type Card struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	IDList   string `json:"idList"`
	ShortURL string `json:"shortUrl"`
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/trello/`
Expected: no errors

**Step 3: Commit**

```bash
git add internal/trello/types.go
git commit -m "feat(trello): add ShortURL field to Card struct"
```

---

### Task 2: Add `CreateCard` method with test

**Files:**
- Modify: `internal/trello/client.go` (add method after `AddComment`)
- Modify: `internal/trello/client_test.go` (add test at end)

**Step 1: Write the failing test**

Append to `internal/trello/client_test.go`:

```go
func TestCreateCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/1/cards" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("idList") != "list1" {
			t.Errorf("expected idList=list1, got %s", r.URL.Query().Get("idList"))
		}
		if r.URL.Query().Get("name") != "My Card" {
			t.Errorf("expected name=My Card, got %s", r.URL.Query().Get("name"))
		}
		if r.URL.Query().Get("desc") != "card body" {
			t.Errorf("expected desc=card body, got %s", r.URL.Query().Get("desc"))
		}
		fmt.Fprint(w, `{"id":"card99","name":"My Card","desc":"card body","idList":"list1","shortUrl":"https://trello.com/c/abc123"}`)
	}))
	defer server.Close()

	client := NewClient("k", "t", WithBaseURL(server.URL))
	card, err := client.CreateCard("list1", "My Card", "card body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.ID != "card99" {
		t.Errorf("expected card99, got %s", card.ID)
	}
	if card.ShortURL != "https://trello.com/c/abc123" {
		t.Errorf("expected shortUrl, got %s", card.ShortURL)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/trello/ -run TestCreateCard -v`
Expected: FAIL — `client.CreateCard` does not exist

**Step 3: Write the implementation**

Add to `internal/trello/client.go` after the `AddComment` method (after line 165):

```go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/trello/ -run TestCreateCard -v`
Expected: PASS

**Step 5: Run all trello tests to ensure nothing broke**

Run: `go test ./internal/trello/ -v`
Expected: all tests PASS

**Step 6: Commit**

```bash
git add internal/trello/client.go internal/trello/client_test.go
git commit -m "feat(trello): add CreateCard method"
```

---

### Task 3: Create the `devkit push` command

**Files:**
- Create: `internal/cli/push.go`

**Step 1: Create the command file**

Create `internal/cli/push.go`:

```go
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/config"
	"github.com/siyuqian/developer-kit/internal/trello"
)

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
		title := extractTitle(string(content))
		if title == "" {
			fmt.Fprintln(os.Stderr, "Error: no # heading found in file")
			os.Exit(1)
		}

		// Load Trello credentials
		creds, err := config.Load("trello")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Not logged in to Trello. Run: devkit login trello")
			os.Exit(1)
		}

		client := trello.NewClient(creds["api_key"], creds["token"])

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

func extractTitle(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

func init() {
	pushCmd.Flags().String("board", "", "Trello board name (required)")
	pushCmd.Flags().String("list", "Ready", "Target list name")
	rootCmd.AddCommand(pushCmd)
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/cli/`
Expected: no errors

**Step 3: Verify the full binary builds**

Run: `go build -o bin/devkit ./cmd/devkit/`
Expected: no errors

**Step 4: Verify help text appears**

Run: `./bin/devkit push --help`
Expected: shows usage with `push <plan-file>`, `--board`, and `--list` flags

**Step 5: Commit**

```bash
git add internal/cli/push.go
git commit -m "feat(cli): add devkit push command"
```

---

### Task 4: Add unit test for `extractTitle`

**Files:**
- Create: `internal/cli/push_test.go`

**Step 1: Write the test**

Create `internal/cli/push_test.go`:

```go
package cli

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

**Step 2: Run the test**

Run: `go test ./internal/cli/ -run TestExtractTitle -v`
Expected: all subtests PASS

**Step 3: Run all tests to make sure nothing is broken**

Run: `go test ./...`
Expected: all PASS

**Step 4: Commit**

```bash
git add internal/cli/push_test.go
git commit -m "test(cli): add extractTitle unit tests"
```
