package trello

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/siyuqian/devpilot/internal/project"
	"github.com/spf13/cobra"
)

// RegisterCommands registers the trello subcommands on the given parent.
func RegisterCommands(parent *cobra.Command) {
	pushCmd.Flags().String("board", "", "Trello board name (required)")
	pushCmd.Flags().String("list", "Ready", "Target list name")
	pushCmd.Flags().String("source", "", "Task source: trello or github (default from .devpilot.yaml, fallback to trello)")
	parent.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push <plan-file>",
	Short: "Create a task from a plan file",
	Long:  "Read a plan markdown file and create a Trello card or GitHub Issue with the title from the first # heading and the full file contents as the description.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(runPush(cmd, args[0]))
	},
}

func runPush(cmd *cobra.Command, filePath string) int {
	fmt.Fprintln(os.Stderr, "WARNING: 'devpilot push' is deprecated. Use OpenSpec + 'devpilot sync' instead.")
	fmt.Fprintln(os.Stderr, "  See: https://github.com/Fission-AI/OpenSpec")
	fmt.Fprintln(os.Stderr, "")

	listName, _ := cmd.Flags().GetString("list")     // flag registered above
	sourceName, _ := cmd.Flags().GetString("source") // flag registered above
	dir, _ := os.Getwd()                             // error handled by downstream calls
	projectCfg, _ := project.Load(dir)               // project config is optional
	sourceName = projectCfg.ResolveSource(sourceName)

	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		return 1
	}

	title := extractTitle(string(content))
	if title == "" {
		fmt.Fprintln(os.Stderr, "Error: no # heading found in file")
		return 1
	}

	switch sourceName {
	case "trello":
		boardName, _ := cmd.Flags().GetString("board") // flag registered above

		if boardName == "" {
			if projectCfg.Board != "" {
				boardName = projectCfg.Board
			}
		}
		if boardName == "" {
			fmt.Fprintln(os.Stderr, "Error: --board is required (or run: devpilot init)")
			return 1
		}

		creds, err := auth.Load("trello")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Not logged in to Trello. Run: devpilot login trello")
			return 1
		}

		client := NewClient(creds["api_key"], creds["token"])
		board, err := client.FindBoardByName(boardName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}

		list, err := client.FindListByName(board.ID, listName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}

		card, err := client.CreateCard(list.ID, title, string(content))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating card: %v\n", err)
			return 1
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
			return 1
		}
		fmt.Printf("Created issue: %s\n", title)
		fmt.Println(strings.TrimSpace(string(out)))
	default:
		fmt.Fprintf(os.Stderr, "Unknown source %q. Must be trello or github.\n", sourceName)
		return 1
	}
	return 0
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
