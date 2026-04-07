package taskrunner

import (
	"testing"
)

func TestIsApproved_InReviewLoop(t *testing.T) {
	tests := []struct {
		name     string
		stdout   string
		approved bool
	}{
		{"clean structured review", "## Verdict\n\nAPPROVE\n\nAll good.", true},
		{"issues found structured", "## Verdict\n\nREQUEST_CHANGES\n\nFound bugs.", false},
		{"empty output", "", false},
		{"approve buried in text but no section", "This PR gets an APPROVE from me", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsApproved(tt.stdout); got != tt.approved {
				t.Errorf("IsApproved() = %v, want %v", got, tt.approved)
			}
		})
	}
}
