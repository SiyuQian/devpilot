package graph

import (
	"errors"

	"github.com/siyuqian/devpilot/internal/graph/cache"
	"github.com/siyuqian/devpilot/internal/graph/envelope"
)

func runBuild(repo string) int {
	e := envelope.New("graph.build")
	abs, err := resolveRepo(repo)
	if err != nil {
		return emit(e.Err("repo_invalid", err.Error()), "build.v1.json")
	}
	b, err := cache.NewBuilder(cache.Home(), abs)
	if err != nil {
		return emit(e.Err("builder_init", err.Error()), "build.v1.json")
	}
	res, err := b.Build()
	if err != nil {
		if errors.Is(err, cache.ErrNoGoModule) {
			e.Suggest("add a go.mod or go.work at the repo root, or run \"go mod init\" to enable the Go graph build")
			return emit(e.Err("go_no_module", err.Error()), "build.v1.json")
		}
		return emit(e.Err("build_failed", err.Error()), "build.v1.json")
	}
	e.Suggest("devpilot graph status --repo " + abs)
	return emit(e.OK(map[string]any{
		"repo":         abs,
		"mode":         res.Mode,
		"files_parsed": res.FilesParsed,
		"nodes":        res.NodesInsert,
		"edges":        res.EdgesInsert,
	}), "build.v1.json")
}
