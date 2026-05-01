package backend_test

import (
	"testing"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/servertest"
)

func TestInMemoryBackend(t *testing.T) {
	servertest.TestBackend(t, func(t *testing.T) backend.Backend {
		return backend.NewInMemoryBackend()
	})
}
