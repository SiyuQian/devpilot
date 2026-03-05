package taskrunner

import "testing"

func TestMaxReviewRetries(t *testing.T) {
	if MaxReviewRetries < 1 {
		t.Error("MaxReviewRetries must be at least 1")
	}
}
