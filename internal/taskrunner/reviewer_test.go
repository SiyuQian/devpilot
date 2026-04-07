package taskrunner

import (
	"testing"
)

func TestIsApproved_StructuredVerdict(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		want   bool
	}{
		{
			"approved with structured output",
			"## Summary\n\nGood PR.\n\n## Verdict\n\nAPPROVE\n\nNo blocking issues.\n",
			true,
		},
		{
			"request changes with structured output",
			"## Summary\n\nHas bugs.\n\n## Verdict\n\nREQUEST_CHANGES\n\nFix the injection.\n",
			false,
		},
		{
			"empty output",
			"",
			false,
		},
		{
			"no verdict section",
			"Some random review text without structure",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsApproved(tt.stdout)
			if got != tt.want {
				t.Errorf("IsApproved() = %v, want %v", got, tt.want)
			}
		})
	}
}
