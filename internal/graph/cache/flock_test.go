package cache

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestBuildLockSerializes(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "build.lock")

	var (
		mu        sync.Mutex
		insideMax int
		inside    int
		wg        sync.WaitGroup
	)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rel, err := AcquireBuildLock(lockPath, 5*time.Second)
			if err != nil {
				t.Errorf("acquire: %v", err)
				return
			}
			defer func() { _ = rel() }()
			mu.Lock()
			inside++
			if inside > insideMax {
				insideMax = inside
			}
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
			mu.Lock()
			inside--
			mu.Unlock()
		}()
	}
	wg.Wait()
	if insideMax != 1 {
		t.Errorf("max concurrent holders=%d, want 1", insideMax)
	}
}

func TestBuildLockTimeout(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "build.lock")

	rel1, err := AcquireBuildLock(lockPath, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rel1() }()

	_, err = AcquireBuildLock(lockPath, 100*time.Millisecond)
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("err=%v, want ErrLockTimeout", err)
	}
}
