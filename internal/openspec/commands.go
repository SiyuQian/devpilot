package openspec

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/siyuqian/devpilot/internal/project"
	"github.com/siyuqian/devpilot/internal/trello"
)

func RegisterCommands(parent *cobra.Command) {
	syncCmd.Flags().String("board", "", "Trello board name (required for trello source)")
	syncCmd.Flags().String("source", "", "Task source: trello or github (default from .devpilot.json)")
	syncCmd.Flags().String("list", "Ready", "Target list name (trello only)")
	parent.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync OpenSpec changes to task board",
	Long:  "Scan openspec/changes/ and create or update cards/issues for each change proposal.",
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to get working directory:", err)
			os.Exit(1)
		}

		// Check OpenSpec is installed
		if err := CheckInstalled("openspec"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Scan changes
		changes, err := ScanChanges(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning changes: %v\n", err)
			os.Exit(1)
		}
		if len(changes) == 0 {
			fmt.Println("No changes found in openspec/changes/")
			return
		}

		// Resolve source
		projectCfg, _ := project.Load(dir)
		sourceName, _ := cmd.Flags().GetString("source")
		sourceName = projectCfg.ResolveSource(sourceName)

		boardName, _ := cmd.Flags().GetString("board")
		if boardName == "" && projectCfg.Board != "" {
			boardName = projectCfg.Board
		}

		listName, _ := cmd.Flags().GetString("list")

		// Build target
		var target SyncTarget
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
			client := trello.NewClient(creds["api_key"], creds["token"])
			board, err := client.FindBoardByName(boardName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			list, err := client.FindListByName(board.ID, listName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			target = NewTrelloTarget(client, list.ID)
		case "github":
			target = NewGitHubTarget()
		default:
			fmt.Fprintf(os.Stderr, "Unknown source %q\n", sourceName)
			os.Exit(1)
		}

		// Sync
		results, err := Sync(changes, target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Sync error: %v\n", err)
			os.Exit(1)
		}

		for _, r := range results {
			fmt.Printf("%s: %s\n", r.Action, r.Name)
		}
		fmt.Printf("\nSynced %d change(s) to %s\n", len(results), sourceName)
	},
}
