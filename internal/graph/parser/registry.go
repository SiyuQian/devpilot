package parser

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Registry maps file extensions to the Parser that handles them.
type Registry struct {
	byExt map[string]Parser
}

// goBackendFromEnv returns the configured Go parser backend based on the
// DEVPILOT_GRAPH_GO_BACKEND environment variable.
// Returns "native" if the flag is set to exactly "native", otherwise "treesitter".
func goBackendFromEnv() string {
	if v := os.Getenv("DEVPILOT_GRAPH_GO_BACKEND"); v == "native" {
		return "native"
	}
	return "treesitter"
}

// DefaultRegistry returns a Registry populated with all built-in parsers
// (Go, TypeScript, JavaScript, Rust). The Go parser backend is selected based on
// the DEVPILOT_GRAPH_GO_BACKEND environment variable: "native" for GoNativeParser,
// or "treesitter" (default) for GoParser.
func DefaultRegistry() *Registry {
	r := &Registry{byExt: make(map[string]Parser)}
	var goParser Parser = NewGoParser()
	if goBackendFromEnv() == "native" {
		goParser = NewGoNativeParser()
	}
	for _, p := range []Parser{
		goParser,
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

// ForLanguage returns the registered parser whose Language() matches lang,
// or nil if no parser claims that language. Used by callers (e.g. cache.Builder)
// that need a language-keyed lookup instead of file-extension routing.
func (r *Registry) ForLanguage(lang string) Parser {
	if r == nil {
		return nil
	}
	for _, p := range r.byExt {
		if p.Language() == lang {
			return p
		}
	}
	return nil
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
	sort.Strings(out)
	return out
}

// GoBackend returns the Go parser backend in use by this registry.
// Returns "native" if the .go files are handled by GoNativeParser, or "treesitter" otherwise.
func (r *Registry) GoBackend() string {
	if r == nil {
		return "treesitter"
	}
	p := r.byExt[".go"]
	if _, ok := p.(*GoNativeParser); ok {
		return "native"
	}
	return "treesitter"
}
