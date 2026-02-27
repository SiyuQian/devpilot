package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/services"
)

var loginCmd = &cobra.Command{
	Use:   "login <service>",
	Short: "Log in to a service",
	Long:  fmt.Sprintf("Authenticate with an external service.\n\nAvailable services: %s", services.AvailableNames()),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, err := services.Get(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := svc.Login(); err != nil {
			fmt.Fprintln(os.Stderr, "Login failed:", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
