package backend_test

import (
	"testing"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apitest"
)

func TestInMemoryBackend(t *testing.T) {
	apitest.RunResolverHTTPTests(t, func(t *testing.T) backend.Backend {
		return backend.NewInMemoryBackend()
	})
}
