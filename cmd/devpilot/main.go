package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/siyuqian/devpilot/internal/auth"
	"github.com/siyuqian/devpilot/internal/initcmd"
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
	taskrunner.RegisterCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
