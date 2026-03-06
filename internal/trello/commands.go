package trello

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/siyuqian/devpilot/internal/project"
)

func RegisterCommands(parent *cobra.Command) {
	pushCmd.Flags().String("board", "", "Trello board name (required)")
	pushCmd.Flags().String("list", "Ready", "Target list name")
	pushCmd.Flags().String("source", "", "Task source: trello or github (default from .devpilot.json, fallback to trello)")
	parent.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push <plan-file>",
	Short: "Create a task from a plan file",
	Long:  "Read a plan markdown file and create a Trello card or GitHub Issue with the title from the first # heading and the full file contents as the description.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "WARNING: 'devpilot push' is deprecated. Use OpenSpec + 'devpilot sync' instead.")
		fmt.Fprintln(os.Stderr, "  See: https://github.com/Fission-AI/OpenSpec")
		fmt.Fprintln(os.Stderr, "")

		filePath := args[0]
		listName, _ := cmd.Flags().GetString("list")

		sourceName, _ := cmd.Flags().GetString("source")
		dir, _ := os.Getwd()
		projectCfg, _ := project.Load(dir)
		sourceName = projectCfg.ResolveSource(sourceName)

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

		switch sourceName {
		case "trello":
			boardName, _ := cmd.Flags().GetString("board")

			if boardName == "" {
				if projectCfg.Board != "" {
					boardName = projectCfg.Board
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
		case "github":
			out, err := exec.Command("gh", "issue", "create",
				"--title", title,
				"--body", string(content),
				"--label", "devpilot",
			).Output()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating issue: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Created issue: %s\n", title)
			fmt.Println(strings.TrimSpace(string(out)))
		default:
			fmt.Fprintf(os.Stderr, "Unknown source %q. Must be trello or github.\n", sourceName)
			os.Exit(1)
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
