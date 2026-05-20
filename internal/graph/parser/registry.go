package parser

import (
	"path/filepath"
	"strings"
)

// Registry maps file extensions to the Parser that handles them.
type Registry struct {
	byExt map[string]Parser
}

// DefaultRegistry returns a Registry populated with all built-in parsers
// (Go, TypeScript, JavaScript, Rust).
func DefaultRegistry() *Registry {
	r := &Registry{byExt: make(map[string]Parser)}
	for _, p := range []Parser{
		NewGoParser(),
		NewTypeScriptParser(),
		NewJavaScriptParser(),
		NewRustParser(),
	} {
		for _, ext := range p.Extensions() {
			r.byExt[strings.ToLower(ext)] = p
		}
	}
	return r
}

// ForPath returns the Parser registered for the file's extension, or nil
// if no parser is registered for that extension.
func (r *Registry) ForPath(path string) Parser {
	if r == nil {
		return nil
	}
	return r.byExt[strings.ToLower(filepath.Ext(path))]
}

// Languages returns the deduplicated list of languages supported by the
// registry.
func (r *Registry) Languages() []string {
	if r == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, p := range r.byExt {
		lang := p.Language()
		if _, ok := seen[lang]; ok {
			continue
		}
		seen[lang] = struct{}{}
		out = append(out, lang)
	}
	return out
}
