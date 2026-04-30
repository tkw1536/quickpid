package apitest

import (
	"testing"

	"github.com/tkw1536/quickpid/backend"
)

// BackendFactory is a function that creates a concrete new Resolver.
// It should call t.Fatal the test if it cannot create a resolver.
// It should use t.Cleanup if cleanup is needed after the test.
type BackendFactory = func(t *testing.T) backend.Backend

// RunResolverHTTPTests starts an httptest server for res and runs a sequential suite of
// subtests against resolver HTTP routes (namespaces, resources, batch, errors).
// Resolver implementations can call this from their tests with their concrete implementation.
func RunResolverHTTPTests(t *testing.T, factory BackendFactory) {
	t.Helper()

	flows, err := loadEmbeddedFlows()
	if err != nil {
		t.Fatal(err)
	}

	for _, flow := range flows {
		t.Run(flow.Name, func(t *testing.T) {
			if flow.Comment != "" {
				t.Log(flow.Comment)
			}
			b := factory(t)
			flow.Run(t, b)
		})
	}
}
