package query

import (
	"testing"
)

func TestCommunityFromPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"internal/payment/processor.go", "internal/payment"},
		{"api/checkout.go", "api"},
		{"cmd/devpilot/main.go", "cmd/devpilot"},
		{"main.go", ""},
		{"internal/a/b/c/d/e.go", "internal/a/b"}, // depth cap 3
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := communityFromPath(c.in); got != c.want {
				t.Errorf("got=%q want=%q", got, c.want)
			}
		})
	}
}
