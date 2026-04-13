package apitest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/server"
)

// ResolverFactory is a function that creates a concrete new Resolver.
// It should call t.Fatal the test if it cannot create a resolver.
// It should use t.Cleanup if cleanup is needed after the test.
type ResolverFactory = func(t *testing.T) api.Resolver

// RunResolverHTTPTests starts an httptest server for res and runs a sequential suite of
// subtests against every resolver HTTP route (namespaces, resources, batch, errors).
// Resolver implementations can call this from their tests with their concrete implementation.
// pidGen is only passed to the HTTP handler; CreateResource/BatchCreateResources receive it per request.
func RunResolverHTTPTests(t *testing.T, factory ResolverFactory, pidGen func() (string, error)) {
	t.Helper()

	testFlow(t, factory, pidGen, "resolverFlow", resolverFlow)
}

func testFlow(t *testing.T, factory ResolverFactory, pidGen func() (string, error), name string, flow func(t *testing.T, srv *httptest.Server)) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		srv := newServer(t, factory(t), pidGen)
		flow(t, srv)
	})
}

// MountPath is the URL prefix used when mounting the handler (matches cmd/quickpid and cmd/quickpid-mem).
const MountPath = "/api/v2"

// newServer creates a new server for testing.
func newServer(tb testing.TB, res api.Resolver, pidGen func() (string, error)) *httptest.Server {
	tb.Helper()
	mux := http.NewServeMux()
	apiHandler := server.NewHandler(server.Options{MountPath: MountPath}, res, pidGen)
	mux.Handle(MountPath+"/", http.StripPrefix(MountPath, apiHandler))
	mux.Handle("GET "+MountPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, MountPath+"/", http.StatusMovedPermanently)
	}))
	srv := httptest.NewServer(mux)
	tb.Cleanup(srv.Close)
	return srv
}
