package graph

import (
	"github.com/spf13/cobra"
	"os"

	"github.com/siyuqian/devpilot/internal/graph/cache"
	"github.com/siyuqian/devpilot/internal/graph/envelope"
)

func runStatus(repo string) int {
	e := envelope.New("graph.status")
	abs, err := resolveRepo(repo)
	if err != nil {
		return emit(e.Err("repo_invalid", err.Error()), "status.v1.json")
	}
	key := cache.RepoKey(abs)
	meta, err := cache.ReadMeta(cache.MetaFile(cache.Home(), key))
	if err != nil {
		return emit(e.Err("meta_unreadable", err.Error()), "status.v1.json")
	}
	st, _, err := openStore(abs)
	if err != nil {
		return emit(e.Err("cache_missing", err.Error()).Suggest("devpilot graph build --repo "+abs), "status.v1.json")
	}
	defer func() { _ = st.Close() }()
	nodes, err := st.CountNodes()
	if err != nil {
		return emit(e.Err("count_failed", err.Error()), "status.v1.json")
	}
	edges, err := st.CountEdges()
	if err != nil {
		return emit(e.Err("count_failed", err.Error()), "status.v1.json")
	}
	langs := meta.Languages
	if langs == nil {
		langs = []string{}
	}
	return emit(e.OK(map[string]any{
		"repo":          abs,
		"head_sha":      meta.HeadSHA,
		"languages":     langs,
		"nodes":         nodes,
		"edges":         edges,
		"built_at_unix": meta.BuiltAtUnix,
	}), "status.v1.json")
}

func statusCmd() *cobra.Command {
	var repo string
	c := &cobra.Command{
		Use:   "status",
		Short: "Report graph-cache state for a repo",
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runStatus(repo))
		},
	}
	c.Flags().StringVar(&repo, "repo", "", "repo root (default: cwd)")
	return c
}
