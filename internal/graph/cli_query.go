package graph

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/siyuqian/devpilot/internal/graph/envelope"
	"github.com/siyuqian/devpilot/internal/graph/query"
)

type queryOpts struct {
	repo      string
	depth     int
	threshold int
}

func runQuery(o queryOpts, pattern, target string) int {
	e := envelope.New("graph.query")
	abs, err := resolveRepo(o.repo)
	if err != nil {
		return emit(e.Err("repo_invalid", err.Error()), "query.v1.json")
	}
	st, _, err := openStore(abs)
	if err != nil {
		return emit(e.Err("cache_missing", err.Error()).Suggest("devpilot graph build --repo "+abs), "query.v1.json")
	}
	defer func() { _ = st.Close() }()

	switch pattern {
	case "callers_of":
		v, err := query.CallersOf(st, target, defaultInt(o.depth, 2))
		return finishQuery(e, pattern, "callers", callersToMaps(v), err)
	case "callees_of":
		v, err := query.CalleesOf(st, target, defaultInt(o.depth, 2))
		return finishQuery(e, pattern, "callees", calleesToMaps(v), err)
	case "tests_for":
		v, err := query.TestsFor(st, target)
		return finishQuery(e, pattern, "tests", v, err)
	case "implementors_of":
		v, err := query.ImplementorsOf(st, target)
		return finishQuery(e, pattern, "implementors", v, err)
	case "hubs":
		v, err := query.Hubs(st, defaultInt(o.threshold, 10))
		return finishQuery(e, pattern, "hubs", hubsToMaps(v), err)
	case "context":
		v, err := query.Context(st, target, defaultInt(o.depth, 1), abs)
		return finishQuery(e, pattern, "context", contextToMap(v), err)
	default:
		return emit(e.Err("unknown_pattern",
			fmt.Sprintf("pattern %q not in {callers_of,callees_of,tests_for,implementors_of,hubs,context}", pattern)),
			"query.v1.json")
	}
}

func finishQuery(e *envelope.Envelope, pattern, key string, value any, err error) int {
	if err != nil {
		return emit(e.Err("query_failed", err.Error()), "query.v1.json")
	}
	if value == nil {
		// Normalise nil slices to empty so schema oneOf branches work cleanly.
		value = []any{}
	}
	return emit(e.OK(map[string]any{
		"pattern":        pattern,
		"pattern_result": map[string]any{key: value},
	}), "query.v1.json")
}

func defaultInt(v, d int) int {
	if v <= 0 {
		return d
	}
	return v
}

func callersToMaps(in []query.Caller) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, c := range in {
		out = append(out, map[string]any{"id": c.ID, "hop": c.Hop})
	}
	return out
}

func calleesToMaps(in []query.Callee) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, c := range in {
		out = append(out, map[string]any{"id": c.ID, "hop": c.Hop})
	}
	return out
}

func hubsToMaps(in []query.Hub) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, h := range in {
		out = append(out, map[string]any{"id": h.ID, "caller_count": h.CallerCount})
	}
	return out
}

func contextToMap(c query.ContextResult) map[string]any {
	return map[string]any{
		"target":  snippetToMap(c.Target),
		"callers": snippetsToMaps(c.Callers),
	}
}

func snippetToMap(s query.Snippet) map[string]any {
	return map[string]any{
		"id":         s.ID,
		"path":       s.Path,
		"start_line": s.StartLine,
		"end_line":   s.EndLine,
		"source":     s.Source,
	}
}

func snippetsToMaps(in []query.Snippet) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, s := range in {
		out = append(out, snippetToMap(s))
	}
	return out
}

func queryCmd() *cobra.Command {
	var o queryOpts
	c := &cobra.Command{
		Use:   "query <pattern> [target]",
		Short: "Run a saved query pattern against the graph",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			pattern := args[0]
			target := ""
			if len(args) > 1 {
				target = args[1]
			}
			os.Exit(runQuery(o, pattern, target))
		},
	}
	c.Flags().StringVar(&o.repo, "repo", "", "repo root (default: cwd)")
	c.Flags().IntVar(&o.depth, "depth", 0, "max BFS depth (callers_of, callees_of, context)")
	c.Flags().IntVar(&o.threshold, "threshold", 0, "min inbound calls (hubs)")
	return c
}
