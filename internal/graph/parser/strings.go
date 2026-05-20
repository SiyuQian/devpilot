package parser

// resolveIntra returns the intra-file ID for name if known, otherwise an
// "external::"-prefixed fallback.
func resolveIntra(name, path string, intra map[string]string) string {
	if id, ok := intra[name]; ok {
		return id
	}
	_ = path
	return "external::" + name
}

// unquote strips a matching pair of surrounding quote characters (", ', or `)
// from s. If s is not so quoted, it is returned unchanged.
func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'' || s[0] == '`') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}
