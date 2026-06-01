package auth

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// RegisterCommands registers the auth-related subcommands (login, logout,
// status) on the given parent command.
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
		os.Exit(runLogin(args[0]))
	},
}

func runLogin(service string) int {
	svc, err := Get(service)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := svc.Login(); err != nil {
		fmt.Fprintln(os.Stderr, "Login failed:", err)
		return 1
	}
	return 0
}

var logoutCmd = &cobra.Command{
	Use:   "logout <service>",
	Short: "Log out of a service",
	Long:  fmt.Sprintf("Remove stored credentials for a service.\n\nAvailable services: %s", AvailableNames()),
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(runLogout(args[0]))
	},
}

func runLogout(service string) int {
	svc, err := Get(service)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := svc.Logout(); err != nil {
		fmt.Fprintln(os.Stderr, "Logout failed:", err)
		return 1
	}
	return 0
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show login status for all services",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		runStatus()
	},
}

func runStatus() {
	loggedIn := ListServices()
	if len(loggedIn) == 0 {
		fmt.Println("No services configured.")
		fmt.Printf("Run 'devpilot login <service>' to get started. Available: %s\n", AvailableNames())
		return
	}
	for _, name := range loggedIn {
		fmt.Printf("%s: logged in\n", name)
	}
}
