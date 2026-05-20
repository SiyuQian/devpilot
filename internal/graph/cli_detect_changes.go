package graph

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/siyuqian/devpilot/internal/graph/envelope"
	"github.com/siyuqian/devpilot/internal/graph/query"
)

func runDetectChanges(repo, base, head string) int {
	e := envelope.New("graph.detect-changes")
	abs, err := resolveRepo(repo)
	if err != nil {
		return emit(e.Err("repo_invalid", err.Error()), "detect_changes.v1.json")
	}
	if base == "" || head == "" {
		return emit(e.Err("args_required", "--base and --head are required"), "detect_changes.v1.json")
	}
	st, _, err := openStore(abs)
	if err != nil {
		return emit(e.Err("cache_missing", err.Error()).Suggest("devpilot graph build --repo "+abs), "detect_changes.v1.json")
	}
	defer func() { _ = st.Close() }()
	ch, err := query.DetectChanges(st, abs, base, head)
	if err != nil {
		return emit(e.Err("detect_failed", err.Error()), "detect_changes.v1.json")
	}
	out := make([]map[string]any, 0, len(ch))
	for _, c := range ch {
		out = append(out, map[string]any{
			"id":          c.ID,
			"kind":        c.Kind,
			"is_exported": c.IsExported,
			"is_new":      c.IsNew,
			"change_type": c.ChangeType,
		})
	}
	e.Suggest("devpilot graph preflight --base " + base + " --head " + head + " --repo " + abs)
	return emit(e.OK(map[string]any{"changed_symbols": out}), "detect_changes.v1.json")
}

func detectChangesCmd() *cobra.Command {
	var repo, base, head string
	c := &cobra.Command{
		Use:   "detect-changes",
		Short: "List symbols changed between two git refs",
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runDetectChanges(repo, base, head))
		},
	}
	c.Flags().StringVar(&repo, "repo", "", "repo root")
	c.Flags().StringVar(&base, "base", "", "base git ref")
	c.Flags().StringVar(&head, "head", "", "head git ref")
	return c
}
