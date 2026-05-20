package envelope

import (
	"testing"
)

func TestValidateAcceptsValidStatus(t *testing.T) {
	e := New("graph.status").OK(map[string]any{
		"repo":          "/tmp/r",
		"nodes":         10,
		"edges":         20,
		"built_at_unix": 1,
		"head_sha":      "deadbeef",
		"languages":     []string{"go"},
	})
	b, _ := e.Marshal()
	if err := Validate(b, "status.v1.json"); err != nil {
		t.Fatalf("want valid, got %v\n%s", err, b)
	}
}

func TestValidateRejectsMissingCommand(t *testing.T) {
	bad := []byte(`{"schema_version":"1","ok":true,"data":null,"error":null,"warnings":[],"next_tool_suggestions":[],"elapsed_ms":0}`)
	if err := Validate(bad, "envelope.v1.json"); err == nil {
		t.Fatal("want error for missing command field")
	}
}

func TestValidateUnknownSchema(t *testing.T) {
	if err := Validate([]byte(`{}`), "no_such.json"); err == nil {
		t.Fatal("want error for unknown schema id")
	}
}

func TestValidateRejectsBadCommandPattern(t *testing.T) {
	e := New("not.graph").OK(nil)
	b, _ := e.Marshal()
	if err := Validate(b, "envelope.v1.json"); err == nil {
		t.Fatal("want pattern violation")
	}
}
