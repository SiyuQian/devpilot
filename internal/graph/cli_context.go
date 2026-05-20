package graph

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/siyuqian/devpilot/internal/graph/envelope"
	"github.com/siyuqian/devpilot/internal/graph/query"
)

func runContext(repo, id string, depth int) int {
	e := envelope.New("graph.context")
	abs, err := resolveRepo(repo)
	if err != nil {
		return emit(e.Err("repo_invalid", err.Error()), "context.v1.json")
	}
	if id == "" {
		return emit(e.Err("args_required", "--id is required"), "context.v1.json")
	}
	st, _, err := openStore(abs)
	if err != nil {
		return emit(e.Err("cache_missing", err.Error()).Suggest("devpilot graph build --repo "+abs), "context.v1.json")
	}
	defer func() { _ = st.Close() }()
	ctx, err := query.Context(st, id, defaultInt(depth, 1), abs)
	if err != nil {
		return emit(e.Err("context_failed", err.Error()), "context.v1.json")
	}
	return emit(e.OK(map[string]any{"context": contextToMap(ctx)}), "context.v1.json")
}

func contextCmd() *cobra.Command {
	var repo, id string
	var depth int
	c := &cobra.Command{
		Use:   "context",
		Short: "Return the source snippet for a symbol and (optionally) its callers",
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runContext(repo, id, depth))
		},
	}
	c.Flags().StringVar(&repo, "repo", "", "repo root")
	c.Flags().StringVar(&id, "id", "", "symbol id")
	c.Flags().IntVar(&depth, "depth", 0, "caller depth (0 = target only)")
	return c
}
