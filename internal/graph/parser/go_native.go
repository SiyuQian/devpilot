package parser

// GoNativeParser extracts nodes and edges from Go source files using the native
// Go backend (whole-module analysis). LoadModule is implemented in a later task.
type GoNativeParser struct{}

// NewGoNativeParser returns a Parser for Go source files using the native backend.
func NewGoNativeParser() *GoNativeParser {
	return &GoNativeParser{}
}

func (p *GoNativeParser) Language() string {
	return "go"
}

func (p *GoNativeParser) Extensions() []string {
	return []string{".go"}
}

// Parse is intentionally a no-op; the native backend produces results via
// LoadModule on the whole module, not per-file Parse.
func (p *GoNativeParser) Parse(path string, src []byte) (ParseResult, error) {
	return ParseResult{}, nil
}
