// Package resolver — additions for TypeScript path aliases.
package resolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

type tsConfigFile struct {
	Extends         string `json:"extends"`
	CompilerOptions struct {
		BaseUrl string              `json:"baseUrl"`
		Paths   map[string][]string `json:"paths"`
	} `json:"compilerOptions"`
}

// TSConfigResolver rewrites import edges whose dst matches a tsconfig path alias
// into edges pointing at the resolved on-disk file path (relative to repo root).
type TSConfigResolver struct {
	root    string
	baseURL string
	paths   map[string][]string
}

// NewTSConfigResolver loads tsconfig.json from root (handling `extends` once).
func NewTSConfigResolver(root string) (*TSConfigResolver, error) {
	r := &TSConfigResolver{root: root, paths: map[string][]string{}}
	if err := r.loadTSConfig(filepath.Join(root, "tsconfig.json")); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *TSConfigResolver) loadTSConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	var cfg tsConfigFile
	if err := json.Unmarshal(stripJSONComments(data), &cfg); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Extends != "" {
		parent := filepath.Join(filepath.Dir(path), cfg.Extends)
		if !strings.HasSuffix(parent, ".json") {
			parent += ".json"
		}
		if err := r.loadTSConfig(parent); err != nil {
			return err
		}
	}
	if cfg.CompilerOptions.BaseUrl != "" {
		r.baseURL = filepath.Join(filepath.Dir(path), cfg.CompilerOptions.BaseUrl)
	}
	for k, v := range cfg.CompilerOptions.Paths {
		r.paths[k] = v
	}
	return nil
}

// Rewrite walks edges and rewrites `external::<alias>` import dsts to repo-relative file paths.
func (r *TSConfigResolver) Rewrite(edges []store.Edge) []store.Edge {
	out := make([]store.Edge, len(edges))
	for i, e := range edges {
		out[i] = e
		if e.Kind != "imports" || !strings.HasPrefix(e.Dst, "external::") {
			continue
		}
		spec := strings.TrimPrefix(e.Dst, "external::")
		if rel, ok := r.resolve(spec); ok {
			out[i].Dst = rel
		}
	}
	return out
}

func (r *TSConfigResolver) resolve(spec string) (string, bool) {
	for pattern, targets := range r.paths {
		prefix := strings.TrimSuffix(pattern, "*")
		if !strings.HasPrefix(spec, prefix) {
			continue
		}
		tail := strings.TrimPrefix(spec, prefix)
		for _, tmpl := range targets {
			candidate := filepath.Join(r.baseURL, strings.Replace(tmpl, "*", tail, 1))
			for _, ext := range []string{".ts", ".tsx", "/index.ts"} {
				p := candidate + ext
				if info, err := os.Stat(p); err == nil && !info.IsDir() {
					rel, err := filepath.Rel(r.root, p)
					if err != nil {
						return "", false
					}
					return filepath.ToSlash(rel), true
				}
			}
		}
	}
	return "", false
}

func stripJSONComments(b []byte) []byte {
	// tsconfig allows // line comments. Strip them naively.
	lines := strings.Split(string(b), "\n")
	for i, l := range lines {
		if idx := strings.Index(l, "//"); idx >= 0 {
			lines[i] = l[:idx]
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
