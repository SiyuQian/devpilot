package resolver

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

func TestTSConfigResolverRewritesAliasImports(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "parser", "testdata", "ts", "alias"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewTSConfigResolver(root)
	if err != nil {
		t.Fatal(err)
	}
	edges := []store.Edge{
		{Src: "src/a.ts", Dst: "external::@lib/b", Kind: "imports"},
	}
	got := r.Rewrite(edges)
	if len(got) != 1 {
		t.Fatalf("want 1 edge, got %d", len(got))
	}
	if got[0].Dst != "src/lib/b.ts" {
		t.Errorf("dst=%q want src/lib/b.ts", got[0].Dst)
	}
}

func TestStripJSONComments(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "line comment stripped",
			in:   "{\n  // a comment\n  \"a\": 1\n}",
			want: "{\n  \n  \"a\": 1\n}",
		},
		{
			name: "simple block comment stripped",
			in:   `{/* hi */"a":1}`,
			want: `{"a":1}`,
		},
		{
			name: "block comment containing // is fully stripped",
			in:   `{/* see // not a line comment */"a":1}`,
			want: `{"a":1}`,
		},
		{
			name: "block comment spanning newlines",
			in:   "{\n/* multi\nline // with slashes\n*/\n\"a\":1}",
			want: "{\n\n\"a\":1}",
		},
		{
			name: "block-comment marker inside string not stripped",
			in:   `{"url":"http://x/*foo*/"}`,
			want: `{"url":"http://x/*foo*/"}`,
		},
		{
			name: "line-comment marker inside string survives",
			in:   `{"url":"https://example.com"}`,
			want: `{"url":"https://example.com"}`,
		},
		{
			name: "unterminated block comment consumes to EOF",
			in:   `{"a":1}/* unterminated`,
			want: `{"a":1}`,
		},
		{
			name: "escaped quote inside string with // does not split",
			in:   `{"a":"he said \"hi\" //x"}`,
			want: `{"a":"he said \"hi\" //x"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := string(stripJSONComments([]byte(tc.in)))
			if got != tc.want {
				t.Errorf("stripJSONComments(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestStripJSONCommentsParsesRealisticTSConfig(t *testing.T) {
	// Regression: /* ... // ... */ used to corrupt the JSON because the
	// embedded `//` triggered the line-comment branch and consumed the
	// `*/` terminator.
	input := []byte(`{
  /* paths config // see docs */
  "compilerOptions": {
    "baseUrl": ".",
    "paths": { "@/*": ["src/*"] }
  }
}`)
	var v map[string]any
	if err := json.Unmarshal(stripJSONComments(input), &v); err != nil {
		t.Fatalf("unmarshal failed: %v\nstripped=%s", err, stripJSONComments(input))
	}
}
