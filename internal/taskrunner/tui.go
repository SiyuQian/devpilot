package taskrunner

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

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

	// Log viewport
	logLines []string
	viewport viewport.Model

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
		boardName: boardName,
		cancel:    cancel,
		phase:     "starting",
		eventCh:   eventCh,
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
		vpHeight := m.height - headerHeight - footerHeight
		if vpHeight < 1 {
			vpHeight = 1
		}
		if !m.ready {
			m.viewport = viewport.New(m.width, vpHeight)
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpHeight
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.cancel()
			return m, tea.Quit
		}
		switch msg.String() {
		case "q":
			m.cancel()
			return m, tea.Quit
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		}
		// Delegate scroll keys (j/k/up/down) to viewport
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
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
		m.logLines = nil // clear logs for new card
		m.phase = "running"
		return m, waitForEvent(m.eventCh)

	case CardOutputEvent:
		line := fmt.Sprintf("[%s] %s", msg.Line.Stream, msg.Line.Text)
		m.logLines = append(m.logLines, line)
		m.viewport.SetContent(joinLines(m.logLines))
		m.viewport.GotoBottom()
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

// View implements tea.Model (stub — rendering in tui_view.go).
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
