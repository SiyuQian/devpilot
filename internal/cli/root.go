package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "devkit",
	Short: "Developer toolkit for managing service integrations",
	Long:  "devkit manages authentication and integrations for external services like Trello, GitHub, and more.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
