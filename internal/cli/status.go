package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/siyuqian/developer-kit/internal/config"
	"github.com/siyuqian/developer-kit/internal/services"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show login status for all services",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		loggedIn := config.ListServices()
		if len(loggedIn) == 0 {
			fmt.Println("No services configured.")
			fmt.Printf("Run 'devkit login <service>' to get started. Available: %s\n", services.AvailableNames())
			return
		}
		for _, name := range loggedIn {
			fmt.Printf("%s: logged in\n", name)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
