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
	reviewCmd.Flags().String("model", "", "Override Claude model for review (default: "+DefaultReviewModel+")")
	reviewCmd.Flags().String("scoring-model", "", "Override model for scoring round (default: "+DefaultScoringModel+")")
	reviewCmd.Flags().Int("threshold", DefaultThreshold, "Minimum confidence score to include a finding (0-100)")
	reviewCmd.Flags().Bool("dry-run", false, "Print assembled prompt without executing Claude")
	reviewCmd.Flags().Int("timeout", 10, "Review timeout in minutes")
	reviewCmd.Flags().Bool("no-post", false, "Skip posting review to GitHub PR")

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
	Long:  "Performs a thorough code review using a multi-round pipeline: Opus reviews, Haiku scores confidence, Go posts results.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		prURL := args[0]
		model := resolveModel(cmd)
		scoringModel, _ := cmd.Flags().GetString("scoring-model")
		threshold, _ := cmd.Flags().GetInt("threshold")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		timeoutMin, _ := cmd.Flags().GetInt("timeout")
		noPost, _ := cmd.Flags().GetBool("no-post")

		// Validate PR URL early
		pr, err := ParsePRURL(prURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		if dryRun {
			rc, err := GatherContext(pr)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error gathering context:", err)
				os.Exit(1)
			}
			prompt := BuildPrompt(rc, rc.Diff)
			fmt.Println(prompt)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMin)*time.Minute)
		defer cancel()

		var pipelineOpts []PipelineOption
		pipelineOpts = append(pipelineOpts, WithReviewModel(model))
		if scoringModel != "" {
			pipelineOpts = append(pipelineOpts, WithScoringModel(scoringModel))
		}
		pipelineOpts = append(pipelineOpts, WithThreshold(threshold))
		pipelineOpts = append(pipelineOpts, WithPipelinePostToGitHub(!noPost))

		streamer := newReviewStreamer()
		pipelineOpts = append(pipelineOpts, WithPipelineEventHandler(streamer.HandleEvent))

		result, err := Review(ctx, prURL, pipelineOpts...)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		// Output human-readable review
		fmt.Print(FormatMarkdown(result))
	},
}
