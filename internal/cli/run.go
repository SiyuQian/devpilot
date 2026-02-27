package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/config"
	"github.com/siyuqian/developer-kit/internal/runner"
	"github.com/siyuqian/developer-kit/internal/trello"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Autonomously process tasks from a Trello board",
	Long:  "Poll a Trello board for Ready cards, execute their plans via Claude Code, and create PRs.",
	Run: func(cmd *cobra.Command, args []string) {
		boardName, _ := cmd.Flags().GetString("board")
		interval, _ := cmd.Flags().GetInt("interval")
		timeout, _ := cmd.Flags().GetInt("timeout")
		once, _ := cmd.Flags().GetBool("once")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if boardName == "" {
			fmt.Fprintln(os.Stderr, "Error: --board is required")
			os.Exit(1)
		}

		// Load Trello credentials
		creds, err := config.Load("trello")
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

		cfg := runner.Config{
			BoardName: boardName,
			Interval:  time.Duration(interval) * time.Second,
			Timeout:   time.Duration(timeout) * time.Minute,
			Once:      once,
			DryRun:    dryRun,
			WorkDir:   dir,
		}

		r := runner.New(cfg, trelloClient)

		// Handle Ctrl+C
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
	},
}

func init() {
	runCmd.Flags().String("board", "", "Trello board name (required)")
	runCmd.Flags().Int("interval", 300, "Poll interval in seconds")
	runCmd.Flags().Int("timeout", 30, "Per-task timeout in minutes")
	runCmd.Flags().Bool("once", false, "Process one card and exit")
	runCmd.Flags().Bool("dry-run", false, "Print actions without executing")
	rootCmd.AddCommand(runCmd)
}
