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

type contractTestLoader struct{}

func (ct *contractTestLoader) LoadModule(repoRoot string) (map[string]ParseResult, error) {
	return nil, nil
}

func TestPackageLoaderInterfaceContract(t *testing.T) {
	// This test verifies that a struct implementing LoadModule(string) (map[string]ParseResult, error)
	// is assignable to the PackageLoader interface.
	tests := []struct {
		name string
	}{
		{"contract_check"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Verify contractTestLoader is assignable to PackageLoader.
			var loader PackageLoader = &contractTestLoader{}
			_ = loader // Suppress unused variable warning.
		})
	}
}
