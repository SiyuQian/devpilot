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
