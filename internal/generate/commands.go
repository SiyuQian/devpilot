package generate

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/siyuqian/devpilot/internal/project"
)

func RegisterCommands(parent *cobra.Command) {
	commitCmd.Flags().StringP("message", "m", "", "Additional context for AI")
	commitCmd.Flags().String("model", "", "Override Claude model")
	commitCmd.Flags().Bool("dry-run", false, "Generate message without committing")

	readmeCmd.Flags().String("model", "", "Override Claude model")
	readmeCmd.Flags().Bool("dry-run", false, "Generate without writing file")

	parent.AddCommand(commitCmd)
	parent.AddCommand(readmeCmd)
}

func resolveModel(cmd *cobra.Command, command string) string {
	if m, _ := cmd.Flags().GetString("model"); m != "" {
		return m
	}
	dir, _ := os.Getwd()
	cfg, _ := project.Load(dir)
	return cfg.ModelFor(command)
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate an AI-powered commit message and commit",
	Long:  "Stages all changes with git add ., generates a conventional commit message using Claude AI, and commits after user confirmation.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		model := resolveModel(cmd, "commit")
		msg, _ := cmd.Flags().GetString("message")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := RunCommit(ctx, model, msg, dryRun); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

var readmeCmd = &cobra.Command{
	Use:   "readme",
	Short: "Generate a README.md using AI",
	Long:  "Analyzes your project structure and generates a professional README.md using Claude AI.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		model := resolveModel(cmd, "readme")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := RunReadme(ctx, model, dryRun); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}
