//go:build bench

package cache

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// findRepoRoot walks upward from this test file until it finds the go.mod that
// declares the devpilot module. Used by native_bench_test.go.
func findRepoRoot(tb testing.TB) string {
	tb.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		tb.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 8; i++ {
		gm := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(gm); err == nil {
			if strings.Contains(string(data), "module github.com/siyuqian/devpilot") {
				abs, err := filepath.Abs(dir)
				if err != nil {
					tb.Fatalf("abs: %v", err)
				}
				return abs
			}
		}
		dir = filepath.Dir(dir)
	}
	tb.Fatal("could not locate devpilot module root")
	return ""
}
