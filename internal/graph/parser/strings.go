package parser

// unquote strips a matching pair of surrounding quote characters (", ', or `)
// from s. If s is not so quoted, it is returned unchanged.
func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'' || s[0] == '`') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}
