package parser

import (
	"sort"
	"testing"
)

func TestRegistryByExtension(t *testing.T) {
	r := DefaultRegistry()

	cases := []struct {
		path string
		want string // expected Language(); "" means no parser
	}{
		{"foo.go", "go"},
		{"foo.ts", "typescript"},
		{"foo.tsx", "typescript"},
		{"foo.js", "javascript"},
		{"foo.mjs", "javascript"},
		{"foo.rs", "rust"},
		{"foo.png", ""},
		{"FOO.GO", "go"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			p := r.ForPath(tc.path)
			if tc.want == "" {
				if p != nil {
					t.Fatalf("ForPath(%q) = %s, want nil", tc.path, p.Language())
				}
				return
			}
			if p == nil {
				t.Fatalf("ForPath(%q) = nil, want %s", tc.path, tc.want)
			}
			if got := p.Language(); got != tc.want {
				t.Errorf("ForPath(%q).Language() = %q, want %q", tc.path, got, tc.want)
			}
		})
	}

	langs := r.Languages()
	sort.Strings(langs)
	wantLangs := []string{"go", "javascript", "rust", "typescript"}
	if len(langs) != len(wantLangs) {
		t.Fatalf("Languages() = %v, want %v", langs, wantLangs)
	}
	for i, l := range wantLangs {
		if langs[i] != l {
			t.Errorf("Languages()[%d] = %q, want %q", i, langs[i], l)
		}
	}
}

func TestDefaultRegistryGoBackendFlag(t *testing.T) {
	tests := []struct {
		name           string
		envValue       string
		wantBackend    string
		wantParserType string // "GoNativeParser" or "GoParser"
	}{
		{
			name:           "unset defaults to treesitter",
			envValue:       "",
			wantBackend:    "treesitter",
			wantParserType: "GoParser",
		},
		{
			name:           "native flag selects native backend",
			envValue:       "native",
			wantBackend:    "native",
			wantParserType: "GoNativeParser",
		},
		{
			name:           "invalid value falls back to treesitter",
			envValue:       "garbage",
			wantBackend:    "treesitter",
			wantParserType: "GoParser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DEVPILOT_GRAPH_GO_BACKEND", tt.envValue)

			r := DefaultRegistry()

			// Check GoBackend() returns expected value
			if got := r.GoBackend(); got != tt.wantBackend {
				t.Errorf("GoBackend() = %q, want %q", got, tt.wantBackend)
			}

			// Check ForPath("x.go") returns the correct parser type
			p := r.ForPath("x.go")
			if p == nil {
				t.Fatalf("ForPath(\"x.go\") = nil, want parser")
			}

			// Type-switch to verify parser type
			switch tt.wantParserType {
			case "GoNativeParser":
				if _, ok := p.(*GoNativeParser); !ok {
					t.Errorf("ForPath(\"x.go\") returned %T, want *GoNativeParser", p)
				}
			case "GoParser":
				if _, ok := p.(*GoParser); !ok {
					t.Errorf("ForPath(\"x.go\") returned %T, want *GoParser", p)
				}
			}
		})
	}
}
