package initcmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/auth"
	"github.com/siyuqian/developer-kit/internal/trello"
)

// RegisterCommands adds the init command to the parent command.
func RegisterCommands(parent *cobra.Command) {
	initCmd.Flags().BoolP("yes", "y", false, "Accept all defaults without prompting")
	parent.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a project for use with devkit",
	Long:  "Detect existing project configuration, report current state, and generate missing pieces.",
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to get working directory:", err)
			os.Exit(1)
		}

		status := Detect(dir)

		fmt.Println("Scanning project...")
		for _, line := range formatStatus(status) {
			fmt.Println(line)
		}

		if allConfigured(status) {
			fmt.Println("\nProject already initialized!")
			return
		}

		yes, _ := cmd.Flags().GetBool("yes")
		opts := GenerateOpts{
			Dir:         dir,
			Interactive: !yes,
		}
		if opts.Interactive {
			opts.Reader = bufio.NewReader(os.Stdin)
		}

		fmt.Println()

		// CLAUDE.md
		if !status.HasClaudeMD {
			if shouldGenerate(opts, "Generate CLAUDE.md? [Y/n]: ") {
				if err := GenerateClaudeMD(opts); err != nil {
					fmt.Fprintf(os.Stderr, "  Error generating CLAUDE.md: %v\n", err)
				}
			}
		}

		// Board configuration
		if !status.HasBoardConfig {
			var listBoardsFn func() ([]Board, error)
			if status.HasTrelloCreds {
				creds, _ := auth.Load("trello")
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

		// Gitignore
		if status.IsGitRepo {
			if err := EnsureGitignore(dir, []string{".devkit/logs/"}); err != nil {
				fmt.Fprintf(os.Stderr, "  Error updating .gitignore: %v\n", err)
			}
		}

		// Skills
		if !status.HasSkills {
			if shouldGenerate(opts, "Create an initial skill? [Y/n]: ") {
				if err := CreateSkill(opts); err != nil {
					fmt.Fprintf(os.Stderr, "  Error creating skill: %v\n", err)
				}
			}
		}

		fmt.Println("\nDone!")
	},
}

func formatStatus(s *Status) []string {
	var lines []string

	if !s.IsGitRepo {
		lines = append(lines, "  ✗ Not a git repository")
	}

	if s.HasClaudeMD {
		lines = append(lines, "  ✓ CLAUDE.md")
	} else {
		lines = append(lines, "  ✗ CLAUDE.md not found")
	}

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

	if s.HasSkills {
		lines = append(lines, "  ✓ Skills")
	} else {
		lines = append(lines, "  ✗ Skills not found")
	}

	return lines
}

func allConfigured(s *Status) bool {
	return s.HasClaudeMD && s.HasTrelloCreds && s.HasBoardConfig && s.HasSkills && s.IsGitRepo
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
