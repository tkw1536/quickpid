// Package pidtest provides common test utilities for testing parts of the QuickPID library.
//
//spellchecker:words pidtest
package pidtest

//spellchecker:words slog sync testing github quickpid backend
import (
	"log/slog"
	"sync"
	"testing"

	"github.com/tkw1536/quickpid/backend"
)

// StoreFactory creates a concrete [backend.Store] for HTTP integration tests.
type StoreFactory func(t *testing.T, l *slog.Logger) backend.Store

type StoreResetter func(t *testing.T) error

func (s StoreResetter) onceFunc(t *testing.T) func() {
	t.Helper()

	if s == nil {
		return func() {}
	}
	return sync.OnceFunc(func() {
		s.startTest(t)
	})
}

func (s StoreResetter) startTest(t *testing.T) {
	t.Helper()

	// if the resetter is nil, then we can run the test in parallel
	if s == nil {
		t.Parallel()
		return
	}

	// if not, we need to run the test serially
	if err := s(t); err != nil {
		t.Fatalf("reset failed: %s", err)
	}
}
