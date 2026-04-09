package taskrunner

import (
	"testing"

	"github.com/siyuqian/devpilot/internal/review"
)

func TestIsApproved_InReviewLoop(t *testing.T) {
	tests := []struct {
		name     string
		verdict  string
		approved bool
	}{
		{"approved", "APPROVE", true},
		{"request changes", "REQUEST_CHANGES", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &review.PipelineResult{Verdict: tt.verdict}
			if got := IsApproved(result); got != tt.approved {
				t.Errorf("IsApproved() = %v, want %v", got, tt.approved)
			}
		})
	}
}
