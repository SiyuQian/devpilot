package graph

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/siyuqian/devpilot/internal/graph/envelope"
	"github.com/siyuqian/devpilot/internal/graph/query"
)

func runImpact(repo, filesCSV string, depth int) int {
	e := envelope.New("graph.impact")
	abs, err := resolveRepo(repo)
	if err != nil {
		return emit(e.Err("repo_invalid", err.Error()), "impact.v1.json")
	}
	if filesCSV == "" {
		return emit(e.Err("args_required", "--files is required"), "impact.v1.json")
	}
	files := strings.Split(filesCSV, ",")
	st, _, err := openStore(abs)
	if err != nil {
		return emit(e.Err("cache_missing", err.Error()).Suggest("devpilot graph build --repo "+abs), "impact.v1.json")
	}
	defer func() { _ = st.Close() }()
	im, err := query.ImpactRadius(st, files, defaultInt(depth, 2))
	if err != nil {
		return emit(e.Err("impact_failed", err.Error()), "impact.v1.json")
	}
	changed := im.ChangedSymbols
	if changed == nil {
		changed = []string{}
	}
	return emit(e.OK(map[string]any{
		"changed_symbols": changed,
		"callers":         callersToMaps(im.Symbols),
	}), "impact.v1.json")
}

func impactCmd() *cobra.Command {
	var repo, files string
	var depth int
	c := &cobra.Command{
		Use:   "impact",
		Short: "Union of callers for symbols defined in the given files",
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runImpact(repo, files, depth))
		},
	}
	c.Flags().StringVar(&repo, "repo", "", "repo root")
	c.Flags().StringVar(&files, "files", "", "comma-separated file paths")
	c.Flags().IntVar(&depth, "depth", 0, "max caller hops")
	return c
}
