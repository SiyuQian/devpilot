package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/auth"
	"github.com/siyuqian/developer-kit/internal/initcmd"
	"github.com/siyuqian/developer-kit/internal/taskrunner"
	"github.com/siyuqian/developer-kit/internal/trello"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "devkit",
		Short: "Developer toolkit for managing service integrations",
		Long:  "devkit manages authentication and integrations for external services like Trello, GitHub, and more.",
	}

	auth.RegisterCommands(rootCmd)
	initcmd.RegisterCommands(rootCmd)
	trello.RegisterCommands(rootCmd)
	taskrunner.RegisterCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
