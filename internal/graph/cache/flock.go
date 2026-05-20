package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/flock"
)

// ErrLockTimeout indicates AcquireBuildLock could not obtain the lock within timeout.
var ErrLockTimeout = errors.New("build lock acquire timed out")

// AcquireBuildLock takes an exclusive flock on lockPath, polling every 100ms until
// timeout. Returns a release function the caller must invoke.
func AcquireBuildLock(lockPath string, timeout time.Duration) (release func() error, err error) {
	fl := flock.New(lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	locked, err := fl.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, ErrLockTimeout
		}
		return nil, fmt.Errorf("flock %s: %w", lockPath, err)
	}
	if !locked {
		return nil, ErrLockTimeout
	}
	return fl.Unlock, nil
}
