package cache

import "testing"

func TestHomeUsesEnvironment(t *testing.T) {
	t.Setenv("DEVPILOT_HOME", "/tmp/devpilot-home")
	if got := Home(); got != "/tmp/devpilot-home" {
		t.Fatalf("Home() = %q", got)
	}
}
