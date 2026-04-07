package review

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/siyuqian/devpilot/internal/project"
	"github.com/spf13/cobra"
)

// RegisterCommands adds the review subcommand to the given parent command.
func RegisterCommands(parent *cobra.Command) {
	reviewCmd.Flags().String("model", "", "Override Claude model (default: "+DefaultReviewModel+")")
	reviewCmd.Flags().Bool("dry-run", false, "Print assembled prompt without executing Claude")
	reviewCmd.Flags().Int("timeout", 10, "Review timeout in minutes")

	parent.AddCommand(reviewCmd)
}

func resolveModel(cmd *cobra.Command) string {
	if m, _ := cmd.Flags().GetString("model"); m != "" {
		return m
	}
	dir, _ := os.Getwd()
	cfg, _ := project.Load(dir)
	if m := cfg.ModelFor("review"); m != "" {
		return m
	}
	return DefaultReviewModel
}

var reviewCmd = &cobra.Command{
	Use:   "review <pr-url>",
	Short: "AI-powered code review of a GitHub pull request",
	Long:  "Performs a thorough code review using Claude with extended thinking. Gathers project context from the target repository and outputs a structured review.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		prURL := args[0]
		model := resolveModel(cmd)
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		timeoutMin, _ := cmd.Flags().GetInt("timeout")

		// Validate PR URL early
		pr, err := ParsePRURL(prURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		if dryRun {
			dryCtx, dryCancel := context.WithTimeout(context.Background(), time.Duration(timeoutMin)*time.Minute)
			defer dryCancel()
			projectCtx := GatherContext(dryCtx, pr)
			prompt := BuildPrompt(pr, projectCtx)
			fmt.Println(prompt)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMin)*time.Minute)
		defer cancel()

		result, err := Review(ctx, prURL, WithModel(model))
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		fmt.Print(result.Stdout)
		if result.ExitCode != 0 {
			os.Exit(result.ExitCode)
		}
	},
}
