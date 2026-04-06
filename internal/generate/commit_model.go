package generate

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	errAborted   = errors.New("aborted")
	errNoChanges = errors.New("no changes to commit")
)

// Phases for the commit TUI.
const (
	phaseStagingFiles = "staging"
	phaseAnalyzing    = "analyzing"
	phasePlan         = "plan"
	phaseExecuting    = "executing"
	phaseDone         = "done"
)

// CommitModel is the Bubble Tea model for the commit workflow.
type CommitModel struct {
	// Config
	model   string
	context string
	dryRun  bool
	genCtx  context.Context

	// State
	phase    string
	plan     CommitPlan
	warnings []string
	err      error

	// Git data collected during staging
	nameStatus  string
	diffStat    string
	diffContent string
	stagedFiles []string

	// Execution progress
	completedCommits []commitResult
	currentCommit    int

	// UI
	spinner spinner.Model
	width   int
}

type commitResult struct {
	hash    string
	message string
	err     error
}

// Message types for async operations.
type stagingDoneMsg struct {
	nameStatus  string
	diffStat    string
	diffContent string
	stagedFiles []string
	err         error
}

type analysisDoneMsg struct {
	plan     CommitPlan
	warnings []string
	err      error
}

type commitExecMsg struct {
	index int
	hash  string
	err   error
}

type editDoneMsg struct {
	plan CommitPlan
	err  error
}

func (m CommitModel) finishWithError(err error) (tea.Model, tea.Cmd) {
	m.phase = phaseDone
	m.err = err
	return m, tea.Quit
}

func (m CommitModel) isMultiCommit() bool {
	return len(m.plan.Commits) > 1 || len(m.plan.Excluded) > 0
}

// NewCommitModel creates a new CommitModel.
func NewCommitModel(ctx context.Context, model, userContext string, dryRun bool) CommitModel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return CommitModel{
		model:   model,
		context: userContext,
		dryRun:  dryRun,
		genCtx:  ctx,
		phase:   phaseStagingFiles,
		spinner: s,
	}
}

func (m CommitModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.stageAndCollectDiff())
}

func (m CommitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case stagingDoneMsg:
		return m.handleStagingDone(msg)
	case analysisDoneMsg:
		return m.handleAnalysisDone(msg)
	case commitExecMsg:
		return m.handleCommitExec(msg)
	case editDoneMsg:
		return m.handleEditDone(msg)
	}
	return m, nil
}

func (m CommitModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	}

	if m.phase != phasePlan {
		return m, nil
	}

	switch msg.String() {
	case "a":
		if m.isMultiCommit() {
			return m.startExecution()
		}
	case "y":
		if !m.isMultiCommit() {
			return m.startExecution()
		}
	case "e":
		return m, m.openEditor()
	case "n":
		return m.finishWithError(errAborted)
	}
	return m, nil
}

func (m CommitModel) handleStagingDone(msg stagingDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m.finishWithError(msg.err)
	}

	if len(msg.stagedFiles) == 0 {
		return m.finishWithError(errNoChanges)
	}

	m.nameStatus = msg.nameStatus
	m.diffStat = msg.diffStat
	m.diffContent = msg.diffContent
	m.stagedFiles = msg.stagedFiles
	m.phase = phaseAnalyzing

	return m, m.analyzeChanges()
}

func (m CommitModel) handleAnalysisDone(msg analysisDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m.finishWithError(msg.err)
	}

	m.plan = msg.plan
	m.warnings = msg.warnings
	m.phase = phasePlan

	if m.dryRun {
		m.phase = phaseDone
		return m, tea.Quit
	}

	return m, nil
}

func (m CommitModel) handleCommitExec(msg commitExecMsg) (tea.Model, tea.Cmd) {
	result := commitResult{hash: msg.hash, message: m.plan.Commits[msg.index].Message}
	if msg.err != nil {
		result.err = msg.err
		m.completedCommits = append(m.completedCommits, result)
		return m.finishWithError(msg.err)
	}

	m.completedCommits = append(m.completedCommits, result)
	m.currentCommit = msg.index + 1

	if m.currentCommit >= len(m.plan.Commits) {
		m.phase = phaseDone
		return m, tea.Quit
	}

	return m, m.executeCommit(m.currentCommit)
}

func (m CommitModel) handleEditDone(msg editDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		// Stay in plan phase, show error
		m.warnings = append(m.warnings, fmt.Sprintf("edit error: %v", msg.err))
		return m, nil
	}
	m.plan = msg.plan
	m.warnings = nil
	return m, nil
}

func (m CommitModel) startExecution() (tea.Model, tea.Cmd) {
	m.phase = phaseExecuting
	m.currentCommit = 0
	m.completedCommits = nil

	if len(m.plan.Excluded) > 0 {
		// Unstage excluded files first, then execute the first commit
		return m, tea.Sequence(m.unstageExcluded(), m.executeCommit(0))
	}
	return m, m.executeCommit(0)
}

// Async commands

func (m CommitModel) stageAndCollectDiff() tea.Cmd {
	return func() tea.Msg {
		// Stage all changes
		if out, err := exec.Command("git", "add", ".").CombinedOutput(); err != nil {
			return stagingDoneMsg{err: fmt.Errorf("git add: %w\n%s", err, out)}
		}

		diffStat, err := gitOutput("diff", "--cached", "--stat")
		if err != nil {
			return stagingDoneMsg{err: err}
		}
		if diffStat == "" {
			return stagingDoneMsg{} // no changes
		}

		nameStatus, err := gitOutput("diff", "--cached", "--name-status")
		if err != nil {
			return stagingDoneMsg{err: err}
		}

		diffContent, err := gitOutput("diff", "--cached")
		if err != nil {
			return stagingDoneMsg{err: err}
		}

		nameOnly, err := gitOutput("diff", "--cached", "--name-only")
		if err != nil {
			return stagingDoneMsg{err: err}
		}
		stagedFiles := strings.Split(nameOnly, "\n")

		return stagingDoneMsg{
			nameStatus:  nameStatus,
			diffStat:    diffStat,
			diffContent: diffContent,
			stagedFiles: stagedFiles,
		}
	}
}

func (m CommitModel) analyzeChanges() tea.Cmd {
	return func() tea.Msg {
		prompt, err := buildCommitPrompt(m.nameStatus, m.diffStat, m.diffContent, m.context)
		if err != nil {
			return analysisDoneMsg{err: fmt.Errorf("build prompt: %w", err)}
		}

		output, err := Generate(m.genCtx, prompt, m.model)
		if err != nil {
			return analysisDoneMsg{err: err}
		}

		plan := parseCommitPlan(output, m.stagedFiles)
		plan, warnings := validatePlan(plan, m.stagedFiles)

		return analysisDoneMsg{plan: plan, warnings: warnings}
	}
}

func (m CommitModel) executeCommit(index int) tea.Cmd {
	commit := m.plan.Commits[index]
	return func() tea.Msg {
		// Reset staging area
		if out, err := exec.Command("git", "reset", "HEAD", "--", ".").CombinedOutput(); err != nil {
			return commitExecMsg{index: index, err: fmt.Errorf("git reset: %w\n%s", err, out)}
		}

		// Stage only this commit's files
		addArgs := append([]string{"add", "--"}, commit.Files...)
		if out, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
			return commitExecMsg{index: index, err: fmt.Errorf("git add: %w\n%s", err, out)}
		}

		// Commit
		if out, err := exec.Command("git", "commit", "-m", commit.Message).CombinedOutput(); err != nil {
			return commitExecMsg{index: index, err: fmt.Errorf("git commit: %w\n%s", err, out)}
		}

		hash, err := gitOutput("rev-parse", "--short", "HEAD")
		if err != nil {
			hash = "???????"
		}

		return commitExecMsg{index: index, hash: hash}
	}
}

func (m CommitModel) unstageExcluded() tea.Cmd {
	excluded := m.plan.Excluded
	return func() tea.Msg {
		var files []string
		for _, e := range excluded {
			files = append(files, e.File)
		}
		args := append([]string{"reset", "HEAD", "--"}, files...)
		_ = exec.Command("git", args...).Run() // best-effort
		return nil
	}
}

func (m CommitModel) openEditor() tea.Cmd {
	return func() tea.Msg {
		edited, err := editPlanInTerminal(m.plan)
		if err != nil {
			return editDoneMsg{err: err}
		}
		return editDoneMsg{plan: edited}
	}
}
