# GitHub Issues Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add GitHub Issues as a selectable alternative to Trello for task management in the devpilot runner.

**Architecture:** Introduce a `TaskSource` interface in the `taskrunner` package. Both the existing Trello integration and the new GitHub Issues integration implement this interface. The `Runner` depends only on the interface, not on `*trello.Client` directly. All adapters live in `internal/taskrunner/` to avoid circular imports.

**Tech Stack:** Go, Cobra, `gh` CLI (already a dep), existing `trello.Client`

---

### Task 1: Define Task struct, SourceInfo, and TaskSource interface

**Files:**
- Create: `internal/taskrunner/source.go`
- Create: `internal/taskrunner/source_test.go`

**Step 1: Write failing test**

```go
// internal/taskrunner/source_test.go
package taskrunner

import "testing"

func TestTaskDefaultPriority(t *testing.T) {
	task := Task{ID: "1", Name: "Test"}
	if task.Priority != 0 {
		t.Errorf("expected zero Priority, got %d", task.Priority)
	}
}

func TestSourceInfoZeroValue(t *testing.T) {
	var info SourceInfo
	if info.DisplayName != "" || info.BoardID != "" || info.Lists != nil {
		t.Error("SourceInfo zero value should have empty fields")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/taskrunner/ -run TestTask -v
go test ./internal/taskrunner/ -run TestSourceInfo -v
```
Expected: FAIL — `Task`, `SourceInfo` not defined

**Step 3: Write implementation**

```go
// internal/taskrunner/source.go
package taskrunner

// Task is a provider-agnostic unit of work.
type Task struct {
	ID          string
	Name        string
	Description string
	URL         string
	Priority    int // 0=P0, 1=P1, 2=P2 (default)
}

// SourceInfo is returned by TaskSource.Init and used to populate RunnerStartedEvent.
type SourceInfo struct {
	DisplayName string
	BoardID     string            // optional; empty for GitHub
	Lists       map[string]string // optional; nil for GitHub
}

// TaskSource is the interface for task management backends.
type TaskSource interface {
	Init() (SourceInfo, error)
	FetchReady() ([]Task, error)
	MarkInProgress(id string) error
	MarkDone(id, comment string) error
	MarkFailed(id, comment string) error
}
```

**Step 4: Run tests**

```bash
go test ./internal/taskrunner/ -run TestTask -v
go test ./internal/taskrunner/ -run TestSourceInfo -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/taskrunner/source.go internal/taskrunner/source_test.go
git commit -m "feat(taskrunner): add TaskSource interface and Task struct"
```

---

### Task 2: Refactor priority.go to use Task

**Files:**
- Modify: `internal/taskrunner/priority.go`
- Modify: `internal/taskrunner/priority_test.go`

**Step 1: Update tests first**

Replace the entire `priority_test.go` content — change all `[]trello.Card` and `trello.Label` to `[]Task` with `Priority` pre-set:

```go
// internal/taskrunner/priority_test.go
package taskrunner

import "testing"

func TestSortByPriority_AllPriorities(t *testing.T) {
	tasks := []Task{
		{ID: "c3", Name: "Low", Priority: 2},
		{ID: "c1", Name: "Critical", Priority: 0},
		{ID: "c2", Name: "High", Priority: 1},
	}
	SortByPriority(tasks)
	if tasks[0].ID != "c1" {
		t.Errorf("expected P0 first, got %s", tasks[0].ID)
	}
	if tasks[1].ID != "c2" {
		t.Errorf("expected P1 second, got %s", tasks[1].ID)
	}
	if tasks[2].ID != "c3" {
		t.Errorf("expected P2 third, got %s", tasks[2].ID)
	}
}

func TestSortByPriority_DefaultP2(t *testing.T) {
	tasks := []Task{
		{ID: "c1", Name: "No priority", Priority: 2},
		{ID: "c2", Name: "Critical", Priority: 0},
	}
	SortByPriority(tasks)
	if tasks[0].ID != "c2" {
		t.Errorf("expected P0 first, got %s", tasks[0].ID)
	}
}

func TestSortByPriority_StableSort(t *testing.T) {
	tasks := []Task{
		{ID: "c1", Priority: 1},
		{ID: "c2", Priority: 1},
		{ID: "c3", Priority: 1},
	}
	SortByPriority(tasks)
	if tasks[0].ID != "c1" || tasks[1].ID != "c2" || tasks[2].ID != "c3" {
		t.Errorf("stable sort not preserved: got %s, %s, %s", tasks[0].ID, tasks[1].ID, tasks[2].ID)
	}
}

func TestSortByPriority_EmptySlice(t *testing.T) {
	var tasks []Task
	SortByPriority(tasks) // should not panic
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/taskrunner/ -run TestSortByPriority -v
```
Expected: FAIL — `SortByPriority` still takes `[]trello.Card`

**Step 3: Update priority.go**

Replace the entire `priority.go`:

```go
// internal/taskrunner/priority.go
package taskrunner

import "sort"

// SortByPriority sorts tasks by Priority field (0=highest, 2=lowest).
// Stable sort preserves original order within the same priority.
func SortByPriority(tasks []Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].Priority < tasks[j].Priority
	})
}
```

**Step 4: Run tests**

```bash
go test ./internal/taskrunner/ -run TestSortByPriority -v
```
Expected: PASS

**Step 5: Verify the package still compiles (runner.go still imports trello — that's fine for now)**

```bash
go build ./internal/taskrunner/
```
Expected: compile error mentioning `SortByPriority` in runner.go — that's OK, we'll fix it in Task 4.

**Step 6: Commit**

```bash
git add internal/taskrunner/priority.go internal/taskrunner/priority_test.go
git commit -m "refactor(taskrunner): SortByPriority uses Task instead of trello.Card"
```

---

### Task 3: Create TrelloSource adapter

**Files:**
- Create: `internal/taskrunner/trello_source.go`
- Create: `internal/taskrunner/trello_source_test.go`

**Step 1: Write failing test**

```go
// internal/taskrunner/trello_source_test.go
package taskrunner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/siyuqian/devpilot/internal/trello"
)

func TestTrelloSource_FetchReady_MapsToTasks(t *testing.T) {
	cards := []trello.Card{
		{ID: "c1", Name: "Task 1", Desc: "Do something", ShortURL: "https://trello.com/c/c1",
			Labels: []trello.Label{{Name: "P0-critical"}}},
		{ID: "c2", Name: "Task 2", Desc: "Do another thing", ShortURL: "https://trello.com/c/c2"},
	}
	data, _ := json.Marshal(cards)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer ts.Close()

	client := trello.NewClient("key", "token", trello.WithBaseURL(ts.URL))
	source := &TrelloSource{client: client, readyListID: "list1"}

	tasks, err := source.FetchReady()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "c1" || tasks[0].Name != "Task 1" || tasks[0].Priority != 0 {
		t.Errorf("task 0 mismatch: %+v", tasks[0])
	}
	if tasks[1].Priority != 2 {
		t.Errorf("expected default P2 for unlabeled task, got %d", tasks[1].Priority)
	}
}

func TestParseTrelloPriority(t *testing.T) {
	cases := []struct {
		labels   []trello.Label
		expected int
	}{
		{[]trello.Label{{Name: "P0-critical"}}, 0},
		{[]trello.Label{{Name: "P1-high"}}, 1},
		{[]trello.Label{{Name: "P2-normal"}}, 2},
		{[]trello.Label{{Name: "p0-critical"}}, 0}, // case insensitive
		{[]trello.Label{{Name: "bug"}}, 2},          // non-priority label defaults to P2
		{nil, 2},
	}
	for _, c := range cases {
		card := trello.Card{Labels: c.labels}
		got := trelloPriority(card)
		if got != c.expected {
			t.Errorf("labels %v: expected %d, got %d", c.labels, c.expected, got)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/taskrunner/ -run TestTrelloSource -run TestParseTrello -v
```
Expected: FAIL — `TrelloSource` not defined

**Step 3: Write implementation**

```go
// internal/taskrunner/trello_source.go
package taskrunner

import (
	"fmt"
	"strings"

	"github.com/siyuqian/devpilot/internal/trello"
)

// TrelloSource implements TaskSource for Trello boards.
type TrelloSource struct {
	client       *trello.Client
	boardName    string
	readyListID  string
	inProgListID string
	doneListID   string
	failedListID string
}

func NewTrelloSource(client *trello.Client, boardName string) *TrelloSource {
	return &TrelloSource{client: client, boardName: boardName}
}

func (s *TrelloSource) Init() (SourceInfo, error) {
	board, err := s.client.FindBoardByName(s.boardName)
	if err != nil {
		return SourceInfo{}, fmt.Errorf("find board: %w", err)
	}

	listNames := map[string]*string{
		"Ready":       &s.readyListID,
		"In Progress": &s.inProgListID,
		"Done":        &s.doneListID,
		"Failed":      &s.failedListID,
	}
	resolved := make(map[string]string, len(listNames))
	for name, idPtr := range listNames {
		list, err := s.client.FindListByName(board.ID, name)
		if err != nil {
			return SourceInfo{}, fmt.Errorf("find list %q: %w", name, err)
		}
		*idPtr = list.ID
		resolved[name] = list.ID
	}
	return SourceInfo{
		DisplayName: board.Name,
		BoardID:     board.ID,
		Lists:       resolved,
	}, nil
}

func (s *TrelloSource) FetchReady() ([]Task, error) {
	cards, err := s.client.GetListCards(s.readyListID)
	if err != nil {
		return nil, err
	}
	tasks := make([]Task, 0, len(cards))
	for _, c := range cards {
		tasks = append(tasks, Task{
			ID:          c.ID,
			Name:        c.Name,
			Description: c.Desc,
			URL:         c.ShortURL,
			Priority:    trelloPriority(c),
		})
	}
	return tasks, nil
}

func (s *TrelloSource) MarkInProgress(id string) error {
	return s.client.MoveCard(id, s.inProgListID)
}

func (s *TrelloSource) MarkDone(id, comment string) error {
	if err := s.client.MoveCard(id, s.doneListID); err != nil {
		return err
	}
	return s.client.AddComment(id, comment)
}

func (s *TrelloSource) MarkFailed(id, comment string) error {
	if err := s.client.MoveCard(id, s.failedListID); err != nil {
		return err
	}
	return s.client.AddComment(id, comment)
}

func trelloPriority(c trello.Card) int {
	for _, label := range c.Labels {
		name := strings.ToUpper(label.Name)
		if strings.HasPrefix(name, "P0") {
			return 0
		}
		if strings.HasPrefix(name, "P1") {
			return 1
		}
		if strings.HasPrefix(name, "P2") {
			return 2
		}
	}
	return 2
}
```

**Step 4: Run tests**

```bash
go test ./internal/taskrunner/ -run TestTrelloSource -run TestParseTrello -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/taskrunner/trello_source.go internal/taskrunner/trello_source_test.go
git commit -m "feat(taskrunner): add TrelloSource adapter implementing TaskSource"
```

---

### Task 4: Refactor Runner to use TaskSource

**Files:**
- Modify: `internal/taskrunner/runner.go`

The runner currently depends on `*trello.Client` and holds list IDs directly. We replace all of that with `TaskSource`.

**Step 1: Replace the Runner struct and New() function**

In `internal/taskrunner/runner.go`, make these changes:

1. Remove the `trello` import.
2. Replace the `Runner` struct fields `trello *trello.Client`, `boardID`, `readyListID`, `inProgListID`, `doneListID`, `failedListID` with a single `source TaskSource`.
3. Update `New()` signature from `func New(cfg Config, trelloClient *trello.Client, opts ...RunnerOption)` to `func New(cfg Config, source TaskSource, opts ...RunnerOption)`.

New struct:
```go
type Runner struct {
	config       Config
	source       TaskSource
	executor     *Executor
	reviewer     *Reviewer
	git          *GitOps
	logger       *log.Logger
	eventHandler EventHandler
}
```

New `New()`:
```go
func New(cfg Config, source TaskSource, opts ...RunnerOption) *Runner {
	r := &Runner{
		config: cfg,
		source: source,
		git:    NewGitOps(cfg.WorkDir),
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
	for _, opt := range opts {
		opt(r)
	}
	if r.eventHandler != nil {
		r.logger = log.New(io.Discard, "", 0)
	}
	var execOpts []ExecutorOption
	if r.eventHandler != nil {
		bridge := newEventBridge(r.eventHandler)
		execOpts = append(execOpts, WithClaudeEventHandler(bridge.Handle))
	}
	r.executor = NewExecutor(execOpts...)
	if cfg.ReviewTimeout > 0 {
		r.reviewer = NewReviewer()
	}
	return r
}
```

**Step 2: Replace init()**

```go
func (r *Runner) init() error {
	r.logger.Printf("Initializing task source...")
	info, err := r.source.Init()
	if err != nil {
		return err
	}
	r.logger.Printf("Connected: %s", info.DisplayName)
	r.emit(RunnerStartedEvent{BoardName: info.DisplayName, BoardID: info.BoardID, Lists: info.Lists})
	return nil
}
```

**Step 3: Update Run() — replace FetchReady call**

In `Run()`, replace:
```go
cards, err := r.trello.GetListCards(r.readyListID)
```
with:
```go
cards, err := r.source.FetchReady()
```

Replace all `cards` variable usage:
- `SortByPriority(cards)` — already uses `[]Task` after Task 2
- `card := cards[0]` → `task := cards[0]`
- `r.processCard(ctx, card)` → `r.processCard(ctx, task)`

**Step 4: Update processCard(), failCard(), buildPrompt()**

Change `processCard(ctx context.Context, card trello.Card)` to `processCard(ctx context.Context, task Task)`.

Replace all `card.` references:
- `card.Name` → `task.Name`
- `card.ID` → `task.ID`
- `card.Desc` → `task.Description`

Replace Trello state calls:
- `r.trello.MoveCard(card.ID, r.failedListID)` + `r.trello.AddComment(...)` → handled in `failCard`/`MarkDone`

In `processCard`, the Done path currently does:
```go
r.trello.MoveCard(card.ID, r.doneListID)
r.trello.AddComment(card.ID, fmt.Sprintf("✅ ..."))
```
Replace with:
```go
comment := fmt.Sprintf("✅ Task completed by devpilot runner\nDuration: %s\nPR: %s", duration, prURL)
r.source.MarkDone(task.ID, comment)
```

In `processCard`, the InProgress call:
```go
r.trello.MoveCard(card.ID, r.inProgListID)
```
Replace with:
```go
r.source.MarkInProgress(task.ID)
```

In `processCard`, the PR body:
```go
// Remove: cardURL := fmt.Sprintf("https://trello.com/c/%s", card.ID)
prBody := fmt.Sprintf("## Task\n%s\n\n🤖 Executed by devpilot runner", task.URL)
```

Change `failCard(card trello.Card, ...)` to `failCard(task Task, ...)`:
```go
func (r *Runner) failCard(task Task, start time.Time, errMsg string) {
	duration := time.Since(start).Round(time.Second)
	r.emit(CardFailedEvent{CardID: task.ID, CardName: task.Name, ErrMsg: errMsg, Duration: duration})
	logPath := filepath.Join(r.config.WorkDir, ".devpilot", "logs", task.ID+".log")
	comment := fmt.Sprintf("❌ Task failed\nDuration: %s\nError: %s\nSee full log: %s", duration, errMsg, logPath)
	r.source.MarkFailed(task.ID, comment)
	r.logger.Printf("Card %q failed: %s", task.Name, errMsg)
}
```

Change `buildPrompt(card trello.Card)` to `buildPrompt(task Task)`:
```go
func (r *Runner) buildPrompt(task Task) string {
	return fmt.Sprintf(`Execute the following task plan autonomously...

Task: %s

Plan:
%s
...`, task.Name, task.Description)
}
```

Also update `saveLog(cardID string, ...)` calls to use `task.ID`, and `git.BranchName(task.ID, task.Name)`.

**Step 5: Verify it compiles**

```bash
go build ./internal/taskrunner/
```
Expected: compile error in `commands.go` — `New()` signature changed. That's OK, fix in Task 5.

```bash
go test ./internal/taskrunner/ -run TestSortByPriority -run TestTrelloSource -v
```
Expected: PASS

**Step 6: Commit**

```bash
git add internal/taskrunner/runner.go
git commit -m "refactor(taskrunner): Runner uses TaskSource interface instead of *trello.Client"
```

---

### Task 5: Update taskrunner/commands.go

**Files:**
- Modify: `internal/taskrunner/commands.go`

**Step 1: Add `--source` flag and wire up TaskSource**

Key changes:
1. Add `--source` flag (default "")
2. Read source from config if flag not set, fall back to "trello"
3. Create `TrelloSource` or `GitHubSource` based on value
4. Change `runWithTUI` and `runPlainText` signatures to accept `TaskSource`

Updated `RegisterCommands`:
```go
runCmd.Flags().String("source", "", "Task source: trello or github (default from .devpilot.json, fallback to trello)")
```

Updated handler logic (replacing the Trello-specific auth block):
```go
sourceName, _ := cmd.Flags().GetString("source")

cfg, _ := project.Load(dir)
if boardName == "" && cfg.Board != "" {
    boardName = cfg.Board
}
if sourceName == "" && cfg.Source != "" {
    sourceName = cfg.Source
}
if sourceName == "" {
    sourceName = "trello"
}

var source taskrunner.TaskSource
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
    trelloClient := trello.NewClient(creds["api_key"], creds["token"])
    source = taskrunner.NewTrelloSource(trelloClient, boardName)
case "github":
    source = taskrunner.NewGitHubSource()
default:
    fmt.Fprintf(os.Stderr, "Unknown source %q. Must be trello or github.\n", sourceName)
    os.Exit(1)
}
```

Change `runWithTUI` and `runPlainText` to take `(cfg Config, source TaskSource, boardName string)` and `(cfg Config, source TaskSource)` respectively. Remove `trelloClient *trello.Client` parameter and replace `New(cfg, trelloClient, ...)` with `New(cfg, source, ...)`.

Update the short/long description of runCmd to be source-agnostic:
```go
Short: "Autonomously process tasks from a board or issue tracker",
Long:  "Poll a task source (Trello or GitHub Issues) for ready tasks, execute their plans via Claude Code, and create PRs.",
```

**Step 2: Verify build**

```bash
go build ./...
```
Expected: PASS (GitHubSource not yet defined — we'll stub it next)

Actually `NewGitHubSource()` is referenced before Task 6, so add a temporary stub or do Task 6 first. Adjust: write the GitHubSource type stub (empty struct + constructor) at the top of the github_source.go file before wiring commands.go, then fill in the implementation.

**Step 3: Commit after Task 6 is done** (combine into one commit, see Task 6 step 5)

---

### Task 6: Create GitHubSource adapter

**Files:**
- Create: `internal/taskrunner/github_source.go`
- Create: `internal/taskrunner/github_source_test.go`

**Step 1: Write failing test (pure logic, no exec)**

The test focuses on the filtering and priority parsing logic, using a helper that takes raw issue data:

```go
// internal/taskrunner/github_source_test.go
package taskrunner

import "testing"

func TestGitHubSource_FilterReady(t *testing.T) {
	issues := []ghIssue{
		{Number: 1, Title: "Ready task", Body: "Do this", URL: "https://github.com/o/r/issues/1",
			Labels: []ghLabel{{Name: "devpilot"}}},
		{Number: 2, Title: "In progress", URL: "https://github.com/o/r/issues/2",
			Labels: []ghLabel{{Name: "devpilot"}, {Name: "in-progress"}}},
		{Number: 3, Title: "Failed task", URL: "https://github.com/o/r/issues/3",
			Labels: []ghLabel{{Name: "devpilot"}, {Name: "failed"}}},
	}

	tasks := issuesToReadyTasks(issues)

	if len(tasks) != 1 {
		t.Fatalf("expected 1 ready task, got %d", len(tasks))
	}
	if tasks[0].ID != "1" || tasks[0].Name != "Ready task" {
		t.Errorf("unexpected task: %+v", tasks[0])
	}
}

func TestGitHubPriority(t *testing.T) {
	cases := []struct {
		labels   []ghLabel
		expected int
	}{
		{[]ghLabel{{Name: "P0-critical"}, {Name: "devpilot"}}, 0},
		{[]ghLabel{{Name: "p1-high"}, {Name: "devpilot"}}, 1},
		{[]ghLabel{{Name: "devpilot"}}, 2},
	}
	for _, c := range cases {
		issue := ghIssue{Labels: c.labels}
		got := ghPriority(issue)
		if got != c.expected {
			t.Errorf("labels %v: expected %d, got %d", c.labels, c.expected, got)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/taskrunner/ -run TestGitHub -v
```
Expected: FAIL — `ghIssue`, `issuesToReadyTasks`, `ghPriority` not defined

**Step 3: Write implementation**

```go
// internal/taskrunner/github_source.go
package taskrunner

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitHubSource implements TaskSource using the gh CLI.
// Authentication is handled by gh (run 'gh auth login' separately).
type GitHubSource struct{}

func NewGitHubSource() *GitHubSource {
	return &GitHubSource{}
}

func (s *GitHubSource) Init() (SourceInfo, error) {
	if out, err := exec.Command("gh", "auth", "status").CombinedOutput(); err != nil {
		return SourceInfo{}, fmt.Errorf("not authenticated with GitHub CLI: run 'gh auth login'\n%s", string(out))
	}
	out, err := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner").Output()
	if err != nil {
		return SourceInfo{}, fmt.Errorf("detect repo from origin: %w (are you in a GitHub repo?)", err)
	}
	repo := strings.TrimSpace(string(out))
	return SourceInfo{DisplayName: repo}, nil
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghIssue struct {
	Number int       `json:"number"`
	Title  string    `json:"title"`
	Body   string    `json:"body"`
	URL    string    `json:"url"`
	Labels []ghLabel `json:"labels"`
}

func (s *GitHubSource) FetchReady() ([]Task, error) {
	out, err := exec.Command("gh", "issue", "list",
		"--label", "devpilot",
		"--state", "open",
		"--json", "number,title,body,url,labels",
		"--limit", "100",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("gh issue list: %w", err)
	}
	var issues []ghIssue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parse issues: %w", err)
	}
	return issuesToReadyTasks(issues), nil
}

// issuesToReadyTasks filters out in-progress and failed issues, maps the rest to Tasks.
func issuesToReadyTasks(issues []ghIssue) []Task {
	var tasks []Task
	for _, issue := range issues {
		if ghHasLabel(issue, "in-progress") || ghHasLabel(issue, "failed") {
			continue
		}
		tasks = append(tasks, Task{
			ID:          fmt.Sprintf("%d", issue.Number),
			Name:        issue.Title,
			Description: issue.Body,
			URL:         issue.URL,
			Priority:    ghPriority(issue),
		})
	}
	return tasks
}

func (s *GitHubSource) MarkInProgress(id string) error {
	_, err := exec.Command("gh", "issue", "edit", id, "--add-label", "in-progress").Output()
	return err
}

func (s *GitHubSource) MarkDone(id, comment string) error {
	if err := s.addComment(id, comment); err != nil {
		return err
	}
	_, err := exec.Command("gh", "issue", "close", id).Output()
	return err
}

func (s *GitHubSource) MarkFailed(id, comment string) error {
	_, err := exec.Command("gh", "issue", "edit", id,
		"--remove-label", "in-progress",
		"--add-label", "failed",
	).Output()
	if err != nil {
		return err
	}
	return s.addComment(id, comment)
}

func (s *GitHubSource) addComment(id, comment string) error {
	_, err := exec.Command("gh", "issue", "comment", id, "--body", comment).Output()
	return err
}

func ghHasLabel(issue ghIssue, name string) bool {
	for _, l := range issue.Labels {
		if l.Name == name {
			return true
		}
	}
	return false
}

func ghPriority(issue ghIssue) int {
	for _, l := range issue.Labels {
		name := strings.ToUpper(l.Name)
		if strings.HasPrefix(name, "P0") {
			return 0
		}
		if strings.HasPrefix(name, "P1") {
			return 1
		}
		if strings.HasPrefix(name, "P2") {
			return 2
		}
	}
	return 2
}
```

**Step 4: Run tests**

```bash
go test ./internal/taskrunner/ -run TestGitHub -v
go test ./internal/taskrunner/ -v
```
Expected: PASS all

**Step 5: Build everything and commit Tasks 5+6**

```bash
go build ./...
```
Expected: PASS

```bash
git add internal/taskrunner/github_source.go internal/taskrunner/github_source_test.go internal/taskrunner/commands.go
git commit -m "feat(taskrunner): add GitHubSource adapter and --source flag to devpilot run"
```

---

### Task 7: Add Source field to project config

**Files:**
- Modify: `internal/project/config.go`
- Modify: `internal/project/config_test.go`

**Step 1: Write failing test**

In `config_test.go`, add:
```go
func TestConfig_SourceField(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Board: "My Board", Source: "github"}
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Source != "github" {
		t.Errorf("expected source=github, got %q", got.Source)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/project/ -run TestConfig_SourceField -v
```
Expected: FAIL — `Config` has no `Source` field

**Step 3: Add Source field**

In `internal/project/config.go`, update the Config struct:
```go
type Config struct {
	Board  string            `json:"board,omitempty"`
	Source string            `json:"source,omitempty"` // "trello" or "github"
	Models map[string]string `json:"models,omitempty"`
}
```

**Step 4: Run tests**

```bash
go test ./internal/project/ -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/project/config.go internal/project/config_test.go
git commit -m "feat(project): add Source field to config for task backend selection"
```

---

### Task 8: Update push command to support GitHub Issues

**Files:**
- Modify: `internal/trello/commands.go`

**Step 1: Write failing test**

In `internal/trello/commands_test.go`, verify the `extractTitle` function still works (it's already tested). The new piece is the `--source` flag routing. This is a command-level change, best tested manually, but we can add a unit test for the GitHub issue creation helper.

Add a test for the source detection logic (if extracted to a helper).

Actually the GitHub issue creation is a `gh` CLI call — hard to unit test. We'll verify manually in step 4.

**Step 2: Add `--source` flag and GitHub branch to push command**

In `internal/trello/commands.go`:

1. Add `--source` flag to `pushCmd`:
```go
pushCmd.Flags().String("source", "", "Task source: trello or github (default from .devpilot.json, fallback to trello)")
```

2. In the handler, read source (same pattern as run command):
```go
sourceName, _ := cmd.Flags().GetString("source")
dir, _ := os.Getwd()
projectCfg, _ := project.Load(dir)
if sourceName == "" && projectCfg.Source != "" {
    sourceName = projectCfg.Source
}
if sourceName == "" {
    sourceName = "trello"
}
```

3. Add GitHub branch after the existing Trello logic:
```go
switch sourceName {
case "trello":
    // existing code (board resolution, card creation)
    // ...
case "github":
    out, err := exec.Command("gh", "issue", "create",
        "--title", title,
        "--body", string(content),
        "--label", "devpilot",
    ).Output()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error creating issue: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("Created issue: %s\n", title)
    fmt.Println(strings.TrimSpace(string(out)))
default:
    fmt.Fprintf(os.Stderr, "Unknown source %q\n", sourceName)
    os.Exit(1)
}
```

4. Add `"os/exec"` and `"strings"` to imports.

**Step 3: Build and verify**

```bash
go build ./...
go test ./...
```
Expected: PASS

**Step 4: Manual smoke test (optional, in a test repo)**

```bash
# In a GitHub repo:
devpilot push myplan.md --source github
# Should create an issue with devpilot label
```

**Step 5: Update runCmd description and commit**

Also update `cmd/devpilot/main.go` if the run command description references "Trello" — make it generic.

```bash
git add internal/trello/commands.go
git commit -m "feat(push): add --source flag to support creating GitHub Issues"
```

---

### Task 9: Update rejected ideas doc

**Files:**
- Modify: `docs/rejected/2026-03-01-github-issues-task-source.md`

Update status from `deferred` to `implemented`:

```yaml
---
status: implemented
idea: "GitHub Issues as Task Source"
date: 2026-03-01
implemented: 2026-03-04
---
```

```bash
git add docs/rejected/2026-03-01-github-issues-task-source.md
git commit -m "docs: mark github-issues-task-source as implemented"
```

---

## Final Verification

```bash
go test ./...
go build ./...
```

Expected: all tests pass, clean build.
