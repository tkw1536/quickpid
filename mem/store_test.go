package mem_test

import (
	"testing"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/internal/apitest"
	"github.com/tkw1536/quickpid/mem"
)

func TestHTTP_ResolverFlow(t *testing.T) {
	apitest.RunResolverHTTPTests(t, func(t *testing.T) api.Resolver { return mem.NewStore() })
}
