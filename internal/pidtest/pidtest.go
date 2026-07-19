// Package pidtest provides common test utilities for testing parts of the QuickPID library.
//
//spellchecker:words pidtest
package pidtest

//spellchecker:words testing github quickpid backend
import (
	"testing"

	"github.com/tkw1536/quickpid/backend"
)

// StoreFactory creates a concrete [backend.Store] for HTTP integration tests.
type StoreFactory = func(t *testing.T) backend.Store
