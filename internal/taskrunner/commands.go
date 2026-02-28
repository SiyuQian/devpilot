package taskrunner

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/siyuqian/developer-kit/internal/auth"
	"github.com/siyuqian/developer-kit/internal/trello"
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

		if boardName == "" {
			fmt.Fprintln(os.Stderr, "Error: --board is required")
			os.Exit(1)
		}

		// Load Trello credentials
		creds, err := auth.Load("trello")
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

	go func() {
		if err := r.Run(ctx); err != nil {
			eventCh <- RunnerErrorEvent{Err: err}
		}
		close(eventCh)
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "TUI error:", err)
		os.Exit(1)
	}
}

func runPlainText(cfg Config, trelloClient *trello.Client) {
	r := New(cfg, trelloClient)

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
