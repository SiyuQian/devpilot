//go:build bench

package cache

import (
	"runtime"
	"testing"
)

// BenchmarkNativeFullBuild measures cold-cache FullBuild against the devpilot
// repo itself with the native Go backend enabled. Run with:
//
//	go test -tags=bench -bench=. -benchtime=1x ./internal/graph/cache/
func BenchmarkNativeFullBuild(b *testing.B) {
	b.Setenv("DEVPILOT_GRAPH_GO_BACKEND", "native")
	repoRoot := findRepoRoot(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		home := b.TempDir()
		bb, err := NewBuilder(home, repoRoot)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := bb.FullBuild(); err != nil {
			b.Fatal(err)
		}
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b.ReportMetric(float64(m.Sys)/1024/1024, "MB-sys")
	b.ReportMetric(float64(m.HeapAlloc)/1024/1024, "MB-heap")
	b.ReportMetric(float64(m.TotalAlloc)/1024/1024, "MB-total-alloc")
}

// BenchmarkTreesitterFullBuild measures cold-cache FullBuild against the devpilot
// repo itself with the tree-sitter Go backend (default).
func BenchmarkTreesitterFullBuild(b *testing.B) {
	b.Setenv("DEVPILOT_GRAPH_GO_BACKEND", "")
	repoRoot := findRepoRoot(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		home := b.TempDir()
		bb, err := NewBuilder(home, repoRoot)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := bb.FullBuild(); err != nil {
			b.Fatal(err)
		}
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b.ReportMetric(float64(m.Sys)/1024/1024, "MB-sys")
	b.ReportMetric(float64(m.HeapAlloc)/1024/1024, "MB-heap")
	b.ReportMetric(float64(m.TotalAlloc)/1024/1024, "MB-total-alloc")
}
