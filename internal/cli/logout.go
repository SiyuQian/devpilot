package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/services"
)

var logoutCmd = &cobra.Command{
	Use:   "logout <service>",
	Short: "Log out of a service",
	Long:  fmt.Sprintf("Remove stored credentials for a service.\n\nAvailable services: %s", services.AvailableNames()),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, err := services.Get(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := svc.Logout(); err != nil {
			fmt.Fprintln(os.Stderr, "Logout failed:", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
