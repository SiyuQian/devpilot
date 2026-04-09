package taskrunner

import (
	"testing"

	"github.com/siyuqian/devpilot/internal/review"
)

func TestIsApproved_PipelineResult(t *testing.T) {
	tests := []struct {
		name    string
		verdict string
		want    bool
	}{
		{"approved", "APPROVE", true},
		{"request changes", "REQUEST_CHANGES", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &review.PipelineResult{Verdict: tt.verdict}
			got := IsApproved(result)
			if got != tt.want {
				t.Errorf("IsApproved() = %v, want %v", got, tt.want)
			}
		})
	}
}
