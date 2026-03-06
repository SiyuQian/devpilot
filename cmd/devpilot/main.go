package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/siyuqian/devpilot/internal/generate"
	"github.com/siyuqian/devpilot/internal/gmail"
	"github.com/siyuqian/devpilot/internal/initcmd"
	"github.com/siyuqian/devpilot/internal/openspec"
	"github.com/siyuqian/devpilot/internal/slack"
	"github.com/siyuqian/devpilot/internal/taskrunner"
	"github.com/siyuqian/devpilot/internal/trello"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "devpilot",
		Short: "Developer toolkit for managing service integrations",
		Long:  "devpilot manages authentication and integrations for external services like Trello, GitHub, and more.",
	}

	rootCmd.Version = version

	auth.RegisterCommands(rootCmd)
	initcmd.RegisterCommands(rootCmd)
	trello.RegisterCommands(rootCmd)
	gmail.RegisterCommands(rootCmd)
	slack.RegisterCommands(rootCmd)
	taskrunner.RegisterCommands(rootCmd)
	generate.RegisterCommands(rootCmd)
	openspec.RegisterCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
