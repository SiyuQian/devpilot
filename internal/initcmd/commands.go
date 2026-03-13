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
			fmt.Println()
		}

		// Task source selection + board/label configuration.
		// If source is already set to "github" in config, nothing more to set up.
		// If source is "trello" (or unset) and no board is configured yet, proceed.
		if status.Source == "github" {
			// Already configured as GitHub Issues — labels may already exist; skip.
		} else if !status.HasBoardConfig {
			// Determine source: respect explicit config value; ask only when truly unset.
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
		}

		// Gitignore
		if status.IsGitRepo {
			if err := EnsureGitignore(dir, []string{".devpilot/logs/"}); err != nil {
				fmt.Fprintf(os.Stderr, "  Error updating .gitignore: %v\n", err)
			}
		}

		// Install skills from devpilot catalog
		if opts.Interactive {
			if err := InstallSkills(opts, nil, nil); err != nil {
				fmt.Fprintf(os.Stderr, "  Error installing skills: %v\n", err)
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
	if !s.HasClaudeMD || !s.HasSkills || !s.IsGitRepo {
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
