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
		{"clean review", "No issues found. Checked for bugs.", true},
		{"issues found", "Found 2 issues:\n1. Bug", false},
		{"empty output", "", false},
		{"partial match", "No issues found buried in text", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsApproved(tt.stdout); got != tt.approved {
				t.Errorf("IsApproved() = %v, want %v", got, tt.approved)
			}
		})
	}
}
