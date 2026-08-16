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

// BackendFactory creates a concrete [backend.Backend] for HTTP integration tests.
type BackendFactory func(t *testing.T, l *slog.Logger) backend.Backend

type BackendResetter func(t *testing.T) error

func (s BackendResetter) onceFunc(t *testing.T) func() {
	t.Helper()

	if s == nil {
		return func() {}
	}
	return sync.OnceFunc(func() {
		s.startTest(t)
	})
}

func (s BackendResetter) startTest(t *testing.T) {
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
