package graph

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/siyuqian/devpilot/internal/graph/envelope"
	"github.com/siyuqian/devpilot/internal/graph/query"
)

func runHubs(repo string, threshold int) int {
	e := envelope.New("graph.hubs")
	abs, err := resolveRepo(repo)
	if err != nil {
		return emit(e.Err("repo_invalid", err.Error()), "hubs.v1.json")
	}
	st, _, err := openStore(abs)
	if err != nil {
		return emit(e.Err("cache_missing", err.Error()).Suggest("devpilot graph build --repo "+abs), "hubs.v1.json")
	}
	defer func() { _ = st.Close() }()
	hs, err := query.Hubs(st, defaultInt(threshold, 10))
	if err != nil {
		return emit(e.Err("hubs_failed", err.Error()), "hubs.v1.json")
	}
	return emit(e.OK(map[string]any{"hubs": hubsToMaps(hs)}), "hubs.v1.json")
}

func hubsCmd() *cobra.Command {
	var repo string
	var threshold int
	c := &cobra.Command{
		Use:   "hubs",
		Short: "List high-fanin nodes (frequent call targets)",
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runHubs(repo, threshold))
		},
	}
	c.Flags().StringVar(&repo, "repo", "", "repo root")
	c.Flags().IntVar(&threshold, "threshold", 0, "min inbound calls (default 10)")
	return c
}
