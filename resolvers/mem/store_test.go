package mem_test

import (
	"testing"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/internal/apitest"
	"github.com/tkw1536/quickpid/resolvers/mem"
	"github.com/tkw1536/quickpid/server"
)

func TestHTTP_ResolverFlow(t *testing.T) {
	apitest.RunResolverHTTPTests(t, func(t *testing.T) backend.ResolverBackend {
		return mem.NewStore(server.DefaultPIDMaxAttempts)
	})
}
