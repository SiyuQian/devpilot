package parser

import "testing"

func TestParseResultZero(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"zero_value"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var r ParseResult
			if r.Nodes != nil || r.Edges != nil || r.Errors != nil {
				t.Fatal("ParseResult zero value must be empty")
			}
		})
	}
}

// Compile-time assertion that any type implementing the contract is assignable
// to PackageLoader. If this file no longer compiles, the interface broke.
var _ PackageLoader = (*packageLoaderContract)(nil)

type packageLoaderContract struct{}

func (packageLoaderContract) LoadModule(repoRoot string) (map[string]ParseResult, error) {
	return nil, nil
}
