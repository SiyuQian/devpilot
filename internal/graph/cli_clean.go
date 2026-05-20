package graph

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/siyuqian/devpilot/internal/graph/cache"
	"github.com/siyuqian/devpilot/internal/graph/envelope"
)

func runClean(repo string, all bool) int {
	e := envelope.New("graph.clean")
	if !all && repo == "" {
		return emit(e.Err("args_required", "pass --repo or --all"), "clean.v1.json")
	}
	home := cache.Home()
	if all {
		dir := filepath.Join(home, "graphs")
		if err := os.RemoveAll(dir); err != nil {
			return emit(e.Err("remove_failed", err.Error()), "clean.v1.json")
		}
		return emit(e.OK(map[string]any{"removed": dir, "all": true}), "clean.v1.json")
	}
	abs, err := resolveRepo(repo)
	if err != nil {
		return emit(e.Err("repo_invalid", err.Error()), "clean.v1.json")
	}
	dir := cache.GraphDir(home, cache.RepoKey(abs))
	if err := os.RemoveAll(dir); err != nil {
		return emit(e.Err("remove_failed", err.Error()), "clean.v1.json")
	}
	return emit(e.OK(map[string]any{"removed": dir, "all": false}), "clean.v1.json")
}

func cleanCmd() *cobra.Command {
	var repo string
	var all bool
	c := &cobra.Command{
		Use:   "clean",
		Short: "Delete graph cache for a repo (--repo) or every repo (--all)",
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runClean(repo, all))
		},
	}
	c.Flags().StringVar(&repo, "repo", "", "repo root")
	c.Flags().BoolVar(&all, "all", false, "remove every cached graph")
	return c
}
