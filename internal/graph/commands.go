// Package graph hosts the `devpilot graph` Cobra surface. Subcommands live in
// sibling files (cli_<verb>.go) but are registered here, matching the project
// convention of one commands.go per domain.
package graph

import (
	"os"

	"github.com/spf13/cobra"
)

// RegisterCommands installs the `graph` parent and all subcommands.
func RegisterCommands(parent *cobra.Command) {
	g := &cobra.Command{
		Use:   "graph",
		Short: "Code-graph build and query commands",
	}
	g.AddCommand(
		buildCmd(),
		statusCmd(),
		cleanCmd(),
		queryCmd(),
		impactCmd(),
		hubsCmd(),
		contextCmd(),
		detectChangesCmd(),
		preflightCmd(),
	)
	parent.AddCommand(g)
}

func buildCmd() *cobra.Command {
	var repo string
	c := &cobra.Command{
		Use:   "build [repo]",
		Short: "Build or refresh the graph cache for a repo",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			r := repo
			if r == "" && len(args) > 0 {
				r = args[0]
			}
			os.Exit(runBuild(r))
		},
	}
	c.Flags().StringVar(&repo, "repo", "", "repo root (default: cwd)")
	return c
}
