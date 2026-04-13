package apitest

import (
	"testing"

	"github.com/tkw1536/quickpid/api"
)

// ResolverFactory is a function that creates a concrete new Resolver.
// It should call t.Fatal the test if it cannot create a resolver.
// It should use t.Cleanup if cleanup is needed after the test.
type ResolverFactory = func(t *testing.T) api.Resolver

// RunResolverHTTPTests starts an httptest server for res and runs a sequential suite of
// subtests against resolver HTTP routes (namespaces, resources, batch, errors).
// Resolver implementations can call this from their tests with their concrete implementation.
func RunResolverHTTPTests(t *testing.T, factory ResolverFactory) {
	t.Helper()

	testFlow(t, factory, "flowListNamespaces", flowListNamespaces)
	testFlow(t, factory, "flowCreateNamespace", flowCreateNamespace)
	testFlow(t, factory, "flowListResources", flowListResources)
	testFlow(t, factory, "flowCreateResource", flowCreateResource)
	testFlow(t, factory, "flowBatchCreateResources", flowBatchCreateResources)
	testFlow(t, factory, "flowGetResource", flowGetResource)
	testFlow(t, factory, "flowUpdateResource", flowUpdateResource)
}

func testFlow(t *testing.T, factory ResolverFactory, name string, flow func(t *testing.T, h *harness)) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		h := newHarness(t, factory)
		flow(t, h)
	})
}

// MountPath is the URL prefix used when mounting the handler (matches cmd/quickpid and cmd/quickpid-mem).
const MountPath = "/api/v2"
