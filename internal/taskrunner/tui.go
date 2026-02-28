package taskrunner

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type toolCallEntry struct {
	toolName   string
	summary    string // e.g. "main.go" for Read, "go test ./..." for Bash
	durationMs int    // -1 while in progress
	timestamp  time.Time
}

type sessionStats struct {
	inputTokens     int
	outputTokens    int
	cacheReadTokens int
	turns           int
}

// TUIModel is the Bubble Tea model for the devkit run dashboard.
type TUIModel struct {
	// Config
	boardName string
	cancel    context.CancelFunc

	// State from runner events
	boardID    string
	lists      []listState
	phase      string // "starting", "polling", "running", "idle", "stopped"
	activeCard *cardState
	lastErr    string
	history    []cardState

	// Structured state (replaces logLines)
	toolCalls  []toolCallEntry
	activeCall *toolCallEntry
	textLines  []string
	stats      sessionStats

	// File tracking
	filesRead   []string
	filesEdited []string

	// Viewports (replaces single viewport)
	toolViewport viewport.Model
	textViewport viewport.Model

	// Panel focus
	focusedPane string // "tools" or "text"

	// Layout
	width  int
	height int
	ready  bool

	// Event channel
	eventCh <-chan Event
}

type listState struct {
	name string
	id   string
}

type cardState struct {
	id       string
	name     string
	branch   string
	status   string // "running", "done", "failed"
	prURL    string
	errMsg   string
	duration time.Duration
	started  time.Time
}

type runnerDoneMsg struct{}

type tickMsg time.Time

func tickEvery() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// NewTUIModel creates a new TUI model.
func NewTUIModel(boardName string, eventCh <-chan Event, cancel context.CancelFunc) TUIModel {
	return TUIModel{
		boardName:   boardName,
		cancel:      cancel,
		phase:       "starting",
		eventCh:     eventCh,
		focusedPane: "tools",
	}
}

func waitForEvent(ch <-chan Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return runnerDoneMsg{}
		}
		return event
	}
}

// Init implements tea.Model.
func (m TUIModel) Init() tea.Cmd {
	return tea.Batch(waitForEvent(m.eventCh), tickEvery())
}

// Update implements tea.Model.
func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 3
		footerHeight := 2
		availableHeight := m.height - headerHeight - footerHeight - 6
		if availableHeight < 2 {
			availableHeight = 2
		}
		toolVPHeight := availableHeight * 6 / 10
		textVPHeight := availableHeight - toolVPHeight
		if toolVPHeight < 1 {
			toolVPHeight = 1
		}
		if textVPHeight < 1 {
			textVPHeight = 1
		}

		if !m.ready {
			m.toolViewport = viewport.New(m.width, toolVPHeight)
			m.textViewport = viewport.New(m.width, textVPHeight)
			m.ready = true
		} else {
			m.toolViewport.Width = m.width
			m.toolViewport.Height = toolVPHeight
			m.textViewport.Width = m.width
			m.textViewport.Height = textVPHeight
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.cancel()
			return m, tea.Quit
		case tea.KeyTab:
			if m.focusedPane == "tools" {
				m.focusedPane = "text"
			} else {
				m.focusedPane = "tools"
			}
			return m, nil
		}
		switch msg.String() {
		case "q":
			m.cancel()
			return m, tea.Quit
		case "tab":
			if m.focusedPane == "tools" {
				m.focusedPane = "text"
			} else {
				m.focusedPane = "tools"
			}
			return m, nil
		case "g":
			if m.focusedPane == "tools" {
				m.toolViewport.GotoTop()
			} else {
				m.textViewport.GotoTop()
			}
			return m, nil
		case "G":
			if m.focusedPane == "tools" {
				m.toolViewport.GotoBottom()
			} else {
				m.textViewport.GotoBottom()
			}
			return m, nil
		}
		// Delegate scroll keys (j/k/up/down) to focused viewport
		var cmd tea.Cmd
		if m.focusedPane == "tools" {
			m.toolViewport, cmd = m.toolViewport.Update(msg)
		} else {
			m.textViewport, cmd = m.textViewport.Update(msg)
		}
		return m, cmd

	case tickMsg:
		// Re-render to update elapsed time display
		return m, tickEvery()

	case runnerDoneMsg:
		m.phase = "stopped"
		return m, nil

	case RunnerStartedEvent:
		m.boardID = msg.BoardID
		m.lists = make([]listState, 0, len(msg.Lists))
		for name, id := range msg.Lists {
			m.lists = append(m.lists, listState{name: name, id: id})
		}
		m.phase = "polling"
		return m, waitForEvent(m.eventCh)

	case PollingEvent:
		if m.phase != "running" {
			m.phase = "polling"
		}
		return m, waitForEvent(m.eventCh)

	case NoTasksEvent:
		m.phase = "idle"
		return m, waitForEvent(m.eventCh)

	case CardStartedEvent:
		m.activeCard = &cardState{
			id:      msg.CardID,
			name:    msg.CardName,
			branch:  msg.Branch,
			status:  "running",
			started: time.Now(),
		}
		// Clear new state
		m.toolCalls = nil
		m.activeCall = nil
		m.textLines = nil
		m.stats = sessionStats{}
		m.filesRead = nil
		m.filesEdited = nil
		m.phase = "running"
		return m, waitForEvent(m.eventCh)

	case ToolStartEvent:
		summary := toolSummary(msg.ToolName, msg.Input)
		m.activeCall = &toolCallEntry{
			toolName:   msg.ToolName,
			summary:    summary,
			durationMs: -1,
			timestamp:  time.Now(),
		}
		// Track files
		if fp := extractFilePath(msg.Input); fp != "" {
			switch msg.ToolName {
			case "Read", "Grep", "Glob":
				m.filesRead = addUnique(m.filesRead, fp)
			case "Edit", "Write":
				m.filesEdited = addUnique(m.filesEdited, fp)
			}
		}
		return m, waitForEvent(m.eventCh)

	case ToolResultEvent:
		if m.activeCall != nil {
			m.activeCall.durationMs = msg.DurationMs
			m.toolCalls = append(m.toolCalls, *m.activeCall)
			m.activeCall = nil
		}
		m.toolViewport.SetContent(renderToolCallsList(m))
		m.toolViewport.GotoBottom()
		return m, waitForEvent(m.eventCh)

	case TextOutputEvent:
		m.textLines = append(m.textLines, msg.Text)
		m.textViewport.SetContent(joinLines(m.textLines))
		m.textViewport.GotoBottom()
		return m, waitForEvent(m.eventCh)

	case StatsUpdateEvent:
		m.stats.inputTokens += msg.InputTokens
		m.stats.outputTokens += msg.OutputTokens
		m.stats.cacheReadTokens += msg.CacheReadTokens
		if msg.Turns > 0 {
			m.stats.turns = msg.Turns
		}
		return m, waitForEvent(m.eventCh)

	case CardDoneEvent:
		entry := cardState{
			id:       msg.CardID,
			name:     msg.CardName,
			status:   "done",
			prURL:    msg.PRURL,
			duration: msg.Duration,
		}
		m.history = append(m.history, entry)
		m.activeCard = nil
		m.phase = "polling"
		return m, waitForEvent(m.eventCh)

	case CardFailedEvent:
		entry := cardState{
			id:       msg.CardID,
			name:     msg.CardName,
			status:   "failed",
			errMsg:   msg.ErrMsg,
			duration: msg.Duration,
		}
		m.history = append(m.history, entry)
		m.activeCard = nil
		m.phase = "polling"
		return m, waitForEvent(m.eventCh)

	case ReviewStartedEvent:
		return m, waitForEvent(m.eventCh)

	case ReviewDoneEvent:
		return m, waitForEvent(m.eventCh)

	case RunnerStoppedEvent:
		m.phase = "stopped"
		return m, waitForEvent(m.eventCh)

	case RunnerErrorEvent:
		if msg.Err != nil {
			m.lastErr = msg.Err.Error()
		}
		return m, waitForEvent(m.eventCh)
	}

	return m, nil
}

// View implements tea.Model (stub -- rendering in tui_view.go).
func (m TUIModel) View() string {
	if !m.ready {
		return "  Starting devkit run..."
	}
	return m.renderView()
}

func joinLines(lines []string) string {
	var s string
	for i, l := range lines {
		if i > 0 {
			s += "\n"
		}
		s += l
	}
	return s
}

// Helper functions for model updates

func toolSummary(toolName string, input map[string]any) string {
	switch toolName {
	case "Read":
		if fp, ok := input["file_path"].(string); ok {
			return shortenPath(fp)
		}
	case "Edit", "Write":
		if fp, ok := input["file_path"].(string); ok {
			return shortenPath(fp)
		}
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			if len(cmd) > 60 {
				return cmd[:60] + "..."
			}
			return cmd
		}
	case "Grep":
		if pat, ok := input["pattern"].(string); ok {
			return pat
		}
	case "Glob":
		if pat, ok := input["pattern"].(string); ok {
			return pat
		}
	}
	return ""
}

func extractFilePath(input map[string]any) string {
	if fp, ok := input["file_path"].(string); ok {
		return fp
	}
	return ""
}

func addUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func shortenPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}
	return strings.Join(parts[len(parts)-2:], "/")
}
