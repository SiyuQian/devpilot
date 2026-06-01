package generate

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommitModelStateTransitions(t *testing.T) {
	model := newCommitModel(context.Background(), "model", "ctx", false)
	if model.phase != phaseStagingFiles {
		t.Fatalf("phase = %q, want %q", model.phase, phaseStagingFiles)
	}
	if model.model != "model" || model.context != "ctx" {
		t.Fatalf("model context not initialized: %#v", model)
	}
	if model.Init() == nil {
		t.Fatalf("Init() returned nil command")
	}

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 120})
	model = updated.(commitModel)
	if cmd != nil || model.width != 120 {
		t.Fatalf("window update: width=%d cmd=%v", model.width, cmd)
	}

	updated, _ = model.handleStagingDone(stagingDoneMsg{
		nameStatus:  "M\ta.go",
		diffStat:    "a.go | 1 +",
		diffContent: "diff --git a/a.go b/a.go",
		stagedFiles: []string{"a.go"},
	})
	model = updated.(commitModel)
	if model.phase != phaseAnalyzing {
		t.Fatalf("phase = %q, want analyzing", model.phase)
	}

	plan := commitPlan{Commits: []commitEntry{{Message: "feat: x", Files: []string{"a.go"}}}}
	updated, cmd = model.handleAnalysisDone(analysisDoneMsg{plan: plan, warnings: []string{"warn"}})
	model = updated.(commitModel)
	if cmd != nil || model.phase != phasePlan || len(model.warnings) != 1 {
		t.Fatalf("analysis transition failed: phase=%q warnings=%v cmd=%v", model.phase, model.warnings, cmd)
	}

	updated, cmd = model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	model = updated.(commitModel)
	if cmd == nil || model.phase != phaseExecuting || model.currentCommit != 0 {
		t.Fatalf("start execution failed: phase=%q current=%d cmd=%v", model.phase, model.currentCommit, cmd)
	}

	updated, cmd = model.handleCommitExec(commitExecMsg{index: 0, hash: "abc123"})
	model = updated.(commitModel)
	if cmd == nil || model.phase != phaseDone || len(model.completedCommits) != 1 {
		t.Fatalf("commit exec transition failed: phase=%q completed=%d cmd=%v", model.phase, len(model.completedCommits), cmd)
	}
}

func TestCommitModelAsyncGitCommands(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "a.go"), []byte("package p\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	model := newCommitModel(context.Background(), "", "", false)
	msg := model.stageAndCollectDiff()()
	staging, ok := msg.(stagingDoneMsg)
	if !ok {
		t.Fatalf("stageAndCollectDiff() = %T", msg)
	}
	if staging.err != nil {
		t.Fatalf("stageAndCollectDiff() error = %v", staging.err)
	}
	if !contains(staging.stagedFiles, "a.go") || !strings.Contains(staging.nameStatus, "a.go") {
		t.Fatalf("staging msg = %#v", staging)
	}

	model.plan = commitPlan{
		Commits:  []commitEntry{{Message: "feat: add a", Files: []string{"a.go"}}},
		Excluded: []excludedFile{{File: "scratch.txt", Reason: "ignore"}},
	}
	if msg := model.unstageExcluded()(); msg != nil {
		t.Fatalf("unstageExcluded() = %#v, want nil", msg)
	}

	msg = model.executeCommit(0)()
	execMsg, ok := msg.(commitExecMsg)
	if !ok {
		t.Fatalf("executeCommit() = %T", msg)
	}
	if execMsg.err != nil {
		t.Fatalf("executeCommit() error = %v", execMsg.err)
	}
	if execMsg.hash == "" {
		t.Fatalf("hash is empty")
	}
}

func TestCommitModelAnalyzeChangesWithFakeClaude(t *testing.T) {
	bin := t.TempDir()
	claude := filepath.Join(bin, "claude")
	script := "#!/bin/sh\nprintf '%s\\n' '{\"commits\":[{\"message\":\"feat: x\",\"files\":[\"a.go\"]}]}'\n"
	if err := os.WriteFile(claude, []byte(script), 0o755); err != nil {
		t.Fatalf("write claude: %v", err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	model := commitModel{
		genCtx:      context.Background(),
		model:       "test-model",
		nameStatus:  "M\ta.go",
		diffStat:    "a.go | 1 +",
		diffContent: "diff --git a/a.go b/a.go\n",
		stagedFiles: []string{"a.go"},
	}
	msg := model.analyzeChanges()()
	analysis, ok := msg.(analysisDoneMsg)
	if !ok {
		t.Fatalf("analyzeChanges() = %T", msg)
	}
	if analysis.err != nil {
		t.Fatalf("analyzeChanges() error = %v", analysis.err)
	}
	if len(analysis.plan.Commits) != 1 || analysis.plan.Commits[0].Message != "feat: x" {
		t.Fatalf("plan = %#v", analysis.plan)
	}
}

func TestRunClaudeFailure(t *testing.T) {
	bin := t.TempDir()
	claude := filepath.Join(bin, "claude")
	if err := os.WriteFile(claude, []byte("#!/bin/sh\necho nope >&2\nexit 2\n"), 0o755); err != nil {
		t.Fatalf("write claude: %v", err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if _, err := run(context.Background(), "prompt", ""); err == nil {
		t.Fatalf("run() succeeded, want error")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestCommitModelErrorTransitions(t *testing.T) {
	model := newCommitModel(context.Background(), "", "", false)

	updated, cmd := model.finishWithError(errors.New("bad"))
	model = updated.(commitModel)
	if cmd == nil || model.phase != phaseDone || model.err == nil {
		t.Fatalf("finishWithError failed: phase=%q err=%v cmd=%v", model.phase, model.err, cmd)
	}

	model = newCommitModel(context.Background(), "", "", false)
	updated, cmd = model.handleStagingDone(stagingDoneMsg{err: errors.New("stage")})
	model = updated.(commitModel)
	if cmd == nil || model.phase != phaseDone || model.err == nil {
		t.Fatalf("staging error transition failed: phase=%q err=%v cmd=%v", model.phase, model.err, cmd)
	}

	model = newCommitModel(context.Background(), "", "", false)
	updated, cmd = model.handleStagingDone(stagingDoneMsg{})
	model = updated.(commitModel)
	if cmd == nil || !errors.Is(model.err, errNoChanges) {
		t.Fatalf("no changes transition failed: err=%v cmd=%v", model.err, cmd)
	}

	model = commitModel{phase: phasePlan}
	updated, cmd = model.handleAnalysisDone(analysisDoneMsg{err: errors.New("analyze")})
	model = updated.(commitModel)
	if cmd == nil || model.phase != phaseDone || model.err == nil {
		t.Fatalf("analysis error transition failed: phase=%q err=%v cmd=%v", model.phase, model.err, cmd)
	}

	model = commitModel{phase: phasePlan, plan: commitPlan{Commits: []commitEntry{{Message: "feat: x"}}}}
	updated, cmd = model.handleEditDone(editDoneMsg{err: errors.New("edit")})
	model = updated.(commitModel)
	if cmd != nil || len(model.warnings) != 1 {
		t.Fatalf("edit error should warn without command: warnings=%v cmd=%v", model.warnings, cmd)
	}

	updated, cmd = model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	model = updated.(commitModel)
	if cmd == nil || !errors.Is(model.err, errAborted) {
		t.Fatalf("abort key failed: err=%v cmd=%v", model.err, cmd)
	}
}

func TestCommitModelAdditionalTransitions(t *testing.T) {
	plan := commitPlan{
		Commits: []commitEntry{
			{Message: "feat: one", Files: []string{"a.go"}},
			{Message: "feat: two", Files: []string{"b.go"}},
		},
		Excluded: []excludedFile{{File: "scratch.txt", Reason: "ignore"}},
	}
	model := commitModel{phase: phasePlan, plan: plan}

	updated, cmd := model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	model = updated.(commitModel)
	if cmd == nil || model.phase != phaseExecuting {
		t.Fatalf("multi-commit accept failed: phase=%q cmd=%v", model.phase, cmd)
	}

	updated, cmd = model.handleCommitExec(commitExecMsg{index: 0, hash: "abc123"})
	model = updated.(commitModel)
	if cmd == nil || model.phase != phaseExecuting || model.currentCommit != 1 || len(model.completedCommits) != 1 {
		t.Fatalf("next commit transition failed: phase=%q current=%d completed=%d cmd=%v", model.phase, model.currentCommit, len(model.completedCommits), cmd)
	}

	updated, cmd = model.handleCommitExec(commitExecMsg{index: 1, err: errors.New("commit failed")})
	model = updated.(commitModel)
	if cmd == nil || model.phase != phaseDone || model.err == nil || len(model.completedCommits) != 2 {
		t.Fatalf("commit error transition failed: phase=%q err=%v completed=%d cmd=%v", model.phase, model.err, len(model.completedCommits), cmd)
	}

	model = commitModel{phase: phasePlan}
	updated, cmd = model.handleEditDone(editDoneMsg{plan: commitPlan{Commits: []commitEntry{{Message: "fix: edited"}}}})
	model = updated.(commitModel)
	if cmd != nil || len(model.warnings) != 0 || model.plan.Commits[0].Message != "fix: edited" {
		t.Fatalf("edit success transition failed: plan=%#v warnings=%v cmd=%v", model.plan, model.warnings, cmd)
	}

	model = commitModel{phase: phaseAnalyzing}
	updated, cmd = model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd != nil {
		t.Fatalf("non-plan key produced cmd: %v", cmd)
	}
	if updated.(commitModel).phase != phaseAnalyzing {
		t.Fatalf("phase changed outside plan: %q", updated.(commitModel).phase)
	}

	updated, cmd = model.Update(struct{}{})
	if cmd != nil || updated.(commitModel).phase != phaseAnalyzing {
		t.Fatalf("unknown update changed state: %#v cmd=%v", updated, cmd)
	}
}
