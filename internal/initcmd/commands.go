package initcmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/siyuqian/devpilot/internal/trello"
	"github.com/spf13/cobra"
)

// RegisterCommands adds the init command to the parent command.
func RegisterCommands(parent *cobra.Command) {
	initCmd.Flags().BoolP("yes", "y", false, "Accept all defaults without prompting")
	parent.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a project for use with devpilot",
	Long:  "Detect existing project configuration, report current state, and generate missing pieces.",
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to get working directory:", err)
			os.Exit(1)
		}
		yes, _ := cmd.Flags().GetBool("yes")
		os.Exit(runInit(dir, yes, bufio.NewReader(os.Stdin)))
	},
}

func runInit(dir string, yes bool, reader *bufio.Reader) int {
	status := Detect(dir)

	fmt.Println("Scanning project...")
	for _, line := range formatStatus(status) {
		fmt.Println(line)
	}

	if allConfigured(status) {
		fmt.Println("\nProject already initialized!")
		return 0
	}

	opts := GenerateOpts{
		Dir:         dir,
		Interactive: !yes,
	}
	if opts.Interactive {
		opts.Reader = reader
	}

	fmt.Println()

	if status.Source == "github" {
		// Already configured as GitHub Issues — labels may already exist; skip.
	} else if !status.HasBoardConfig && shouldGenerate(opts, "Configure task source? [Y/n]: ") {
		sourceName := status.Source // "trello" or ""
		if sourceName == "" && opts.Interactive {
			fmt.Print("  Task source (trello/github) [trello]: ")
			line, err := opts.Reader.ReadString('\n')
			if err == nil {
				if input := strings.TrimSpace(strings.ToLower(line)); input == "github" {
					sourceName = "github"
				}
			}
		}

		if sourceName == "github" {
			if err := ConfigureGitHubSource(opts); err != nil {
				fmt.Fprintf(os.Stderr, "  Error configuring GitHub source: %v\n", err)
			}
		} else {
			var listBoardsFn func() ([]Board, error)
			if status.HasTrelloCreds {
				creds, _ := auth.Load("trello") // Ignore error; missing credentials detected as nil below
				client := trello.NewClient(creds["api_key"], creds["token"])
				listBoardsFn = func() ([]Board, error) {
					boards, err := client.GetBoards()
					if err != nil {
						return nil, err
					}
					result := make([]Board, len(boards))
					for i, b := range boards {
						result[i] = Board{Name: b.Name}
					}
					return result, nil
				}
			}
			if err := ConfigureBoard(opts, listBoardsFn); err != nil {
				fmt.Fprintf(os.Stderr, "  Error configuring board: %v\n", err)
			}
		}
	}

	if status.IsGitRepo {
		if err := EnsureGitignore(dir, []string{".devpilot/logs/"}); err != nil {
			fmt.Fprintf(os.Stderr, "  Error updating .gitignore: %v\n", err)
		}
	}

	fmt.Println("\nDone!")
	fmt.Println("\nTo install Claude Code skills, run:")
	fmt.Println("  npx skills add siyuqian/devpilot")
	return 0
}

func formatStatus(s *Status) []string {
	var lines []string

	if !s.IsGitRepo {
		lines = append(lines, "  ✗ Not a git repository")
	}

	switch s.Source {
	case "github":
		lines = append(lines, "  ✓ Task source: GitHub Issues")
	default: // "trello" or ""
		if s.HasBoardConfig {
			lines = append(lines, "  ✓ Trello board configured")
		} else {
			lines = append(lines, "  ✗ Trello board not configured")
		}
		if s.HasTrelloCreds {
			lines = append(lines, "  ✓ Trello credentials")
		} else {
			lines = append(lines, "  ✗ Trello credentials not found")
		}
	}

	if s.HasSkills {
		lines = append(lines, "  ✓ Skills")
	} else {
		lines = append(lines, "  ✗ Skills not found")
	}

	return lines
}

func allConfigured(s *Status) bool {
	if !s.HasSkills || !s.IsGitRepo {
		return false
	}
	switch s.Source {
	case "github":
		return true // gh CLI auth is already required for PR creation; no extra creds needed
	default: // "trello" or ""
		return s.HasTrelloCreds && s.HasBoardConfig
	}
}

// shouldGenerate returns true if the user confirms or we're in non-interactive mode.
func shouldGenerate(opts GenerateOpts, prompt string) bool {
	if !opts.Interactive {
		return true
	}
	fmt.Print(prompt)
	line, err := opts.Reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "" || answer == "y" || answer == "yes"
}
