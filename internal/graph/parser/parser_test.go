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
