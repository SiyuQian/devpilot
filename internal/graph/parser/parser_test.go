package parser

import "testing"

func TestParseResultZero(t *testing.T) {
	var r ParseResult
	if r.Nodes != nil || r.Edges != nil || r.Errors != nil {
		t.Fatal("ParseResult zero value must be empty")
	}
}
