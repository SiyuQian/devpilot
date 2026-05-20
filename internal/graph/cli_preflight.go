package graph

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/siyuqian/devpilot/internal/graph/envelope"
	"github.com/siyuqian/devpilot/internal/graph/query"
)

func runPreflight(repo, base, head string) int {
	e := envelope.New("graph.preflight")
	abs, err := resolveRepo(repo)
	if err != nil {
		return emit(e.Err("repo_invalid", err.Error()), "preflight.v1.json")
	}
	if base == "" || head == "" {
		return emit(e.Err("args_required", "--base and --head are required"), "preflight.v1.json")
	}
	st, _, err := openStore(abs)
	if err != nil {
		return emit(e.Err("cache_missing", err.Error()).Suggest("devpilot graph build --repo "+abs), "preflight.v1.json")
	}
	defer func() { _ = st.Close() }()
	res, err := query.Preflight(st, query.PreflightInput{
		RepoRoot: abs, Base: base, Head: head,
	})
	if err != nil {
		return emit(e.Err("preflight_failed", err.Error()), "preflight.v1.json")
	}
	for i, s := range res.ChangedSymbols {
		if i >= 3 {
			break
		}
		e.Suggest("devpilot graph context --id " + s.ID + " --depth 1 --repo " + abs)
	}
	return emit(e.OK(res), "preflight.v1.json")
}

func preflightCmd() *cobra.Command {
	var repo, base, head string
	c := &cobra.Command{
		Use:   "preflight",
		Short: "Emit the §6 preflight payload for a PR diff",
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runPreflight(repo, base, head))
		},
	}
	c.Flags().StringVar(&repo, "repo", "", "repo root")
	c.Flags().StringVar(&base, "base", "", "base git ref")
	c.Flags().StringVar(&head, "head", "", "head git ref")
	return c
}
