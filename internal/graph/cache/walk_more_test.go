package cache

import "testing"

func TestIsHidden(t *testing.T) {
	if !IsHidden(".git") || !IsHidden(".devpilot") {
		t.Fatalf("expected hidden paths")
	}
	if IsHidden("src") {
		t.Fatalf("src should not be hidden")
	}
}
