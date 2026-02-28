package trello

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/auth"
	"github.com/siyuqian/developer-kit/internal/project"
)

func RegisterCommands(parent *cobra.Command) {
	pushCmd.Flags().String("board", "", "Trello board name (required)")
	pushCmd.Flags().String("list", "Ready", "Target list name")
	parent.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push <plan-file>",
	Short: "Create a Trello card from a plan file",
	Long:  "Read a plan markdown file and create a Trello card with the title from the first # heading and the full file contents as the description.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		boardName, _ := cmd.Flags().GetString("board")
		listName, _ := cmd.Flags().GetString("list")

		if boardName == "" {
			dir, _ := os.Getwd()
			cfg, _ := project.Load(dir)
			if cfg.Board != "" {
				boardName = cfg.Board
			}
		}
		if boardName == "" {
			fmt.Fprintln(os.Stderr, "Error: --board is required (or run: devkit init)")
			os.Exit(1)
		}

		// Read the plan file
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		// Extract title from first # heading
		title := extractTitle(string(content))
		if title == "" {
			fmt.Fprintln(os.Stderr, "Error: no # heading found in file")
			os.Exit(1)
		}

		// Load Trello credentials
		creds, err := auth.Load("trello")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Not logged in to Trello. Run: devkit login trello")
			os.Exit(1)
		}

		client := NewClient(creds["api_key"], creds["token"])

		// Resolve board
		board, err := client.FindBoardByName(boardName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Resolve list
		list, err := client.FindListByName(board.ID, listName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Create card
		card, err := client.CreateCard(list.ID, title, string(content))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating card: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created card: %s\n", title)
		if card.ShortURL != "" {
			fmt.Println(card.ShortURL)
		}
	},
}

func extractTitle(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if title, ok := strings.CutPrefix(line, "# "); ok {
			return title
		}
	}
	return ""
}
