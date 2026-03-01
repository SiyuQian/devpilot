package auth

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func RegisterCommands(parent *cobra.Command) {
	parent.AddCommand(loginCmd)
	parent.AddCommand(logoutCmd)
	parent.AddCommand(statusCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login <service>",
	Short: "Log in to a service",
	Long:  fmt.Sprintf("Authenticate with an external service.\n\nAvailable services: %s", AvailableNames()),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, err := Get(args[0])
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

var logoutCmd = &cobra.Command{
	Use:   "logout <service>",
	Short: "Log out of a service",
	Long:  fmt.Sprintf("Remove stored credentials for a service.\n\nAvailable services: %s", AvailableNames()),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		svc, err := Get(args[0])
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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show login status for all services",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		loggedIn := ListServices()
		if len(loggedIn) == 0 {
			fmt.Println("No services configured.")
			fmt.Printf("Run 'devpilot login <service>' to get started. Available: %s\n", AvailableNames())
			return
		}
		for _, name := range loggedIn {
			fmt.Printf("%s: logged in\n", name)
		}
	},
}
