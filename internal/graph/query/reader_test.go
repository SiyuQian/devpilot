package query

import (
	"testing"

	"github.com/siyuqian/devpilot/internal/graph/store"
)

// TestStoreSatisfiesReader is a compile-time assertion that *store.Store implements Reader.
func TestStoreSatisfiesReader(t *testing.T) {
	var _ Reader = (*store.Store)(nil)
}
