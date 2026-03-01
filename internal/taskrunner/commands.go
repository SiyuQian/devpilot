package taskrunner

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/siyuqian/devpilot/internal/project"
	"github.com/siyuqian/devpilot/internal/trello"
)

func RegisterCommands(parent *cobra.Command) {
	runCmd.Flags().String("board", "", "Trello board name (required)")
	runCmd.Flags().Int("interval", 300, "Poll interval in seconds")
	runCmd.Flags().Int("timeout", 30, "Per-task timeout in minutes")
	runCmd.Flags().Int("review-timeout", 10, "Code review timeout in minutes (0 to disable)")
	runCmd.Flags().Bool("once", false, "Process one card and exit")
	runCmd.Flags().Bool("dry-run", false, "Print actions without executing")
	runCmd.Flags().Bool("no-tui", false, "Disable TUI, use plain text output")
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
		noTUI, _ := cmd.Flags().GetBool("no-tui")

		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to get working directory:", err)
			os.Exit(1)
		}

		if boardName == "" {
			cfg, _ := project.Load(dir)
			if cfg.Board != "" {
				boardName = cfg.Board
			}
		}
		if boardName == "" {
			fmt.Fprintln(os.Stderr, "Error: --board is required (or run: devpilot init)")
			os.Exit(1)
		}

		// Load Trello credentials
		creds, err := auth.Load("trello")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Not logged in to Trello. Run: devpilot login trello")
			os.Exit(1)
		}

		trelloClient := trello.NewClient(creds["api_key"], creds["token"])

		cfg := Config{
			BoardName:     boardName,
			Interval:      time.Duration(interval) * time.Second,
			Timeout:       time.Duration(timeout) * time.Minute,
			ReviewTimeout: time.Duration(reviewTimeout) * time.Minute,
			Once:          once,
			DryRun:        dryRun,
			WorkDir:       dir,
		}

		isInteractive := term.IsTerminal(int(os.Stdout.Fd()))

		if isInteractive && !noTUI {
			runWithTUI(cfg, trelloClient, boardName)
		} else {
			runPlainText(cfg, trelloClient)
		}
	},
}

func runWithTUI(cfg Config, trelloClient *trello.Client, boardName string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventCh := make(chan Event, 100)
	handler := func(e Event) {
		eventCh <- e
	}

	r := New(cfg, trelloClient, WithEventHandler(handler))
	model := NewTUIModel(boardName, eventCh, cancel)

	p := tea.NewProgram(model, tea.WithAltScreen())

	var runErr error
	go func() {
		if err := r.Run(ctx); err != nil {
			runErr = err
			eventCh <- RunnerErrorEvent{Err: err}
		}
		close(eventCh)
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "TUI error:", err)
		os.Exit(1)
	}

	if runErr != nil {
		fmt.Fprintln(os.Stderr, "Runner error:", runErr)
		os.Exit(1)
	}
}

func runPlainText(cfg Config, trelloClient *trello.Client) {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	handler := func(e Event) {
		switch ev := e.(type) {
		case RunnerStartedEvent:
			logger.Printf("Board: %s (%s)", ev.BoardName, ev.BoardID)
			for name, id := range ev.Lists {
				logger.Printf("  List %q → %s", name, id)
			}
		case PollingEvent:
			logger.Printf("Polling for tasks...")
		case NoTasksEvent:
			logger.Printf("No tasks. Next poll in %s", ev.NextPoll)
		case CardStartedEvent:
			logger.Printf("[card] Started: %q on branch %s", ev.CardName, ev.Branch)
		case ToolStartEvent:
			summary := toolSummary(ev.ToolName, ev.Input)
			logger.Printf("[tool] %s %s ...", ev.ToolName, summary)
		case ToolResultEvent:
			logger.Printf("[tool] %s done (%s)", ev.ToolName, formatDuration(ev.DurationMs))
		case TextOutputEvent:
			logger.Printf("[text] %s", truncate(ev.Text, 120))
		case StatsUpdateEvent:
			if ev.Turns > 0 {
				logger.Printf("[stats] ↑%s ↓%s turns:%d", formatTokens(ev.InputTokens), formatTokens(ev.OutputTokens), ev.Turns)
			}
		case CardDoneEvent:
			logger.Printf("[card] Done: %q (%s) PR: %s", ev.CardName, ev.Duration, ev.PRURL)
		case CardFailedEvent:
			logger.Printf("[card] Failed: %q — %s", ev.CardName, ev.ErrMsg)
		case ReviewStartedEvent:
			logger.Printf("[review] Starting code review for %s", ev.PRURL)
		case ReviewDoneEvent:
			logger.Printf("[review] Done (exit %d)", ev.ExitCode)
		case RunnerStoppedEvent:
			logger.Printf("Runner stopped.")
		case RunnerErrorEvent:
			if ev.Err != nil {
				logger.Printf("[error] %v", ev.Err)
			}
		}
	}

	r := New(cfg, trelloClient, WithEventHandler(handler))

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
}
