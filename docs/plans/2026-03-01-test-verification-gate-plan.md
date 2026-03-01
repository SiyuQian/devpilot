# Test Verification Gate Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a mandatory test verification step in the task runner between Claude execution and PR creation. If tests fail, the card moves to Failed with test output attached — preventing broken code from becoming PRs.

**Architecture:** Add a `TestRunner` that executes the project's test command (from `.devkit.json`) after Claude finishes but before `git push`. If tests fail, attach the output to the card and fail it. If no test command is configured, skip the gate with a warning.

**Tech Stack:** Go, `os/exec`

---

### Task 1: Add test_command to project config

**Files:**
- Edit: `internal/project/config.go`
- Edit: `internal/project/config_test.go`

**Step 1: Write failing test for TestCommand field**

Add a test that loads a `.devkit.json` with a `test_command` field and asserts it's parsed correctly:

```json
{"board": "my-board", "test_command": "go test ./..."}
```

**Step 2: Add TestCommand field to Config struct**

Add `TestCommand string` field to the project config struct with JSON tag `test_command`. Update `LoadConfig` if needed.

**Step 3: Verify tests pass**

---

### Task 2: Implement test runner

**Files:**
- Create: `internal/taskrunner/testgate.go`
- Create: `internal/taskrunner/testgate_test.go`

**Step 1: Write failing tests for RunTests**

Test cases:
- `TestRunTests_Success`: Command exits 0 → returns nil error, captures stdout
- `TestRunTests_Failure`: Command exits non-zero → returns error with stdout/stderr
- `TestRunTests_Timeout`: Command exceeds timeout → returns timeout error
- `TestRunTests_EmptyCommand`: No test command configured → returns nil (skip)

Use a mock command pattern (test helper binary or simple shell commands like `true`/`false`).

**Step 2: Implement RunTests function**

```go
type TestGate struct {
    Command string        // e.g. "go test ./..."
    Timeout time.Duration // default 5 minutes
    Dir     string        // working directory
}

type TestResult struct {
    Passed bool
    Output string // combined stdout+stderr
    Duration time.Duration
}

func (tg *TestGate) Run(ctx context.Context) (*TestResult, error)
```

Execute the test command via `exec.CommandContext`. Capture combined output. Return structured result.

**Step 3: Verify all tests pass**

---

### Task 3: Integrate test gate into runner

**Files:**
- Edit: `internal/taskrunner/runner.go`
- Edit: `internal/taskrunner/events.go`

**Step 1: Add TestGateEvent types**

Add two new events:
- `TestGateStartedEvent` — emitted when tests begin
- `TestGateResultEvent` — emitted with pass/fail and output summary

**Step 2: Insert test gate into processCard**

After the "check for new commits" step (line ~244) and before `git push` (line ~251):

1. Load test command from project config
2. If test command is empty, log a warning and skip
3. Create `TestGate` with the command and card timeout
4. Emit `TestGateStartedEvent`
5. Run tests
6. Emit `TestGateResultEvent`
7. If tests failed: call `failCard()` with test output, return early
8. If tests passed: continue to push & PR

**Step 3: Verify with integration test**

Write a test for `processCard` that verifies:
- With passing test command: card proceeds to Done
- With failing test command: card moves to Failed with test output in comment

---

### Task 4: Update TUI to show test gate status

**Files:**
- Edit: `internal/taskrunner/tui.go` (or relevant TUI file)

**Step 1: Handle TestGateStartedEvent and TestGateResultEvent**

Add cases in the TUI event handler to display:
- "Running tests..." when started
- "Tests passed" or "Tests FAILED" with summary when complete

**Step 2: Verify TUI compiles and events render**

---

### Task 5: Update init wizard to prompt for test command

**Files:**
- Edit: `internal/project/init.go` (or relevant init file)

**Step 1: Add test command prompt to init wizard**

After detecting project type, suggest a default test command:
- Go project: `go test ./...`
- Node project: `npm test`
- Python project: `pytest`

Ask the user to confirm or customize. Write to `.devkit.json`.

**Step 2: Verify init wizard works end-to-end**
